#!/usr/bin/env bash
# An explicit operator-run experiment download, never used by the application.
set -euo pipefail

target=${1:-/Volumes/extended/foliopath/model-experiments/joytag}
if [[ "$target" != /Volumes/extended/foliopath/model-experiments/joytag ]]; then
  echo "Only the dedicated experiment directory is allowed." >&2
  exit 2
fi
for parent in /Volumes/extended /Volumes/extended/foliopath /Volumes/extended/foliopath/model-experiments "$target"; do
  [[ ! -L "$parent" ]] || { echo "Symlink destination rejected." >&2; exit 2; }
done
available_kib=$(df -Pk /Volumes/extended/foliopath | awk 'NR == 2 {print $4}')
(( available_kib >= 1048576 )) || { echo "At least 1 GiB free space required." >&2; exit 2; }
mkdir -p "$target"
revision=6b7f16331a6ccf0fdce37d5a9564715f6e772b22

fetch_verified() {
  local name=$1 expected=$2 expected_bytes=$3 blob=${4:-}
  local destination="$target/$name" source actual bytes
  if [[ -e "$destination" || -L "$destination" ]]; then
    [[ -f "$destination" && ! -L "$destination" ]] || { echo "Invalid existing file: $name" >&2; exit 2; }
    source=$destination
  else
    source=$(mktemp "$target/.${name}.download.XXXXXX")
    # No ambient proxy, plain HTTP, remote Python execution, or silent fallback.
    curl --noproxy '*' --proto '=https' --proto-redir '=https' --tlsv1.2 \
      --fail --location --show-error --connect-timeout 20 --max-time 1200 \
      --retry 2 --output "$source" \
      "https://huggingface.co/fancyfeast/joytag/resolve/$revision/$name"
  fi
  actual=$(shasum -a 256 "$source" | awk '{print $1}')
  bytes=$(wc -c < "$source" | tr -d ' ')
  [[ "$actual" == "$expected" && "$bytes" == "$expected_bytes" ]] || {
    echo "Hash/size mismatch for $name; left untrusted file at $source, not activated." >&2
    exit 1
  }
  if [[ -n "$blob" ]]; then
    [[ "$(git hash-object "$source")" == "$blob" ]] || { echo "Git object mismatch: $name" >&2; exit 1; }
  fi
  if [[ "$source" != "$destination" ]]; then mv -n "$source" "$destination"; fi
  echo "$name $bytes bytes SHA256=$actual verified"
}

fetch_verified config.json b53ee29a8f0d6353331f455fd640c70ef45e5bd979b656f7b9983cffa3724562 330 ce0dab109bc5f9b55494d29667fa1e421f42e530
fetch_verified top_tags.txt 32b1963a234af848643b2bbf47d8eff1f1c7889406810c57b980f41b2b9e01d0 76752 a3131d7ccefe5f2b15cf49f2cc15eb1a9c6b110a
fetch_verified model.onnx f85b7130e6e549b5b0822537007b7482e8c4c8e754c8d9a5bee08e27050e1097 366116154
