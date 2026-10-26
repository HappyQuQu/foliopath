<p align="center">
  <img src="web/public/foliopath-mark-tree.svg" alt="FolioPath" width="80">
</p>

<h1 align="center">FolioPath</h1>

<p align="center">
  <strong>你的文件夹，就是你的媒体库。</strong><br>
  只读、自托管的照片与视频浏览器，让现有归档更好看，也更好找。
</p>

<p align="center">
  <strong>简体中文</strong> · <a href="README.md">English</a> · <a href="#快速开始">快速开始</a> · <a href="docs/README.md">文档</a>
</p>

<p align="center">
  <a href="LICENSE"><img alt="AGPL-3.0-or-later" src="https://img.shields.io/badge/license-AGPL--3.0--or--later-58a6ff?style=flat-square"></a>
  <a href="https://hub.docker.com/r/evanqu/foliopath"><img alt="Docker Hub" src="https://img.shields.io/badge/Docker-evanqu%2Ffoliopath-2496ED?style=flat-square&amp;logo=docker&amp;logoColor=white"></a>
  <img alt="Linux amd64 / arm64" src="https://img.shields.io/badge/Linux-amd64%20%7C%20arm64-303d53?style=flat-square&amp;logo=linux&amp;logoColor=white">
</p>

![FolioPath 暗色界面：按原有目录浏览照片，无缝瀑布流铺满内容区](docs/screenshots/dark/browse.jpg)

不必重新上传，不必建立相册，也不用改变文件夹结构。FolioPath 直接读取你已经按作品、旅行、活动或日期整理好的目录，在 NAS 和家庭服务器上提供统一的浏览、搜索与整理入口。

> **版本说明**：下方展示当前开发及本地部署界面。智能搜索、OCR、自动标签和人物分析属于可选的 POST-MVP-5 候选功能，尚未正式发布；`latest` 不保证包含这些能力。安装前请核对[候选范围与验证状态](docs/changes/VERIFY-2026-10-26-release-candidate.md)。

## 顺着目录，浏览整套内容

| 基础能力 | 可以做什么 |
| --- | --- |
| 多媒体库与目录树 | 在同一个媒体根目录下添加多个互不重叠的库，保留层级、空目录和媒体数量；一次查看当前目录及全部子目录。 |
| 三种浏览布局 | 自适应网格、瀑布流、无缝瀑布流随时切换；缩略图大小可调，并在当前浏览器记住。 |
| 搜索、筛选与排序 | 按文件名或路径查找，选择媒体类型、修改时间和搜索范围，按名称、修改时间或文件大小排序。 |
| 侧边预览 | 单击查看大图，固定预览后继续浏览列表；可调整侧栏宽度并查看来源目录和基本信息。 |
| 完整查看器 | 适应窗口、缩放、平移、1:1、上一张／下一张、全屏和基本文件信息。 |
| 视频浏览 | 视频封面、4 或 10 帧悬停预览与浏览器原生播放；不转码，无法播放时仍保留封面和信息。 |
| 收藏与手动标签 | 跨目录收藏，添加、选择和移除标签，再按收藏或标签集中查看；整理记录独立于原文件。 |
| 外观与语言 | 浅色／深色／跟随系统，简体中文／English；桌面侧栏与手机抽屉适配不同屏幕。 |

### 从整面照片，到一张照片

**自适应网格**适合核对文件与来源，**无缝瀑布流**适合连续看图。跨目录浏览仍能返回照片所在目录。

![暗色自适应网格：照片卡片、来源目录与可调缩略图](docs/screenshots/dark/grid.jpg)

**侧边预览**让列表保持可见；进入完整查看器后，可以专注细节。

![暗色侧边预览：一边浏览照片，一边查看所选图片与基本信息](docs/screenshots/dark/preview.jpg)

<details>
<summary>展开查看完整图片查看器</summary>

![暗色完整查看器：适应窗口、1:1、缩放与全屏](docs/screenshots/dark/viewer.jpg)

