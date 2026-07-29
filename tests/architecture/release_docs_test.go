package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseDocumentationMatchesCandidateBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		return string(content)
	}

	readme := read("README.md")
	deployment := read("docs/deployment.md")
	compose := read("compose.yaml")
	environment := read(".env.example")
	dockerfile := read("Dockerfile")
	formats := read("internal/media/formats.go")

	requireFragments(t, "README.md", readme, []string{
		"不要把当前 candidate",
		"docker build --build-arg VERSION=stage5-local -t foliopath:stage5-local .",
		"## Supported Media",
		"视频不会转码",
		"SVG、HEIC/HEIF、AVIF 和 RAW",
		"1 Critical / 8 High",
	})
	requireFragments(t, "docs/deployment.md", deployment, []string{
		"# FolioPath 候选部署与运维",
		"## 备份与恢复",
		"docker compose stop foliopath",
		"不要在运行时只复制 `foliopath.db`",
		"回滚”必须同时恢复升级前数据备份",
		"## 媒体格式与播放边界",
		"## 当前候选已知限制",
		"WebKit 不等同于 Safari 真机",
		"1 Critical / 8 High",
	})
	requireFragments(t, "compose.yaml", compose, []string{
		"${FOLIOPATH_IMAGE:?set FOLIOPATH_IMAGE to an immutable version tag or digest}",
		`FOLIOPATH_LISTEN: "0.0.0.0:8080"`,
		`- "${FOLIOPATH_BIND_ADDRESS:-0.0.0.0}:${FOLIOPATH_PORT:-8080}:8080"`,
		"target: /library",
		"read_only: true",
		"target: /app/data",
		"no-new-privileges:true",
		"- ALL",
	})
	requireFragments(t, ".env.example", environment, []string{
		"FOLIOPATH_IMAGE=ghcr.io/HappyQuQu/foliopath:VERSION",
		"FOLIOPATH_LIBRARY_PATH=/mnt/photos",
		"FOLIOPATH_DATA_PATH=./foliopath-data",
		"FOLIOPATH_BIND_ADDRESS=0.0.0.0",
		"FOLIOPATH_PORT=8080",
		"TZ=UTC",
	})
	requireFragments(t, "Dockerfile", dockerfile, []string{
		`org.opencontainers.image.description="Stage 5 release candidate; not a stable release"`,
		"ADD --checksum=sha256:d114d7c132ec5b45f116d654e17bb4af84561e3041183cd4bfd79abfb85cf724",
		"-Dauto_features=disabled",
		"-Djpeg=enabled",
		"-Dpng=enabled",
		"-Dwebp=enabled",
		"-Dnsgif=true",
		"foliopath-libvips_8.16.1-1.deb",
		"/opt/vips/share/licenses/libvips/LICENSE",
		"ADD --checksum=sha256:e3963a50831c985933e1a625ed566ec4c7adb5c012c34fa9f84438e1d61bdacc",
		"--disable-network",
		"--enable-zlib",
		"--enable-demuxer=mov,matroska",
		"--enable-encoder=libwebp",
		"--enable-encoder=libwebp,png",
		"--enable-decoder=h264,hevc,mpeg4,mjpeg,webp,png",
		"--enable-demuxer=mov,matroska,image2",
		"--enable-filter=scale,setsar,xstack",
		"foliopath-ffmpeg_7.1.5-2.deb",
		"/opt/ffmpeg/share/licenses/ffmpeg/LICENSE.md",
		"AS runtime-assemble",
		"gcr.io/distroless/cc-debian13@sha256:d97bc0a941b8d4be647dc0ee75b264ddbb772f1ac5ba690a4309c00723b23775",
		"/rootfs/var/lib/dpkg/status.d",
		"COPY --from=runtime-assemble /rootfs/ /",
		`CMD ["/app/foliopath", "healthcheck"]`,
		"USER 65532:65532",
		`ENTRYPOINT ["/app/foliopath"]`,
	})
	if strings.Contains(dockerfile, "ca-certificates curl") ||
		strings.Contains(dockerfile, `CMD ["curl"`) {
		t.Error("Dockerfile retains the removed production curl healthcheck closure")
	}

	for _, extension := range []string{
		`".jpg"`, `".jpeg"`, `".png"`, `".webp"`,
		`".gif"`, `".mp4"`, `".mov"`, `".mkv"`,
	} {
		if !strings.Contains(formats, extension) {
			t.Fatalf("canonical media format source is missing %s", extension)
		}
		documented := "`" + strings.Trim(extension, `"`) + "`"
		if !strings.Contains(readme, documented) {
			t.Errorf("README.md is missing canonical media extension %s", documented)
		}
		if !strings.Contains(deployment, documented) {
			t.Errorf("docs/deployment.md is missing canonical media extension %s", documented)
		}
	}

	for _, stale := range []string{
		"当前还没有 React 产品前端",
		"当前还没有 React 产品前端、正式发布 Dockerfile",
		"具体文件名在实现迁移前可调整",
	} {
		if strings.Contains(readme, stale) || strings.Contains(deployment, stale) {
			t.Errorf("release documentation retains stale implementation claim %q", stale)
		}
	}
}

func requireFragments(
	t *testing.T,
	filename string,
	content string,
	fragments []string,
) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(content, fragment) {
			t.Errorf("%s is missing release documentation fragment %q", filename, fragment)
		}
	}
}
