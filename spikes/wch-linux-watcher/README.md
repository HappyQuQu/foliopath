# WCH-001 Linux watcher spike

This isolated probe validates Linux directory-level inotify behavior for
`FTR-SCN-001`. It is not imported by the production module and does not access
SQLite, HTTP, or the Web application.

Run on native Linux:

```sh
go run . -watch-directories 10000 -overflow-events 50000
```

Or from the repository root with the pinned build image:

```sh
docker run --rm --platform linux/arm64 \
  --mount type=bind,src="$PWD",dst=/src,readonly \
  --tmpfs /tmp:rw,size=2g \
  -w /src/spikes/wch-linux-watcher \
  -e GOMODCACHE=/tmp/modcache \
  -e GOCACHE=/tmp/buildcache \
  golang:1.26.4-bookworm \
  go run . -watch-directories 10000 -overflow-events 50000
```

The JSON result records the event classes, rename cookie pairing, symlink
reopen rejection, root replacement detection, queue overflow observation,
kernel limits, and the FD/RSS/time cost of 10,000 directory watches.

Limitations:

- The probe reports but never changes host `inotify` sysctls.
- Deterministic `ENOSPC` requires a separately controlled host or namespace.
- `IN_UNMOUNT` requires a privileged isolated mount-namespace test.
- Emulated Linux does not replace native linux/amd64 and linux/arm64 evidence.
