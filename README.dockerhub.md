# FolioPath

<p align="center">
  <strong><a href="#english">English</a></strong> · <a href="#简体中文">简体中文</a>
</p>

<a id="english"></a>

<p align="center">
  <img src="https://raw.githubusercontent.com/HappyQuQu/foliopath/main/web/public/foliopath-mark-tree.svg" alt="FolioPath logo" width="96">
</p>

<p align="center">
  <strong>📚 A read-only, self-hosted media browser built for browsing complete photo and video sets</strong>
</p>

<p align="center">
  No imports. No reorganization. Browse the folders you already have while your originals remain untouched.
</p>

<p align="center">
  <a href="https://github.com/HappyQuQu/foliopath"><img alt="Source code" src="https://img.shields.io/badge/source-GitHub-181717?logo=github"></a>
  <a href="https://github.com/HappyQuQu/foliopath/blob/main/LICENSE"><img alt="License: AGPL-3.0-or-later" src="https://img.shields.io/badge/license-AGPL--3.0--or--later-blue.svg"></a>
  <img alt="Platforms: Linux amd64 and arm64" src="https://img.shields.io/badge/platform-linux%2Famd64%20%7C%20linux%2Farm64-lightgrey.svg">
</p>

FolioPath directly reads an existing directory tree and lets you browse a folder and all its descendants as one set. It is designed for photography archives, family media, NAS libraries, and collections already organized by project, trip, event, person, or date.