</details>

## 新增：从记得的线索，找到想要的照片

可选智能功能在本地 AI 服务中运行，按媒体库单独开启。媒体和查询不上传到云端；普通浏览与文件名搜索不依赖 AI 服务。

### 融合搜索与图片文字搜索

- **全部来源**：把文件名、库内路径、图片画面与已启用的 OCR 文字匹配合并去重，显示实际匹配来源。
- **画面内容**：用自然语言描述场景；可选多语言模型支持中文与英文查询。视频画面使用独立语义入口。
- **图片文字（OCR）**：识别后可按图片中的中文、英文及多词文本查找，并在预览中查看匹配原文。默认关闭，需先配置 OCR 服务、按库启用并完成处理；仅搜索当前图片版本，不支持视频 OCR。
- **搜索范围**：当前媒体库、当前目录（可包含子目录）或全部媒体库；预览中可查看“为什么匹配”。

![暗色融合搜索：森林查询同时显示文件名和画面内容匹配](docs/screenshots/dark/search.jpg)

### 自动标签与人物整理

| 智能能力 | 当前实现 |
| --- | --- |
| 自动图片标签 | 识别图片内容并直接添加标签，无需逐项审核；可选 153 类图片标签方案，支持中文、英文、双语显示与查找。 |
| 人工整理优先 | 显示 AI／人工标签来源；重新识别保留手动标签，手动移除的标签不会自动加回。切换显示语言无需重新分析。 |
| 图片标签目录 | 按媒体库查看标签与图片数量，搜索标签并集中浏览相关照片；数量包含图片和动图。 |
| 人物与人脸 | 匿名人脸分组、人工命名人物、按人物看照片，逐张核对归类、移出或排除；不识别现实身份。 |
| 人物整理增强（工作区候选） | 单图逐脸确认、受保护的撤销、未确认操作的结果恢复、“不同人物”关系和合并前冲突核对；是否纳入部署以对应变更记录为准。 |

![暗色标签工作区：双语标签、图片数量与海滩照片筛选](docs/screenshots/dark/tags.jpg)

### 功能开关与处理任务分开管理

在“智能分析”中按库开启搜索、人物、自动标签和文字识别，查看各项进度；缺失内容可以单独补齐，全部重建或清理需要明确操作。关闭或取消保留已提交的分析进度，OCR 关闭后暂停文字搜索。

![暗色智能分析：智能搜索、人物分析、自动标签与文字识别四个功能区](docs/screenshots/dark/ai-features.jpg)

<details>
<summary>展开查看处理任务与进度</summary>

![暗色处理任务：图片、人物、视频、OCR 和自动标签分别管理](docs/screenshots/dark/ai-tasks.jpg)

</details>

**AI 是可选部署。** 主应用单独运行即可浏览；开启 AI 后增加一个独立推理容器，模型由部署者配置，不提供模型商店。见 [AI 部署指南](docs/ai-compose.md)、[功能进度与边界](docs/features/intelligent-media-discovery-remaining-tasks.md)和[人物整理变更记录](docs/changes/CR-2026-044-person-merge-check.md)。

