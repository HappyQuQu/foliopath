# FolioPath

<p align="center">
  <strong>简体中文</strong> · <a href="README.md">English</a> · <a href="CHANGELOG.md">更新日志</a>
</p>

<p align="center">
  <img src="web/public/foliopath-mark-tree.svg" alt="FolioPath 标志" width="96">
</p>

<p align="center">
  <strong>📚 为套图浏览而生的只读、自托管媒体浏览器</strong>
</p>

<p align="center">
  不导入，不重组。沿着现有文件夹，自由浏览整套照片和视频。
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: AGPL-3.0-or-later" src="https://img.shields.io/badge/license-AGPL--3.0--or--later-blue.svg"></a>
  <a href="https://hub.docker.com/r/evanqu/foliopath"><img alt="Docker Hub" src="https://img.shields.io/badge/docker-evanqu%2Ffoliopath-2496ED?logo=docker&logoColor=white"></a>
  <img alt="Platforms: Linux amd64 and arm64" src="https://img.shields.io/badge/platform-linux%2Famd64%20%7C%20linux%2Farm64-lightgrey.svg">
</p>

FolioPath 适合已经按**作品、旅行、活动、人物或日期**整理好文件夹的人。它直接读取原有
目录，把一个文件夹及其子目录当作完整套图来浏览，不要求重新上传、建立相册或改变归档方式。

## 🖼️ 界面预览

### 🏠 主页

![FolioPath 浅色主页](docs/screenshots/home-light.webp)

### 🔍 搜索

![FolioPath 搜索页面](docs/screenshots/search.webp)

### ⚙️ 管理中心

![FolioPath 管理中心](docs/screenshots/admin-center.webp)

### 🌙 深色模式

![FolioPath 深色主页](docs/screenshots/home-dark.webp)

## ✨ 套图浏览，更顺手

- **穿透子目录**：一次浏览整套内容，不必逐层点开文件夹。
- **保留来源路径**：汇总浏览时仍知道每张图片来自哪里。
- **固定预览**：看着一张继续挑下一张，方便筛选和比较相似镜头。
- **网格与瀑布流**：适应不同尺寸、横竖构图和大规模媒体库。
- **搜索与筛选**：按文件名、路径、类型、日期和范围快速定位。
- **完整查看器**：支持缩放、平移、1:1、前后切换、全屏和视频播放。

## 🛡️ 原文件始终是原文件

- `/library` 只读：FolioPath 不移动、不重命名、不编辑、不删除媒体。
- 文件系统是事实来源；索引、缩略图和视频封面都可重建。
- 支持多个互不重叠的媒体库，并保留原有目录层级和空目录。
- 媒体库离线或扫描中断时保留最后可靠索引，不会误判为空库。
- 单容器、单进程、SQLite，适合 NAS、家庭服务器和个人媒体归档。
- 可直接设置后台任务及原图读取的并发数，并由服务端安全上限约束。

## 🎞️ 支持格式

| 类型 | 格式 |
| --- | --- |
| 图片 | JPEG（`.jpg`、`.jpeg`）、PNG（`.png`）、WebP（`.webp`）、GIF（`.gif`） |
| 视频 | MP4（`.mp4`）、MOV（`.mov`）、MKV（`.mkv`）、AVI（`.avi`） |

视频不会转码，能否直接播放取决于浏览器支持的内部编码。不兼容内容仍会保留封面和文件信息。
SVG、HEIC/HEIF、AVIF 和 RAW 暂不支持。

## 🚀 快速开始

需要 Linux `amd64` 或 `arm64`、Docker 和 Compose v2。

新建一个空目录，把下面内容保存为 `compose.yaml`，并将 `/mnt/photos` 改为你的媒体目录：

```yaml
services:
  foliopath:
    image: evanqu/foliopath:latest
    restart: unless-stopped
    environment:
      TZ: Asia/Shanghai
    ports:
      - "8080:8080"
    volumes:
      - /mnt/photos:/library:ro
      - ./data:/app/data
```

直接启动：

```bash
docker compose up -d
```

### 参数说明

| 配置项 | 如何修改 |
| --- | --- |
| `image` | `latest` 最省事；需要可控升级时改为明确版本标签或 digest。 |
| `restart` | `unless-stopped` 会在异常退出或宿主机重启后自动恢复，手动停止时不会强行拉起。 |
| `TZ` | 设置时区，例如 `Asia/Shanghai`。 |
| `ports` | 修改左侧的 `8080` 可更换宿主机端口，例如 `"9000:8080"`。 |
| `/mnt/photos:/library:ro` | 将 `/mnt/photos` 改为宿主机唯一媒体根目录；保留 `/library:ro` 不变。 |
| `./data:/app/data` | Docker 会自动创建这个目录，用于保存数据库、设置、任务和缓存；升级前请备份。 |

打开 `http://<服务器局域网地址>:8080`，创建管理员，然后在“管理中心 → 媒体库”中选择
要浏览的目录。

> 建议仅在受信局域网中直接使用。公网访问请在外部配置 HTTPS 和访问控制。

## 🧭 它适合你吗？

**适合：** 文件夹套图、摄影归档、家庭影像、NAS 媒体库，以及不希望软件接管原文件的人。

**不适合：** 照片备份与手机同步、多人相册、AI 人脸识别、图片编辑或视频转码。

## 📖 更多文档

- [高级部署、`.env` 覆盖、升级、备份与反向代理](docs/deployment.md)
- [安全与文件访问边界](docs/security.md)
- [项目文档与开发入口](docs/README.md)
- [系统架构](docs/architecture/README.md)

## 许可证

[GNU Affero General Public License v3.0 or later](LICENSE)（`AGPL-3.0-or-later`）

---

<p align="center">
  <strong>Your folders, beautifully browsed.</strong>
</p>
