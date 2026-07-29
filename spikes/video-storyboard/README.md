# VSP-002 video storyboard spike

This isolated, standard-library-only Go module generates synthetic videos in a
temporary directory. It never reads media from `/library` or another user path.

The spike measures and verifies:

- the frozen 4–10 frame-count rule and midpoint timestamps;
- bounded input-side fast seek extraction at each target timestamp;
- composition into one static WebP sprite;
- sprite dimensions and codec metadata;
- a sequential full-decode `fps + tile` baseline for comparison;
- corrupt-input rejection and temporary-directory cleanup.

Run:

```sh
go run .
```

Optional tool overrides:

```sh
go run . -ffmpeg /path/to/ffmpeg -ffprobe /path/to/ffprobe
```

The program prints one JSON document containing the environment, fixture
dimensions/durations, output sizes, and elapsed/user/system times. Local results
are development evidence only; they do not replace native linux/amd64 and
linux/arm64 Backend Gate measurements.
