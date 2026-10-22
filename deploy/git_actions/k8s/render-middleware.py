#!/usr/bin/env python3
"""Roll observability workloads when their provisioned configuration changes."""

import hashlib
from pathlib import Path
import sys

import yaml


def main():
    path = Path(sys.argv[1])
    documents = list(yaml.safe_load_all(path.read_text()))
    configs = {
        doc["metadata"]["name"]: doc["data"]
        for doc in documents if doc and doc["kind"] == "ConfigMap"
    }
    for doc in documents:
        if not doc or doc["kind"] not in ("StatefulSet", "Deployment"):
            continue
        name = doc["metadata"]["name"]
        config_name = {"loki": "loki-config", "tempo": "tempo-config", "grafana": "grafana-datasources"}.get(name)
        if config_name:
            checksum = hashlib.sha256(yaml.safe_dump(configs[config_name], sort_keys=True).encode()).hexdigest()
            doc["spec"]["template"].setdefault("metadata", {}).setdefault("annotations", {})["checksum/config"] = checksum
    path.write_text(yaml.safe_dump_all(documents, sort_keys=False))


if __name__ == "__main__":
    main()