![FolioPath home](https://raw.githubusercontent.com/HappyQuQu/foliopath/main/docs/screenshots/home-light.webp)

## Highlights

- Browse a directory and its descendants without creating albums.
- Keep source paths visible in aggregated views.
- Use adaptive grid and masonry layouts for mixed media dimensions.
- Search by filename, path, media type, date, and scope.
- Zoom, pan, view at 1:1, navigate between items, use fullscreen, and play videos.
- Configure multiple non-overlapping libraries below one read-only `/library` mount.
- Preserve the last reliable index when a library is offline or a scan is interrupted.
- Run as one non-root container with SQLite and rebuildable thumbnails and posters.

## Supported media

| Type | Formats |
| --- | --- |
| Images | JPEG (`.jpg`, `.jpeg`), PNG (`.png`), WebP (`.webp`), GIF (`.gif`) |
| Video | MP4 (`.mp4`), MOV (`.mov`), MKV (`.mkv`) |

Videos are not transcoded, so direct playback depends on browser codec support. SVG, HEIC/HEIF, AVIF, RAW, and other unlisted formats are outside the current support contract.

## Quick start

FolioPath requires Linux on `amd64` or `arm64`, Docker, and Docker Compose v2.

Create an empty directory, save the following as `compose.yaml`, and replace `/mnt/photos` with the single host directory containing your media:

```yaml
services:
  foliopath:
    image: evanqu/foliopath:latest
    restart: unless-stopped
    user: "65532:65532"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    stop_grace_period: 15s
    environment:
      TZ: Asia/Shanghai
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=16m,uid=65532,gid=65532,mode=0700
    ports:
      - "8080:8080"
    volumes:
      - /mnt/photos:/library:ro
      - ./data:/app/data
```

Start the container:

```bash
docker compose up -d
```

Open `http://<your-server-LAN-address>:8080`, create the administrator account, then add directories under **Administration → Libraries**.

For controlled upgrades, replace `latest` with a specific version tag or immutable digest. See the repository's [authoritative Compose file](https://github.com/HappyQuQu/foliopath/blob/main/compose.yaml) for configurable path, address, port, image, and timezone values.

## Storage and permissions

| Container path | Access | Purpose |
| --- | --- | --- |
| `/library` | Read-only | The one allowed media root and all selectable library directories below it |
| `/app/data` | Read-write | SQLite database, settings, jobs, thumbnails, posters, and temporary state |

The container runs as UID/GID `65532:65532`. That identity needs read and directory traversal access to `/library`, plus full read/write access to `/app/data`.

Mount the media tree exactly once at `/library`. Nested volumes, bind mounts, or other mount points below `/library` are unsupported and rejected. FolioPath never moves, renames, edits, or deletes original media.

## Network safety

Direct HTTP is intended for an authenticated, trusted LAN. For public or otherwise untrusted network access, use an external HTTPS reverse proxy and prevent clients from bypassing it. Review the [deployment guide](https://github.com/HappyQuQu/foliopath/blob/main/docs/deployment.md) and [security model](https://github.com/HappyQuQu/foliopath/blob/main/docs/security.md) before exposing the service.

## Documentation and support

- [English project overview](https://github.com/HappyQuQu/foliopath/blob/main/README.md)
- [简体中文项目说明](https://github.com/HappyQuQu/foliopath/blob/main/README.zh-CN.md)
- [Deployment, upgrades, backup, restore, and reverse proxy](https://github.com/HappyQuQu/foliopath/blob/main/docs/deployment.md)
- [Documentation index](https://github.com/HappyQuQu/foliopath/blob/main/docs/README.md)
- [Issue tracker](https://github.com/HappyQuQu/foliopath/issues)
- [AGPL-3.0-or-later license](https://github.com/HappyQuQu/foliopath/blob/main/LICENSE)

---

<p align="center"><strong>Your folders, beautifully browsed.</strong></p>

---

<a id="简体中文"></a>

## 简体中文

<p align="center">
  <a href="#english">English</a> · <strong><a href="#简体中文">简体中文</a></strong>
</p>

FolioPath 是一款为整套照片和视频浏览而生的只读、自托管媒体浏览器。它直接读取现有目录树，让你把一个文件夹及其所有子目录作为完整内容集浏览，不需要重新上传、建立相册或改变原有归档方式。

适合摄影归档、家庭影像、NAS 媒体库，以及已经按作品、旅行、活动、人物或日期整理好的收藏。

![FolioPath 主页](https://raw.githubusercontent.com/HappyQuQu/foliopath/main/docs/screenshots/home-light.webp)

### 主要功能

- 穿透浏览文件夹及其所有子目录，无需另建相册。
- 汇总浏览时保留媒体来源路径。
- 使用自适应网格和瀑布流展示不同尺寸的媒体。
- 按文件名、路径、媒体类型、日期和范围搜索。
- 支持缩放、平移、1:1、前后切换、全屏和视频播放。
- 在唯一的只读 `/library` 挂载下配置多个互不重叠的媒体库。
- 媒体库离线或扫描中断时保留最后可靠索引。
- 以非 root 用户运行单容器服务，使用 SQLite，并可重建缩略图和视频封面。

### 支持格式

| 类型 | 格式 |
| --- | --- |
| 图片 | JPEG（`.jpg`、`.jpeg`）、PNG（`.png`）、WebP（`.webp`）、GIF（`.gif`） |
| 视频 | MP4（`.mp4`）、MOV（`.mov`）、MKV（`.mkv`） |

视频不会转码，能否直接播放取决于浏览器支持的内部编码。SVG、HEIC/HEIF、AVIF、RAW 以及其他未列出的格式不在当前支持范围内。

### 快速开始

需要 Linux `amd64` 或 `arm64`、Docker 和 Docker Compose v2。

新建一个空目录，把下面内容保存为 `compose.yaml`，并将 `/mnt/photos` 替换为宿主机上唯一的媒体根目录：

```yaml
services:
  foliopath:
    image: evanqu/foliopath:latest
    restart: unless-stopped
    user: "65532:65532"
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    stop_grace_period: 15s
    environment:
      TZ: Asia/Shanghai
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=16m,uid=65532,gid=65532,mode=0700
    ports:
      - "8080:8080"
    volumes:
      - /mnt/photos:/library:ro
      - ./data:/app/data
```

启动容器：

```bash
docker compose up -d
```

打开 `http://<服务器局域网地址>:8080`，创建管理员账户，然后在 **管理中心 → 媒体库** 中添加要浏览的目录。

需要可控升级时，请把 `latest` 替换为明确版本标签或不可变 digest。路径、监听地址、端口、镜像和时区等可配置项见仓库中的[权威 Compose 文件](https://github.com/HappyQuQu/foliopath/blob/main/compose.yaml)。

### 存储与权限

| 容器路径 | 权限 | 用途 |
| --- | --- | --- |
| `/library` | 只读 | 唯一媒体允许根，以及其下可选择的所有媒体库目录 |
| `/app/data` | 读写 | SQLite 数据库、设置、任务、缩略图、视频封面和临时状态 |

容器使用 UID/GID `65532:65532` 运行。该身份需要读取和遍历 `/library` 的权限，以及完整读写 `/app/data` 的权限。

媒体目录只能整体挂载一次到 `/library`。不支持并会拒绝 `/library` 下的嵌套 volume、bind mount 或其他挂载点。FolioPath 不会移动、重命名、编辑或删除原始媒体。

### 网络安全

直接 HTTP 仅适用于经过认证的受信局域网。若要通过公网或其他不可信网络访问，请使用外部 HTTPS 反向代理，并阻止客户端绕过代理直连应用。暴露服务前请阅读[部署指南](https://github.com/HappyQuQu/foliopath/blob/main/docs/deployment.md)和[安全模型](https://github.com/HappyQuQu/foliopath/blob/main/docs/security.md)。

### 文档与支持

- [简体中文项目说明](https://github.com/HappyQuQu/foliopath/blob/main/README.zh-CN.md)
- [English project overview](https://github.com/HappyQuQu/foliopath/blob/main/README.md)
- [部署、升级、备份、恢复与反向代理](https://github.com/HappyQuQu/foliopath/blob/main/docs/deployment.md)
- [项目文档索引](https://github.com/HappyQuQu/foliopath/blob/main/docs/README.md)
- [问题反馈](https://github.com/HappyQuQu/foliopath/issues)
- [AGPL-3.0-or-later 许可证](https://github.com/HappyQuQu/foliopath/blob/main/LICENSE)

---

<p align="center"><strong>Your folders, beautifully browsed.</strong></p>
