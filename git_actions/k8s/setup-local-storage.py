#!/usr/bin/env python3
"""Bind pending middleware PVCs to persistent directories on a single node."""

import json
import os
import subprocess

import yaml


def matches(actual, expected):
    """Ignore API-server defaults while checking every field we manage."""
    if isinstance(expected, dict):
        return isinstance(actual, dict) and all(
            key in actual and matches(actual[key], value) for key, value in expected.items()
        )
    return actual == expected


def get(*args):
    return json.loads(subprocess.check_output(["kubectl", "--request-timeout=30s", *args, "-o", "json"], text=True))


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
    claim_names = tuple(os.environ.get("STORAGE_CLAIMS", "kafka-data,mysql-data,redis-data").split(","))
    if not claim_names or any(name not in {"kafka-data", "mysql-data", "redis-data"} for name in claim_names):
        raise SystemExit("STORAGE_CLAIMS must be a comma-separated list of known project data PVCs.")
    for claim_name in claim_names:
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
            current_spec = current["spec"]
            managed = dict(volume["spec"])
            del managed["claimRef"]
            # Filesystem is an API default, not a meaningful mismatch.
            if (current.get("metadata", {}).get("labels", {}).get("app.kubernetes.io/part-of") != "one-adventure"
                    or current_spec.get("volumeMode", "Filesystem") != "Filesystem"
                    or not matches(current_spec, managed)):
                raise SystemExit(f"PV {name}: storage settings or project ownership differ; refusing to change existing data storage.")
            old_ref = current_spec.get("claimRef", {})
            if old_ref and (old_ref.get("namespace"), old_ref.get("name")) != (namespace, claim_name):
                raise SystemExit(f"PV {name}: belongs to another claim; refusing reassignment.")
            if current.get("metadata", {}).get("deletionTimestamp"):
                raise SystemExit(f"PV {name}: deletion is in progress; wait for it to finish.")
            if old_ref.get("uid") != reference["uid"]:
                # The PVC with this name has a new UID, so the previous PVC no
                # longer exists. Only reuse this project's identical storage.
                patch = [
                    {"op": "test", "path": "/metadata/resourceVersion", "value": current["metadata"]["resourceVersion"]},
                    {"op": "add", "path": "/spec/claimRef", "value": reference},
                ]
                subprocess.run(["kubectl", "patch", "pv", name, "--type=json", "-p", json.dumps(patch)], check=True)
                print(f"{name}: rebound retained storage to new {claim_name}; disk data preserved.")
            else:
                print(f"{name}: matching PV already exists; API default fields ignored.")
        else:
            volumes.append(volume)
        # Explicit prebinding prevents a default provisioner from competing
        # for this claim. Do not change the immutable storage class of a PVC.
        subprocess.run([
            "kubectl", "-n", namespace, "patch", "pvc", claim_name,
            "--type=merge", "-p", json.dumps({"spec": {"volumeName": name}}),
        ], check=True)
    if volumes:
        subprocess.run(
            ["kubectl", "apply", "-f", "-"],
            input=yaml.safe_dump_all(volumes, sort_keys=False), text=True, check=True,
        )


if __name__ == "__main__":
    main()
