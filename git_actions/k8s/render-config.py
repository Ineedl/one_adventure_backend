#!/usr/bin/env python3
"""Copy service configuration and wire MySQL/Redis to Kubernetes Services."""

import hashlib
import os
from pathlib import Path
import re
import shutil
import sys

import yaml


def main():
    source, output = map(Path, sys.argv[1:3])
    mysql_password = os.environ["MYSQL_ROOT_PASSWORD"]
    redis_password = os.environ["REDIS_PASSWORD"]
    os.umask(0o077)
    (output / "mysql-root-password").write_text(mysql_password)
    (output / "redis-password").write_text(redis_password)
    for service in sys.argv[3:]:
        destination = output / "config" / service
        shutil.copytree(
            source / service.replace("-", "_") / "manifest/config-docker",
            destination,
        )
        config_path = destination / "config.yaml"
        config = yaml.safe_load(config_path.read_text())
        for database in (config.get("database") or {}).values():
            link = database.get("link", "")
            if not link.startswith("mysql:"):
                continue
            # Keep the database name and any query parameters from the source.
            link, count = re.subn(
                r"^mysql:.*?@tcp\([^)]*\)/",
                lambda _: f"mysql:root:{mysql_password}@tcp(mysql:3306)/",
                link,
                count=1,
            )
            if count != 1:
                raise ValueError(f"Unsupported MySQL link format in {service}")
            database["link"] = link
        for redis in (config.get("redis") or {}).values():
            redis["address"] = "redis:6379"
            redis["pass"] = redis_password
        config_path.write_text(yaml.safe_dump(config, allow_unicode=True, sort_keys=False))
        digest = hashlib.sha256()
        for filename in sorted(destination.rglob("*")):
            if filename.is_file():
                digest.update(str(filename.relative_to(destination)).encode())
                digest.update(b"\0")
                digest.update(filename.read_bytes())
                digest.update(b"\0")
        (output / f"{service}-config.sha256").write_text(digest.hexdigest())


if __name__ == "__main__":
    main()
