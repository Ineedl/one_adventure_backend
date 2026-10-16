#!/usr/bin/env python3
"""Bind pending middleware PVCs to persistent directories on a single node."""

import json
import os
import subprocess

import yaml


def get(*args):
    return json.loads(subprocess.check_output(["kubectl", *args, "-o", "json"], text=True))


def main():
    namespace = os.environ.get("KUBE_NAMESPACE", "one-adventure")
    nodes = get("get", "nodes")["items"]
    node_name = os.environ.get("STORAGE_NODE_NAME")
    if node_name:
        nodes = [node for node in nodes if node["metadata"]["name"] == node_name]
    if len(nodes) != 1:
        raise SystemExit("Local storage requires one node; select a node with STORAGE_NODE_NAME.")
    hostname = nodes[0]["metadata"].get("labels", {}).get("kubernetes.io/hostname")
    if not hostname:
        raise SystemExit("Selected node has no kubernetes.io/hostname label.")
    volumes = []
    for claim_name in ("mysql-data", "redis-data"):
        claim = get("-n", namespace, "get", "pvc", claim_name)
        if claim.get("status", {}).get("phase") == "Bound":
            print(f"{claim_name}: already bound; retaining existing storage.")
            continue
        spec = claim["spec"]
        if spec.get("selector") or spec.get("volumeMode", "Filesystem") != "Filesystem":
            raise SystemExit(f"{claim_name}: custom PVC selector/volumeMode is unsupported.")
        name = f"{namespace}-{claim_name}"
        if spec.get("volumeName") and spec["volumeName"] != name:
            raise SystemExit(f"{claim_name}: already requests another PV; inspect it manually.")
        reference = {"namespace": namespace, "name": claim_name, "uid": claim["metadata"]["uid"]}
        volume = {
            "apiVersion": "v1", "kind": "PersistentVolume",
            "metadata": {"name": name, "labels": {"app.kubernetes.io/part-of": "one-adventure"}},
            "spec": {
                "capacity": spec["resources"]["requests"],
                "accessModes": spec["accessModes"],
                "storageClassName": spec.get("storageClassName", ""),
                "persistentVolumeReclaimPolicy": "Retain",
                "claimRef": reference,
                "hostPath": {"path": f"/var/lib/one-adventure/{namespace}/{claim_name}", "type": "DirectoryOrCreate"},
                "nodeAffinity": {"required": {"nodeSelectorTerms": [{"matchExpressions": [
                    {"key": "kubernetes.io/hostname", "operator": "In", "values": [hostname]}
                ]}]}},
            },
        }
        existing = subprocess.check_output(
            ["kubectl", "get", "pv", name, "--ignore-not-found", "-o", "json"], text=True
        ).strip()
        if existing:
            current = json.loads(existing)
            # Never reassign a retained volume from a deleted claim automatically.
            if current["spec"] != volume["spec"]:
                raise SystemExit(f"PV {name} already exists; inspect ownership and retained data manually.")
            print(f"{name}: matching PV already exists.")
            continue
        volumes.append(volume)
    if volumes:
        subprocess.run(
            ["kubectl", "apply", "-f", "-"],
            input=yaml.safe_dump_all(volumes, sort_keys=False), text=True, check=True,
        )


if __name__ == "__main__":
    main()
