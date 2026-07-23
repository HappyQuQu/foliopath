# FS-03 govips/libvips isolated spike

This nested Go module keeps the native CGO dependency out of the production Go
module until the media capability reaches its implementation stage.

It uses only in-memory synthetic images and verifies:

- JPEG, PNG, and WebP export/load metadata;
- PNG/WebP alpha preservation;
- bounded 48×32 thumbnail output;
- orientation application and normalization;
- four-frame GIF detection and the explicit first-frame thumbnail policy;
- rejection of a truncated PNG;
- a single libvips worker and bounded cache configuration.

Run it on a machine with libvips 8.14 or newer:

```sh
MALLOC_ARENA_MAX=2 timeout 2m go test -count=1 -v ./...
```

The outer timeout proves only that this fixed fixture suite is bounded. It does
not prove per-asset cancellation of an in-process native call. Production media
isolation, pixel/frame limits, ICC fixtures, hostile-image coverage, concurrency,
and retry behavior remain separate Backend/Release Gate requirements.
