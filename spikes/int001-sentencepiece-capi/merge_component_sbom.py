#!/usr/bin/env python3
"""Deterministically merge explicit native components into a scanner SBOM.

This is an INT-001 evidence utility, not a production packaging tool.  Image
scanners do not discover the manually copied ONNX Runtime and SentencePiece
libraries, so accepting scanner output alone would produce an incomplete bill
of materials.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import pathlib
import re
import sys
import uuid


SHA256 = re.compile(r"^sha256:[0-9a-f]{64}$")


def fail(message: str) -> None:
    raise ValueError(message)


def load_document(path: pathlib.Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict) or value.get("bomFormat") != "CycloneDX":
        fail(f"{path}: expected a CycloneDX object")
    if not isinstance(value.get("components"), list):
        fail(f"{path}: components must be an array")
    return value


def component_ref(component: dict, source: pathlib.Path) -> str:
    ref = component.get("bom-ref")
    if not isinstance(ref, str) or not ref:
        fail(f"{source}: every component requires a non-empty bom-ref")
    return ref


def validate_digest(name: str, value: str) -> None:
    if not SHA256.fullmatch(value):
        fail(f"{name}: expected lowercase sha256:<64 hex>")


def canonicalize(value, parent_key: str = ""):
    """Sort CycloneDX collections whose order has no semantic meaning."""
    if isinstance(value, dict):
        return {key: canonicalize(item, key) for key, item in value.items()}
    if not isinstance(value, list):
        return value

    items = [canonicalize(item) for item in value]
    if parent_key == "components":
        return sorted(items, key=lambda item: item.get("bom-ref", ""))
    if parent_key == "dependencies":
        return sorted(items, key=lambda item: item.get("ref", ""))
    if parent_key == "dependsOn":
        return sorted(set(items))
    if parent_key in {"properties", "tools", "hashes", "externalReferences", "licenses"}:
        return sorted(items, key=lambda item: json.dumps(item, sort_keys=True))
    return items


def flatten_components(components: list, source: pathlib.Path) -> dict[str, dict]:
    """Flatten scanner nesting because its package-to-file ownership is unstable."""
    flattened: dict[str, dict] = {}

    def visit(component: dict) -> None:
        if not isinstance(component, dict):
            fail(f"{source}: every component must be an object")
        item = copy.deepcopy(component)
        children = item.pop("components", [])
        if not isinstance(children, list):
            fail(f"{source}: nested components must be an array")
        ref = component_ref(item, source)
        previous = flattened.get(ref)
        if previous is not None and canonicalize(previous) != canonicalize(item):
            fail(f"{source}: conflicting component bom-ref {ref}")
        flattened[ref] = item
        for child in children:
            visit(child)

    for component in components:
        visit(component)
    return flattened


def consolidate_dependencies(dependencies: list, known_refs: set[str], source: pathlib.Path) -> list:
    consolidated: dict[str, set[str]] = {}
    for dependency in dependencies:
        if not isinstance(dependency, dict) or not isinstance(dependency.get("ref"), str):
            fail(f"{source}: every dependency requires a string ref")
        depends_on = dependency.get("dependsOn", [])
        if not isinstance(depends_on, list) or not all(isinstance(ref, str) for ref in depends_on):
            fail(f"{source}: every dependsOn value must be a string")
        consolidated.setdefault(dependency["ref"], set()).update(depends_on)

    used_refs = set(consolidated)
    for values in consolidated.values():
        used_refs.update(values)
    missing = sorted(used_refs - known_refs)
    if missing:
        fail(f"{source}: dependency graph contains unknown ref {missing[0]}")
    return [
        {"ref": ref, "dependsOn": sorted(consolidated[ref])}
        for ref in sorted(consolidated)
    ]


def merge(
    base_path: pathlib.Path,
    component_paths: list[pathlib.Path],
    architecture: str,
    platform_manifest: str,
    config: str,
    local_index: str,
) -> dict:
    for name, digest in (
        ("platform manifest", platform_manifest),
        ("config", config),
        ("local index", local_index),
    ):
        validate_digest(name, digest)

    result = copy.deepcopy(load_document(base_path))
    metadata = result.get("metadata")
    if not isinstance(metadata, dict) or not isinstance(metadata.get("component"), dict):
        fail(f"{base_path}: metadata.component is required")
    root = metadata["component"]
    root_ref = component_ref(root, base_path)
    root_identity = f"{root_ref} {root.get('purl', '')}"
    if local_index.removeprefix("sha256:") not in root_identity:
        fail(f"{base_path}: root component is not bound to the requested local index")

    merged = flatten_components(result["components"], base_path)

    explicit_refs: list[str] = []
    for path in component_paths:
        document = load_document(path)
        for component in document["components"]:
            ref = component_ref(component, path)
            purl = component.get("purl", "")
            if not isinstance(purl, str) or f"arch={architecture}" not in purl:
                fail(f"{path}: component {ref} does not target arch={architecture}")
            if ref in merged:
                fail(f"{path}: duplicate component bom-ref {ref}")
            merged[ref] = copy.deepcopy(component)
            explicit_refs.append(ref)

    result["$schema"] = "http://cyclonedx.org/schema/bom-1.6.schema.json"
    result["specVersion"] = "1.6"
    result["components"] = [merged[ref] for ref in sorted(merged)]

    properties = metadata.setdefault("properties", [])
    if not isinstance(properties, list):
        fail(f"{base_path}: metadata.properties must be an array")
    properties.extend(
        [
            {"name": "foliopath:int001:architecture", "value": architecture},
            {"name": "foliopath:int001:platform-manifest", "value": platform_manifest},
            {"name": "foliopath:int001:image-config", "value": config},
            {"name": "foliopath:int001:local-index", "value": local_index},
            {
                "name": "foliopath:int001:scope",
                "value": "isolated runtime evidence; not a production or release SBOM",
            },
        ]
    )
    properties.sort(key=lambda item: (item.get("name", ""), item.get("value", "")))

    tools = metadata.setdefault("tools", [])
    if not isinstance(tools, list):
        fail(f"{base_path}: metadata.tools must be an array")
    tools.append({"name": "foliopath-int001-sbom-merge", "version": "1"})
    tools.sort(key=lambda item: (item.get("name", ""), item.get("version", "")))

    dependencies = result.setdefault("dependencies", [])
    if not isinstance(dependencies, list):
        fail(f"{base_path}: dependencies must be an array")
    root_dependency = next((item for item in dependencies if item.get("ref") == root_ref), None)
    if root_dependency is None:
        root_dependency = {"ref": root_ref, "dependsOn": []}
        dependencies.append(root_dependency)
    depends_on = root_dependency.setdefault("dependsOn", [])
    if not isinstance(depends_on, list):
        fail(f"{base_path}: root dependsOn must be an array")
    root_dependency["dependsOn"] = sorted(set(depends_on).union(explicit_refs))
    result["dependencies"] = consolidate_dependencies(
        dependencies, {root_ref, *merged.keys()}, base_path
    )

    result["compositions"] = [{"aggregate": "incomplete", "assemblies": [root_ref]}]
    serial_seed = "\n".join(
        [architecture, platform_manifest, config, local_index, *sorted(merged)]
    )
    result["serialNumber"] = f"urn:uuid:{uuid.uuid5(uuid.NAMESPACE_URL, serial_seed)}"
    result["version"] = 1
    result.setdefault("metadata", {}).pop("timestamp", None)
    return canonicalize(result)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", type=pathlib.Path, required=True)
    parser.add_argument("--component", type=pathlib.Path, action="append", required=True)
    parser.add_argument("--architecture", required=True)
    parser.add_argument("--platform-manifest", required=True)
    parser.add_argument("--config", required=True)
    parser.add_argument("--local-index", required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()

    try:
        document = merge(
            args.base,
            args.component,
            args.architecture,
            args.platform_manifest,
            args.config,
            args.local_index,
        )
        encoded = json.dumps(document, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
        args.output.write_text(encoded, encoding="utf-8")
    except (OSError, json.JSONDecodeError, ValueError) as error:
        print(error, file=sys.stderr)
        return 1
    print(
        json.dumps(
            {
                "components": len(document["components"]),
                "output_sha256": hashlib.sha256(encoded.encode("utf-8")).hexdigest(),
                "serialNumber": document["serialNumber"],
                "status": "incomplete",
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
