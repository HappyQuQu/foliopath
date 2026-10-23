#!/usr/bin/env python3
"""Bind native verification and tag publication to one source/version/image plan."""
import argparse
import json
import os
from pathlib import Path
import re
import secrets
import subprocess

IMAGE = "evanqu/foliopath"
ARCHITECTURES = {"amd64", "arm64"}
DIGEST = re.compile(r"sha256:[a-f0-9]{64}")
TAG = re.compile(r"[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}")


def validate_plan(plan):
    if plan.get("image") != IMAGE or not re.fullmatch(r"[a-f0-9]{40}", plan.get("source", "")):
        raise ValueError("Invalid image repository or source commit")
    if not TAG.fullmatch(plan.get("version", "")):
        raise ValueError("Invalid embedded version")
    labels = plan.get("labels", {})
    if (labels.get("org.opencontainers.image.revision") != plan["source"] or
            labels.get("org.opencontainers.image.version") != plan["version"]):
        raise ValueError("OCI labels must match the source and embedded version")
    if any(not isinstance(key, str) or not isinstance(value, str) or
           any(ord(c) < 32 for c in key + value) or "=" in key
           for key, value in labels.items()):
        raise ValueError("Invalid OCI labels")
    tags = plan.get("tags")
    if not isinstance(tags, list) or not tags:
        raise ValueError("Publication requires at least one tag")
    for tag in tags:
        if not isinstance(tag, str) or not tag.startswith(IMAGE + ":") or not TAG.fullmatch(tag[len(IMAGE) + 1:]):
            raise ValueError("Invalid image tag")
    return plan


def read_plan(path):
    return validate_plan(json.loads(Path(path).read_text()))


def write_json(path, data):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n")


def docker_json(arguments, run):
    return json.loads(run(["docker", *arguments], check=True, capture_output=True, text=True).stdout)


def verify(plan, arch, digest, output, run=subprocess.run, environ=None):
    validate_plan(plan)
    environ = os.environ if environ is None else environ
    if arch not in ARCHITECTURES or not DIGEST.fullmatch(digest):
        raise ValueError("Invalid architecture or digest")
    if environ.get("SOURCE") != plan["source"] or environ.get("VERSION") != plan["version"]:
        raise ValueError("Verification inputs differ from the image plan")
    reference = IMAGE + "@" + digest
    run(["docker", "pull", "--platform", "linux/" + arch, reference], check=True)
    image = docker_json(["image", "inspect", reference], run)[0]
    labels = image.get("Config", {}).get("Labels", {})
    if image.get("Architecture") != arch or image.get("Os") != "linux" or any(
            labels.get(key) != plan["labels"][key] for key in
            ["org.opencontainers.image.version", "org.opencontainers.image.revision"]):
        raise ValueError("Built image identity does not match the plan")
    env = dict(environ, FOLIOPATH_RELEASE_IMAGE=reference,
               FOLIOPATH_RELEASE_BUILD_IMAGE="0",
               FOLIOPATH_RELEASE_EXPECTED_VERSION=plan["version"])
    run(["sh", "tests/release/image_smoke.sh"], check=True, env=env)
    write_json(output, dict(image=IMAGE, source=plan["source"], version=plan["version"],
                            arch=arch, digest=digest))


