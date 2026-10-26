#!/usr/bin/env python3
"""Bind native verification and tag publication to one source/version/image plan."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import subprocess
import tarfile

IMAGE = "evanqu/foliopath"
IMAGES = {"main": IMAGE, "ai": IMAGE}
RECIPES = {"main": "Dockerfile", "ai": "services/inference/Dockerfile.release"}
ARCHITECTURES = {"amd64", "arm64"}
DIGEST = re.compile(r"sha256:[a-f0-9]{64}")
TAG = re.compile(r"[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}")


def validate_plan(plan):
    kind = plan.get("component", "main")
    if kind not in IMAGES or plan.get("image") != IMAGES[kind] or not re.fullmatch(r"[a-f0-9]{40}", plan.get("source", "")):
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
    if "verification_only" in plan and type(plan["verification_only"]) is not bool:
        raise ValueError("Verification-only mode must be boolean")
    if plan.get("verification_only") and labels.get("org.foliopath.component") != kind:
        raise ValueError("Component label must match the verification plan")
    tags = plan.get("tags")
    if plan.get("verification_only") is True:
        if tags != []:
            raise ValueError("Verification-only plans must not contain tags")
    elif not isinstance(tags, list) or not tags or kind != "main":
        raise ValueError("Publication requires a main-image plan and at least one tag")
    for tag in tags:
        if not isinstance(tag, str) or not tag.startswith(plan["image"] + ":") or not TAG.fullmatch(tag[len(plan["image"]) + 1:]):
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


def verify(plan, arch, digest, output, run=subprocess.run, environ=None, archive=None):
    validate_plan(plan)
    environ = os.environ if environ is None else environ
    if arch not in ARCHITECTURES or not DIGEST.fullmatch(digest):
        raise ValueError("Invalid architecture or digest")
    if environ.get("SOURCE") != plan["source"] or environ.get("VERSION") != plan["version"]:
        raise ValueError("Verification inputs differ from the image plan")
    reference = plan["image"] + "@" + digest
    run(["docker", "pull", "--platform", "linux/" + arch, reference], check=True)
    image = docker_json(["image", "inspect", reference], run)[0]
    labels = image.get("Config", {}).get("Labels", {})
    expected_labels = ["org.opencontainers.image.version", "org.opencontainers.image.revision"]
    if plan.get("verification_only"):
        expected_labels.append("org.foliopath.component")
    if image.get("Architecture") != arch or image.get("Os") != "linux" or any(
            labels.get(key) != plan["labels"][key] for key in
            expected_labels):
        raise ValueError("Built image identity does not match the plan")
    if plan.get("component", "main") == "ai":
        run(["python3", "tests/release/inference_existing_image.py", "--image", reference,
             "--output", str(Path(output).parent / "ai-runtime")], check=True)
    else:
        env = dict(environ, FOLIOPATH_RELEASE_IMAGE=reference,
                   FOLIOPATH_RELEASE_BUILD_IMAGE="0",
                   FOLIOPATH_RELEASE_EXPECTED_VERSION=plan["version"])
        run(["sh", "tests/release/image_smoke.sh"], check=True, env=env)
    record = dict(image=plan["image"], source=plan["source"], version=plan["version"], arch=arch, digest=digest)
    if plan.get("verification_only"):
        index_descriptors(reference, arch, run, plan["image"])
        record["oci"] = verify_oci_archive(archive, reference, arch, plan, run)
        evidence = Path(output).parent / "attestations"
        for name, field in (("sbom", "SBOM"), ("provenance", "Provenance")):
            value = docker_json(["buildx", "imagetools", "inspect", reference, "--format", "{{json ." + field + "}}"], run)
            if not value:
                raise ValueError("Missing exact-image " + name)
            write_json(evidence / (name + ".json"), value)
        record.update(component=plan["component"], verification_only=True, tags=[], attestations_verified=True,
                      checks=["canonical-runtime41", "contracts44", "ocr-identity"] if plan["component"] == "ai" else ["image-smoke"],
                      model_inference="not_run_cache_not_provisioned" if plan["component"] == "ai" else "not_applicable")
    write_json(output, record)


def index_descriptors(reference, arch, run, repository=IMAGE):
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
                                repository + "@" + descriptor["digest"]], run)
        predicates.update(layer.get("annotations", {}).get("in-toto.io/predicate-type", "")
                          for layer in manifest.get("layers", []))
    if "https://spdx.dev/Document" not in predicates or not any(
            value.startswith("https://slsa.dev/provenance/") for value in predicates):
        raise ValueError("Both SPDX SBOM and SLSA provenance must be present")
    return result



def verify_oci_archive(archive, reference, arch, plan, run=subprocess.run):
    if archive is None:
        raise ValueError("Verification requires the same-build OCI archive")
    archive = Path(archive)
    registry = docker_json(["buildx", "imagetools", "inspect", "--raw", reference], run)
    descriptor = next(d for d in registry["manifests"]
                      if d.get("platform", {}).get("architecture") == arch and d.get("platform", {}).get("os") == "linux")
    remote = docker_json(["buildx", "imagetools", "inspect", "--raw", plan["image"] + "@" + descriptor["digest"]], run)
    with tarfile.open(archive) as package:
        index = json.load(package.extractfile("index.json"))
        manifests, checked = [], set()

        def read_blob(item):
            digest = item["digest"]
            if not DIGEST.fullmatch(digest):
                raise ValueError("Invalid OCI digest")
            member = package.getmember("blobs/sha256/" + digest.split(":")[1])
            if member.size != item["size"]:
                raise ValueError("OCI blob size differs")
            value = package.extractfile(member).read()
            if "sha256:" + hashlib.sha256(value).hexdigest() != digest:
                raise ValueError("OCI blob hash differs")
            checked.add(digest)
            return value

        def visit(item):
            value = json.loads(read_blob(item))
            if "manifests" in value:
                for child in value["manifests"]:
                    visit(child)
            elif "config" in value:
                config = json.loads(read_blob(value["config"]))
                for layer in value["layers"]:
                    read_blob(layer)
                manifests.append((item, value, config))

        for item in index["manifests"]:
            visit(item)
        runtime = [(item, value, config) for item, value, config in manifests
                   if config.get("architecture") == arch and config.get("os") == "linux"]
        if len(runtime) != 1:
            raise ValueError("OCI archive must contain one native runtime")
        item, value, config = runtime[0]
        if value["config"] != remote["config"] or value["layers"] != remote["layers"]:
            raise ValueError("OCI runtime differs from the registry-verified image")
        if any(config.get("config", {}).get("Labels", {}).get(key) != expected for key, expected in plan["labels"].items()):
            raise ValueError("OCI runtime labels differ from its plan")
        predicates = {layer.get("annotations", {}).get("in-toto.io/predicate-type")
                      for d, value, _ in manifests
                      if d.get("annotations", {}).get("vnd.docker.reference.digest") == item["digest"]
                      for layer in value["layers"]}
        if "https://spdx.dev/Document" not in predicates or not any(p and p.startswith("https://slsa.dev/provenance/") for p in predicates):
            raise ValueError("OCI archive lost SPDX or SLSA attestations")
    digest = hashlib.sha256()
    with archive.open("rb") as stream:
        for chunk in iter(lambda: stream.read(4 * 1024 * 1024), b""):
            digest.update(chunk)
    sha = digest.hexdigest()
    return dict(archive=archive.name, archive_sha256=sha, bytes=archive.stat().st_size,
                runtime_manifest=item["digest"], registry_runtime_manifest=descriptor["digest"],
                config_digest=value["config"]["digest"], runtime_and_layers_equal=True,
                checked_blobs=len(checked), separate_export_index_not_assumed_equal=True)


def publish(plan, records, run=subprocess.run):
    validate_plan(plan)
    if plan.get("verification_only"):
        raise ValueError("Verification-only results cannot publish tags")
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
        if record.get("verification_only"):
            raise ValueError("Verification-only records cannot publish tags")
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
    verification_parser = sub.add_parser("verification-plan")
    verification_parser.add_argument("--kind", choices=sorted(IMAGES), required=True)
    verification_parser.add_argument("--output", required=True)
    labels_parser = sub.add_parser("labels")
    labels_parser.add_argument("--plan", required=True)
    labels_parser.add_argument("--output", required=True)
    verify_parser = sub.add_parser("verify")
    verify_parser.add_argument("--plan", required=True)
    verify_parser.add_argument("--arch", required=True)
    verify_parser.add_argument("--digest", required=True)
    verify_parser.add_argument("--output", required=True)
    verify_parser.add_argument("--oci-archive")
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
    elif args.command == "verification-plan":
        source, version = os.environ["SOURCE"], os.environ["VERSION"]
        plan = validate_plan(dict(component=args.kind, verification_only=True, image=IMAGES[args.kind],
                                  source=source, version=version, tags=[], labels={
                                      "org.opencontainers.image.revision": source,
                                      "org.opencontainers.image.version": version,
                                      "org.foliopath.component": args.kind}))
        write_json(args.output, plan)
    elif args.command == "labels":
        plan = read_plan(args.plan)
        boundary = secrets.token_hex(16)
        with open(args.output, "a") as output:
            output.write("image=" + plan["image"] + "\n")
            output.write("dockerfile=" + RECIPES[plan.get("component", "main")] + "\n")
            output.write("labels<<" + boundary + "\n")
            output.write("\n".join(k + "=" + v for k, v in plan["labels"].items()) + "\n" + boundary + "\n")
    elif args.command == "verify":
        verify(read_plan(args.plan), args.arch, args.digest, args.output, archive=args.oci_archive)
    else:
        publish(read_plan(args.plan), args.records)


if __name__ == "__main__":
    main()