内存规划参考：人物或标签为 AI 服务预留约 **2 GiB**，OCR 约 **3 GiB**，多语言搜索及完整组合约 **6 GiB**；主应用另留约 **512 MiB–1 GiB**，扫描、大图及其他服务还需余量。这是工程预留建议，非已测最低配置，详见[测量场景与限制](docs/ai-compose.md#ai-功能内存参考)。

## 原文件始终由你掌握

- **原件只读**：不移动、不重命名、不编辑、不删除；移除媒体库也只清理应用内配置、索引、整理关联和缓存。
- **目录为准**：文件系统决定媒体是否存在。索引、缩略图与封面可以重建；离线或扫描中断保留最后可靠索引。
- **整理独立保存**：收藏、标签、人物关系和任务存放在 `/app/data`。可靠扫描确认媒体已移除后，其收藏和标签关联会清理，标签名称保留。
- **账户保护**：首次启动创建单一管理员，支持登录、退出与 CSRF 防护；局域网可直接使用 HTTP，公网需外部 HTTPS 与访问控制。
- **任务可维护**：查看扫描、缩略图和视频预览进度，补齐缺失、检查处理结果与日志，配置扫描计划、缓存配额和资源并发。

![暗色媒体库管理：多媒体库、媒体数量、扫描状态和管理入口](docs/screenshots/dark/libraries.jpg)

<details>
<summary>展开查看外观、语言与浏览偏好</summary>

![暗色通用设置：主题、语言、默认布局、排序与视频预览偏好](docs/screenshots/dark/settings.jpg)

</details>

## 🎞️ 支持格式

| 类型 | 格式 |
| --- | --- |
| 图片 | JPEG（`.jpg`、`.jpeg`）、PNG（`.png`）、WebP（`.webp`）、GIF（`.gif`） |
| 视频 | MP4（`.mp4`）、MOV（`.mov`）、MKV（`.mkv`）、AVI（`.avi`） |

视频不会转码，能否播放取决于浏览器支持的编码。SVG、HEIC/HEIF、AVIF 和 RAW 暂不支持。大图和视频处理采用有界资源与超时，预览失败不会修改原件。

## 快速开始

需要 Linux `amd64` 或 `arm64`、Docker 和 Compose v2。普通浏览无需 AI 容器。

**1. 保存配置。** 新建空目录，将以下内容保存为 `compose.yaml`，把 `/mnt/photos` 换成你的媒体目录：

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

**2. 启动。**

```sh
docker compose up -d
```

**3. 打开并建库。** 访问 `http://<服务器局域网地址>:8080`，创建管理员，在“管理中心 → 媒体库”中选择目录。

### 参数说明

| 配置 | 说明 |
| --- | --- |
| `image` | 示例使用 `latest`；可控升级请固定已发布版本或 digest。候选功能不代表已进入此标签。 |
| `TZ` | 设置你的时区。 |
| `8080:8080` | 左侧是宿主端口，例如改为 `9000:8080`。 |
| `/mnt/photos:/library:ro` | 只挂载一个媒体根目录，并保留只读；后代目录不得是嵌套挂载点。多个媒体库在 UI 中选择。 |
| `./data:/app/data` | 保存数据库、设置、任务和缓存。原生 Linux 使用本地磁盘，升级前停写备份，不放在 SMB／NFS。 |

> **Docker Desktop**：应用数据应使用 Linux 命名卷。已有数据须先停止应用，完整备份、复制并核验后再切换；不要从宿主机打开活跃 SQLite，也不要同步旧 data 副本。见[数据与维护说明](docs/deployment.md)及 [ADR-0038](docs/adr/0038-docker-desktop-linux-sqlite-volume.md)。

## 文档与项目状态

适合已有文件夹归档的摄影作品、家庭影像与 NAS 媒体；照片备份、手机同步、多人相册、图片编辑和视频转码不在当前定位内。

- [部署、升级、备份、恢复与反向代理](docs/deployment.md)
- [可选本地 AI 配置](docs/ai-compose.md)
- [候选验证与正式发布边界](docs/changes/VERIFY-2026-10-26-release-candidate.md)
- [安全与文件访问边界](docs/security.md)
- [文档索引与开发入口](docs/README.md) · [架构](docs/architecture/README.md) · [更新日志](CHANGELOG.md)

源码开发使用 `.node-version` 固定的 Node.js **24.22.1** 与 npm **11.19.1**。后端为 Go，前端为 React／TypeScript，SQLite 保存应用状态；默认单容器，可选 AI 为双容器。

截图为 2026-10-26 本地部署的实际暗色界面，界面中的数量与失败状态是该实例快照，不代表模型质量或处理成功率承诺。[截图说明](docs/screenshots/dark/README.md)。

## 许可证

[GNU Affero General Public License v3.0 or later](LICENSE)（`AGPL-3.0-or-later`）