def index_descriptors(reference, arch, run):
    index = docker_json(["buildx", "imagetools", "inspect", "--raw", reference], run)
    descriptors = index.get("manifests", [])
    platforms = {(d.get("platform", {}).get("os"), d.get("platform", {}).get("architecture"))
                 for d in descriptors if d.get("platform", {}).get("os") != "unknown"}
    if platforms != {("linux", arch)}:
        raise ValueError("Native image index has the wrong platform")
    attestations = [d for d in descriptors if
                    d.get("annotations", {}).get("vnd.docker.reference.type") == "attestation-manifest"]
    if not attestations:
        raise ValueError("Native image index is missing attestations")
    result = {d.get("digest") for d in descriptors}
    if not result or any(not isinstance(d, str) or not DIGEST.fullmatch(d) for d in result):
        raise ValueError("Invalid index descriptor digest")
    runnable = {d["digest"] for d in descriptors if d.get("platform", {}).get("os") == "linux"}
    predicates = set()
    for descriptor in attestations:
        if descriptor.get("annotations", {}).get("vnd.docker.reference.digest") not in runnable:
            raise ValueError("Attestation does not refer to the native image")
        manifest = docker_json(["buildx", "imagetools", "inspect", "--raw",
                                IMAGE + "@" + descriptor["digest"]], run)
        predicates.update(layer.get("annotations", {}).get("in-toto.io/predicate-type", "")
                          for layer in manifest.get("layers", []))
    if "https://spdx.dev/Document" not in predicates or not any(
            value.startswith("https://slsa.dev/provenance/") for value in predicates):
        raise ValueError("Both SPDX SBOM and SLSA provenance must be present")
    return result


def publish(plan, records, run=subprocess.run):
    validate_plan(plan)
    root = Path(records)
    if {p.name for p in root.glob("*.json")} != {a + ".json" for a in ARCHITECTURES}:
        raise ValueError("Exactly two verified architecture records are required")
    references = []
    # Validate every record before performing any registry request or tag write.
    for arch in sorted(ARCHITECTURES):
        record = json.loads((root / (arch + ".json")).read_text())
        if any(record.get(key) != value for key, value in dict(
                image=IMAGE, source=plan["source"], version=plan["version"], arch=arch).items()):
            raise ValueError("Verified image record does not match the plan")
        digest = record.get("digest", "")
        if not DIGEST.fullmatch(digest):
            raise ValueError("Invalid verified digest")
        references.append(IMAGE + "@" + digest)
    if len(set(references)) != 2:
        raise ValueError("Native digests must be distinct")
    expected = set()
    for arch, reference in zip(sorted(ARCHITECTURES), references):
        expected.update(index_descriptors(reference, arch, run))
    command = ["docker", "buildx", "imagetools", "create"]
    for tag in dict.fromkeys(plan["tags"]):
        command.extend(["--tag", tag])
    run(command + references, check=True)
    final = docker_json(["buildx", "imagetools", "inspect", "--raw", plan["tags"][0]], run)
    if {d.get("digest") for d in final.get("manifests", [])} != expected:
        raise ValueError("Published index must preserve both images and all attestations")
    print("Published verified native images and attestations without rebuilding")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    plan_parser = sub.add_parser("plan")
    plan_parser.add_argument("--output", required=True)
    labels_parser = sub.add_parser("labels")
    labels_parser.add_argument("--plan", required=True)
    labels_parser.add_argument("--output", required=True)
    verify_parser = sub.add_parser("verify")
    verify_parser.add_argument("--plan", required=True)
    verify_parser.add_argument("--arch", required=True)
    verify_parser.add_argument("--digest", required=True)
    verify_parser.add_argument("--output", required=True)
    publish_parser = sub.add_parser("publish")
    publish_parser.add_argument("--plan", required=True)
    publish_parser.add_argument("--records", required=True)
    args = parser.parse_args()
    if args.command == "plan":
        metadata = json.loads(os.environ["METADATA"])
        plan = validate_plan(dict(image=os.environ["IMAGE_NAME"], source=os.environ["SOURCE"],
                                  version=os.environ["VERSION"], tags=metadata["tags"],
                                  labels=metadata["labels"]))
        write_json(args.output, plan)
    elif args.command == "labels":
        plan = read_plan(args.plan)
        boundary = secrets.token_hex(16)
        with open(args.output, "a") as output:
            output.write("labels<<" + boundary + "\n")
            output.write("\n".join(k + "=" + v for k, v in plan["labels"].items()) + "\n" + boundary + "\n")
    elif args.command == "verify":
        verify(read_plan(args.plan), args.arch, args.digest, args.output)
    else:
        publish(read_plan(args.plan), args.records)


if __name__ == "__main__":
    main()
