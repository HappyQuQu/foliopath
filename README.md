# FolioPath

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

## 🎞️ 支持格式

| 类型 | 格式 |
| --- | --- |
| 图片 | JPEG、PNG、WebP、GIF |
| 视频 | MP4、MOV、MKV、AVI |

视频不转码，能否直接播放取决于浏览器支持的内部编码。不兼容内容仍会保留封面和文件信息。
SVG、HEIC/HEIF、AVIF 和 RAW 暂不支持。

## 🚀 快速开始

需要 Linux `amd64` 或 `arm64`、Docker 和 Compose v2。

```bash
mkdir -p foliopath/foliopath-data
cd foliopath

curl -fsSLO https://raw.githubusercontent.com/HappyQuQu/foliopath/main/compose.yaml
curl -fsSL https://raw.githubusercontent.com/HappyQuQu/foliopath/main/.env.example -o .env

sudo chown -R 65532:65532 foliopath-data
chmod 750 foliopath-data
```

编辑 `.env`，至少确认这三项：

```dotenv
FOLIOPATH_LIBRARY_PATH=/mnt/photos
FOLIOPATH_DATA_PATH=./foliopath-data
FOLIOPATH_PORT=8080
```

启动：

```bash
docker compose up -d
```

打开 `http://<服务器局域网地址>:8080`，创建管理员，然后在“管理中心 → 媒体库”中选择
要浏览的目录。

> 建议仅在受信局域网中直接使用。公网访问请在外部配置 HTTPS 和访问控制。

## 🧭 它适合你吗？

**适合：** 文件夹套图、摄影归档、家庭影像、NAS 媒体库，以及不希望软件接管原文件的人。

**不适合：** 照片备份与手机同步、多人相册、AI 人脸识别、图片编辑或视频转码。

## 📖 更多文档

- [部署、升级、备份与反向代理](docs/deployment.md)
- [安全与文件访问边界](docs/security.md)
- [项目文档与开发入口](docs/README.md)
- [系统架构](docs/architecture/README.md)

## 许可证

[GNU Affero General Public License v3.0 or later](LICENSE)（`AGPL-3.0-or-later`）

---

<p align="center">
  <strong>Your folders, beautifully browsed.</strong>
</p>
