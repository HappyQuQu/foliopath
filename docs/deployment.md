# FolioPath 候选部署与运维

## 状态

本文描述首个可用版本的候选部署方式。Stage 5 `S5-001A` 已建立根 `Dockerfile` 候选镜像：
它构建并嵌入真实 Vite SPA，以 UID/GID `65532:65532` 运行真实 `cmd/foliopath`，并在
linux/arm64 本地验证只读容器根、丢弃 capabilities、只读媒体、健康检查、MVP 媒体矩阵
和 SIGTERM。可信代理边界已通过 `S5-003`；[CR-2026-002](changes/CR-2026-002-authenticated-lan-http.md)
进一步允许经认证的 LAN HTTP，根 `compose.yaml` 已通过同架构
smoke。本机 100k/10k 容量档和 Chromium/Firefox/WebKit 候选自动化也已通过。
原生 amd64/arm64 候选运行、真实升级/回滚与目标容量已经通过；真实 Firefox 核心链及
原生 200%/400% 缩放已通过，物理读屏/触控/移动设备、Safari 缩放、最终不可变 digest、
供应链处置和 Release Candidate Gate 尚未完成，当前候选仍不可作为稳定版部署。
候选运行层已更新到固定 digest 的 Debian 13 distroless，Go 构建层固定为 1.26.5；
这不是安全签署。当前 [S5-007 供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)
的本机 arm64 修复候选为 `0 Critical / 0 High`，但仍被最终干净提交的原生双架构复扫、
provenance 和安全/合规签署阻断。

## 部署目标

- 一个容器、一个应用进程、一个 HTTP 端口。
- 用户通常只映射一个只读媒体根目录和一个可写数据目录。
- 新增 `/library` 下的媒体库只需在 Web 设置中操作，不修改 Compose。
- 媒体原件始终只读；数据库、设置、任务状态和缓存全部位于 `/app/data`。
- 目标发布 `linux/amd64` 与 `linux/arm64` 镜像，并用不可变版本标签升级。

## Docker Hub 自动发布

独立 workflow [`.github/workflows/dockerhub.yml`](../.github/workflows/dockerhub.yml)
负责构建并发布一个同时包含 `linux/amd64` 与 `linux/arm64` 的 Docker Hub image index。
自动测试 CI 已关闭，测试与候选证据由发布者在本地完成。推送到 `main`、推送
`vMAJOR.MINOR.PATCH` 标签或手动运行时，才会触发此 Docker 发布 workflow；PR 不触发。

在 GitHub 仓库的 **Settings → Secrets and variables → Actions** 中配置：

| 类型 | 名称 | 示例 / 用途 |
| --- | --- | --- |
| Secret | `DOCKERHUB_USERNAME` | Docker Hub 用户名或组织机器人账号 |
| Secret | `DOCKERHUB_TOKEN` | 具备目标仓库推送所需最小权限的 Docker Hub access token |
| Secret（可选） | `DOCKERHUB_DESCRIPTION_TOKEN` | 同步根 README 到 Docker Hub Overview |

触发与标签规则：

- 推送到 `main` 时自动构建并推送 `latest` 与 `sha-*` 标签。
- 推送 `vMAJOR.MINOR.PATCH` Git tag 时自动推送版本标签、`major.minor`、`sha-*`
  和 `latest`。
- 从 Actions 手动运行时必须指定标签，默认是 `edge`，不会更新 `latest`。
- 单个 GitHub-hosted runner 通过 Buildx/QEMU 构建并推送 amd64 与 arm64 manifest。
- 构建附带 SBOM 和 provenance；配置可选 description token 时同步根 README。

发布完成后验证：

```sh
docker buildx imagetools inspect evanqu/foliopath:VERSION
docker pull --platform linux/amd64 evanqu/foliopath:VERSION
docker pull --platform linux/arm64 evanqu/foliopath:VERSION
```

输出必须同时列出 `linux/amd64` 和 `linux/arm64`。部署仍应使用明确版本或 digest，而不是
依赖可变的 `latest`。自动发布入口不改变当前 Release Candidate No-Go 判断；只有适用 Gate
全部关闭后才能创建稳定 GitHub Release。

## 候选 Compose

权威候选配置是仓库根 [`compose.yaml`](../compose.yaml)，参数清单见
根 [`.env.example`](../.env.example)。复制为 `.env` 后必须填写：

- `FOLIOPATH_IMAGE`：明确版本或 digest；不得使用会静默漂移的 `latest`；
- `FOLIOPATH_LIBRARY_PATH`：宿主机唯一媒体呈现根；
- `FOLIOPATH_DATA_PATH`：宿主机本地、由 UID/GID `65532:65532` 可写的数据目录；
- `FOLIOPATH_BIND_ADDRESS`：宿主端口绑定地址，默认 `0.0.0.0` 供受信局域网访问；
- `FOLIOPATH_PORT`：宿主端口，默认 `8080`。

官方多架构镜像发布在 `evanqu/foliopath`。README 快速开始使用 `latest`；需要可控升级时，
请在 `.env` 中改为明确版本或 digest。需要验证当前源码时，也可以本地构建：

```sh
docker build --build-arg VERSION=stage5-local -t foliopath:stage5-local .
```

并把 `.env` 中的 `FOLIOPATH_IMAGE` 设为 `foliopath:stage5-local` 后运行
`docker compose up -d`。Compose 默认将宿主端口发布到所有 IPv4 接口，以
UID/GID `65532:65532`、只读根、全部 capabilities 丢弃、`no-new-privileges` 和受限
`/tmp` tmpfs 启动。镜像内 healthcheck 已由候选 smoke 验证。

本地 tag 只标识当前源码验证，不是不可变发布引用。长期部署建议使用 Docker Hub 上的明确
版本或 digest。

### 为什么只需要两个 volume

```text
宿主机 /mnt/photos       →  容器 /library   只读允许边界
宿主机 ./foliopath-data →  容器 /app/data  可写持久数据
```

Web 设置只能从 `/library` 中选择媒体库。例如，`/mnt/photos` 中存在 `family`、
`work/2026` 与 `archive` 普通目录时，可以分别创建 `/library/family`、
`/library/work/2026` 和 `/library/archive` 媒体库；每个库默认递归包含其子目录，
无需重启容器或再写 volume。

`/library` 是唯一媒体挂载目标。它自身可以是一个 bind mount 或 volume，但不得在
`/library/family`、`/library/work` 等后代路径再嵌套 volume、bind mount 或其他
mount point。下面这种拼装方式**不受支持且会被拒绝**：

```yaml
# 错误示例：不要这样配置
volumes:
  - /volume1/family:/library/family:ro
  - /volume2/work:/library/work:ro
  - ./foliopath-data:/app/data
```

若媒体分布在多个独立宿主机卷，部署者必须先在宿主机侧把它们提供为一个、对容器
表现为单一文件系统且没有后代 mount crossing 的呈现根，然后只把该根挂载一次：

```yaml
volumes:
  - /host/media-presentation:/library:ro
  - ./foliopath-data:/app/data
```

FolioPath 不选择、配置或承诺具体 union、聚合文件系统或 NAS 技术；部署者需要验证
其一致性、权限和故障语义。若新的宿主目录不在现有呈现根中，必须先调整宿主机根
布局或根挂载，应用不能在 UI 中扩大容器被授予的边界。完整决策见
[ADR-0009](adr/0009-linux-openat2-single-media-root.md)。

### Linux 路径解析运行条件

Linux 版本从已锚定的 `/library` 目录文件描述符打开每个媒体路径，并要求内核
`openat2` 支持
`RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV`。这同时拒绝越界、
符号链接和 `/library` 下的同设备或跨设备 mount crossing。

内核未实现所需调用/flags，或容器 seccomp/LSM 阻断它们时，FolioPath 必须失败
关闭并把媒体根视为不可用；不会降级为较弱的 realpath、device/inode 或 `os.Root`
检查。发布兼容性取决于实际系统调用，不只取决于内核版本字符串。生产容器不需要
`CAP_SYS_ADMIN`；该 capability 只用于隔离 CI 中构造 mount 边界测试。

非 Linux adapter 仅用于开发与补充测试，不构成同等级 mount-boundary 证明，也
不扩大 MVP 的 Linux 平台承诺。

## 文件权限

- 候选镜像以固定非 root UID/GID `65532:65532` 运行。
- 该用户对 `/library` 及所选普通子目录需要读取和目录遍历权限，对 `/app/data` 需要读写、创建、重命名和同步权限。
- 媒体映射必须使用 `:ro`。应用报“只读”不能替代 Docker 层的只读保护。
- 建议在首次启动前创建数据目录并设置精确所有者，不使用全局可写 `0777` 作为长期方案。
- SELinux 主机可能需要专用 volume label；准确选项应在对应发行版验证后加入平台说明。

应用需要临时文件时，应使用受控的 `/app/data/tmp` 或 Compose 中受限的 `/tmp` tmpfs。
候选 smoke 已确认 libvips、FFmpeg 和健康检查可在只读容器根下运行；时区/证书的最终
平台矩阵仍归发布 Gate。

## 数据目录布局

当前布局：

```text
/app/data/
├── foliopath.db          SQLite 主文件
├── foliopath.db-wal      运行时 WAL，可能存在
├── foliopath.db-shm      运行时共享内存文件，可能存在
├── cache/                可重建缩略图与视频封面
└── tmp/                  可清理的受控临时文件
```

`foliopath.db`、`cache/` 与 `tmp/` 已由当前 composition root 固定。WAL/SHM 是 SQLite
运行时文件，停止后的安全备份中可能不存在。所有持久状态必须留在 `/app/data`；不得把
数据库放入媒体 volume，也不得把原媒体复制进缓存目录。

SQLite WAL 依赖正确的锁和同步语义。`/app/data` 必须使用宿主机本地文件系统或明确验证兼容的存储；MVP 不支持直接把活动数据库放在 SMB/NFS 上。单一 `/library` 根可以来自经过验证的 NAS 挂载，但不得在其后代再嵌套其他挂载；断连时应用应把受影响的库标为离线并保留索引。

每次进程启动都会按媒体库 ID 分页安排或合并一次 `startup` 完整扫描。启动时仍处于
running 的旧任务由 durable lease 约束；无论 lease 在进程启动前还是启动后才到期，
worker 都会有界恢复并重新领取。取消、离线、权限失败或异常退出留下的最后可靠索引会保留，
只有后来一次完整成功扫描才发布新 generation 并清理陈旧派生行。

## 首次启动

当前候选流程：

1. 容器验证 `/app/data` 可写、数据库可打开且迁移可以安全执行。
2. 服务启动 API 和前端；未完成初始化时只开放受限的单管理员创建流程，不存在默认密码。
3. 管理员登录后进入设置，在 `/library` 的安全目录选择器中创建一个或多个媒体库。
4. 创建成功后异步启动首次完整扫描，页面显示阶段、跳过统计与计数；应用启动时校准，之后默认每 24 小时完整扫描。
5. 媒体库可浏览后继续在后台生成所需缩略图；默认 10 GiB 配额达到水位时按 LRU 清理可重建缓存。

当前候选已包含内建单管理员、会话、CSRF 和退出登录，可在受信局域网通过 HTTP 使用。
Stage 5 尚未通过，因此仍不能把候选作为稳定发行版或匿名服务；公网和不可信网络必须由
部署者在应用外提供 TLS 与访问控制。

## 健康检查与可观察性

当前候选应用提供：

- `GET /health/live`：进程事件循环可响应，不检查媒体库是否在线。
- `GET /health/ready`：数据库已打开、迁移完成且服务可以接受业务请求；一个媒体库离线不应让整个实例不就绪。
- `GET /api/v1/status`：认证后返回版本和安全的能力/初始化状态。

应用启动时会先准备固定 `/app/data`，打开 SQLite 并执行嵌入 migration，成功后才启动
HTTP 并进入 ready。数据目录不可用、数据库打不开或 migration 失败时进程失败关闭，不会带着
部分 schema 提供业务服务。系统状态路由要求有效管理员会话。

健康端点不得泄露路径、数据库错误或版本依赖细节。候选 healthcheck 的命令、间隔和启动
宽限已经通过本机镜像 smoke；最终 digest 仍须在原生双架构重复。

当前可观察面包括 JSON 标准输出/错误日志、live/ready、认证系统状态、扫描运行/历史和
设置中的缓存配额。日志轮转由容器平台负责。当前没有 Prometheus/metrics endpoint，也不
直接暴露队列深度或宿主磁盘余量；部署者必须在容器/宿主平台监控 `/app/data` 容量。
正常日志不记录宿主机路径、认证信息或媒体工具完整 stderr。

生产 final stage 有意不包含 shell、curl、tar 或包管理器。不要进入运行容器安装调试
工具或改变其不可变闭包；诊断、归档或网络探测应从宿主机执行，或使用固定版本的短生命周期
辅助容器，并只授予完成该操作所需的精确 volume、网络和权限。下文备份命令均为宿主机命令。

## 网络、反向代理与认证

- 应用二进制默认监听 `127.0.0.1:8080`，根 Compose 在容器内设置为
  `0.0.0.0:8080`。可通过 `FOLIOPATH_LISTEN` 或
  `--listen=<IP>:<PORT>` 设置，参数优先于环境变量；两者都只接受一个数值 IP 与
  `1～65535` 端口。
- `/library` 与 `/app/data` 是固定容器边界，不提供改写它们的环境变量或参数。媒体库在
  Web/SQLite 中配置，不能转化为每库一个启动变量。
- 非回环监听可直接服务经认证的受信 LAN HTTP。没有可信代理配置时，应用清除所有
  `Forwarded`/`X-Forwarded-*` 声明，并使用直连 peer 作为客户端身份。
- 外部 HTTPS 代理可显式设置 `FOLIOPATH_TRUSTED_PROXIES`。只接受逗号分隔的精确 IP CIDR；
  不要使用 `/0` 或包含普通客户端/NAT 出口的宽泛网段。非回环监听设置该变量后进入
  代理专用模式，远程直连失败关闭。
- MVP 契约与当前候选只支持单跳代理。代理必须覆盖客户端提交的转发头并发送恰好一个
  `X-Forwarded-Proto: https`、公共 `X-Forwarded-Host` 和数值客户端 IP
  `X-Forwarded-For`。应用拒绝链式、多值、缺失、非 HTTPS或与 `Forwarded` 混用的请求。
- 公网或其他不可信网络必须由部署者使用 TLS 反向代理，并通过回环绑定、私有容器网络或
  firewall 防止绕过代理。TLS/代理不是 FolioPath 的必需部署单元；可信 CIDR 也不是网络 ACL。
- 反向代理需要允许图片和视频响应流式传输、Range、较长读取和客户端取消；不应缓冲整个视频到内存。
- 验证后的代理 HTTPS transport 会启用 Secure Cookie 与 HSTS；状态修改继续要求
  session-bound CSRF，setup/login 要求公共 HTTPS origin 与 host/port 完整同源。

认证产品决策和威胁要求见[安全模型](security.md)与[需求确认清单](requirements-checklist.md)。

## 备份与恢复

### 备份范围

必须备份整个 SQLite family；它包含管理员凭据、媒体库配置、应用设置、扫描历史、索引和
任务状态。停止应用后通常只剩 `foliopath.db`，但运维流程不应依赖这一假设。

`cache/` 与 `tmp/` 可以不备份：其中只有可重建缩略图、视频 poster 和临时文件。保留
`cache/` 可以减少大型媒体库恢复后的重建成本，但不是数据完整性的必要条件。

原媒体不属于 FolioPath 数据备份；用户仍需独立备份 `/mnt/photos`。

### MVP 安全备份流程

在没有内建在线备份命令前，推荐：

1. 在 `compose.yaml` 所在目录运行 `docker compose stop foliopath`。
2. 用 `docker compose ps --status running --quiet foliopath` 确认没有运行中的应用容器。
3. 从 `.env` 确认 `FOLIOPATH_DATA_PATH` 的精确宿主路径，归档该目录的全部内容；最安全的
   默认是包含 `foliopath.db`、可能存在的 `-wal`/`-shm`、`cache/` 和 `tmp/`。
4. 把归档复制到不与活动数据共盘的受保护位置，记录当前镜像版本/digest，并验证归档可读。
5. 运行 `docker compose start foliopath`，等待 `docker compose ps` 显示 healthy。

示例中的路径必须替换为经过人工确认的精确数据目录，不能把未展开变量、空值、工作区根或
媒体目录作为归档目标：

```sh
docker compose stop foliopath
docker compose ps --status running --quiet foliopath
tar --numeric-owner -C /srv/foliopath-data -cpf /srv/backups/foliopath-data.tar .
docker compose start foliopath
```

不要在运行时只复制 `foliopath.db` 而忽略 `-wal`/`-shm`。未来如提供在线备份，必须使用 SQLite backup API 或受控 checkpoint，并通过恢复演练证明。

Stage 5 候选 smoke 已在应用停止后归档完整 SQLite family，并有意省略可重建的
`cache/` 与 `tmp/`；恢复到新空目录后，管理员初始化状态保留且这两个目录会重新创建。
这证明离线最小恢复路径，不授权在应用运行时复制数据库文件。

### 恢复演练

1. 停止容器并保留当前数据目录，不直接覆盖唯一副本。
2. 把备份恢复到一个新的空数据目录并校正所有者。
3. 使用与备份 schema 兼容的镜像启动。
4. 验证迁移、媒体库配置、扫描历史和抽样媒体浏览。
5. 若宿主机媒体位置改变，先恢复相同的容器内 `/library` 布局；数据库不应依赖宿主机绝对路径。

恢复目标必须由容器内 UID/GID `65532:65532` 读写。rootless Docker 或启用 user namespace
remapping 时，宿主机实际 ID 可能不同，应以该运行时的映射为准，不能机械执行全局
`chown -R`。先在新目录演练并验证成功，再切换 `.env` 的 `FOLIOPATH_DATA_PATH`；不要解包
覆盖唯一活动副本。

候选镜像的自动恢复演练已经建立并在本机 linux/arm64 通过；原生 linux/arm64 与
linux/amd64 还分别以两个不同的不可变候选 image ID 通过向前升级和“旧镜像＋升级前
离线备份”配对回滚。最终不可变 digest 与供应链签署仍是发布阻断。

## 升级与回滚

推荐升级流程：

1. 阅读版本说明，确认格式、迁移和平台支持变化。
2. 停止服务并完成可恢复备份。
3. 拉取明确版本标签或 digest。
4. 以相同 `/library` 与 `/app/data` 映射启动新版本。
5. 检查迁移、readiness、媒体库离线状态和抽样浏览，再清理旧镜像。

数据库只做向前迁移。升级后直接运行旧镜像可能不安全，因此“回滚”必须同时恢复升级前数据备份，除非版本说明明确保证 schema 向后兼容。迁移失败时服务应停止进入业务就绪，而不是带着部分 schema 继续运行。

当前仓库尚无已发布的稳定 digest，因此本阶段使用两个不同的不可变候选 image ID 验证
升级/回滚流程，并已通过 `S5-004B`。首个稳定版本发布后，每次新增 migration 仍必须以
当时真实前一稳定 digest 和升级前离线备份复跑同一演练。

## 媒体格式与播放边界

| 类型 | 扩展名 | 服务端承诺 | 浏览器行为 |
| --- | --- | --- | --- |
| 图片 | `.jpg`、`.jpeg`、`.png`、`.webp` | 索引、元数据、WebP 缩略图 | 原内容查看 |
| 动图 | `.gif` | 索引、元数据、静态缩略图 | 原内容动画 |
| 视频 | `.mp4`、`.mov`、`.mkv`、`.avi` | 索引、ffprobe 元数据、FFmpeg poster、原文件 Range | 仅当前浏览器原生兼容 codec 可直接播放 |

扩展名匹配不区分大小写。MVP 不转码，也不生成兼容播放副本；支持视频容器不等于支持其中
任意 codec。SVG、HEIC/HEIF、AVIF 与 RAW 不进入 MVP 索引/缩略图契约。

`POST-MVP-1` 的视频故事板候选复用同一 FFmpeg runtime，为成功探测且至少 2 秒的视频生成
4 或 10 帧 WebP sprite。它不新增端口、环境变量、volume、服务或媒体写权限，仍只写入
`/app/data` 的可重建缓存。当前生产镜像纵向链已通过 VSP-301，但原生 linux/amd64 与
linux/arm64 同提交候选证据仍由
[VSP-302](gates/POST-MVP-1/vsp-302-target-platform.md)阻断，因此不能把该能力描述为稳定发布。

## 当前候选已知限制

- 官方镜像位于 `evanqu/foliopath`；生产部署建议固定明确版本或 digest。
- 发布平台目标是 Linux amd64/arm64。非 Linux adapter 只用于开发；本轮候选已由本机
  原生 arm64 与操作者指定的原生 amd64 服务器完成运行矩阵，最终不可变 digest 尚未签署。
- `/library` 只能有一个顶层媒体挂载，后代不能是 mount point；目录 symlink 不跟随。
- `/app/data` 只支持单实例独占的可靠本地文件系统，不支持把活动 SQLite 放在 SMB/NFS，
  也不支持多实例共享写入。
- 离线备份/恢复及不同不可变候选间的升级/配对回滚已自动化并通过；在线备份不在当前
  自动化承诺内，未来新增 migration 仍须对真实前一稳定 digest 复跑。
- 100k 媒体/10k 目录候选档已在本机 linux/arm64 与指定原生 linux/amd64 服务器通过；
  100k 全量媒体、cache 90%→80% 水位和 Chromium/Firefox/WebKit 的 100k FPS/RSS
  预算也已通过，`S5-005` 已关闭。
- Chromium、Firefox、WebKit 自动化候选矩阵已建立，但 WebKit 不等同于 Safari 真机；
  最终浏览器版本、读屏和物理设备仍待 `S5-006B`。
- 当前本机 arm64 修复候选扫描为 `0 Critical / 0 High`；最终原生双架构 digest 的
  `all` 策略复扫、provenance 与安全/合规签署仍是发布阻断。
- 当前没有 Prometheus/metrics endpoint、内建日志轮转或宿主磁盘余量采集；这些由部署平台监控。
- MVP 只有一个管理员，不支持匿名 LAN、多用户角色、分享链接、上传或原媒体整理。
- 正确性依赖启动、手动和计划完整扫描，不依赖 filesystem watcher；默认计划间隔为 24 小时。
- 查看器不提供完整 EXIF、显式下载按钮或移动端滑动切换。

## 故障语义

| 故障 | 期望行为 |
| --- | --- |
| 单一 `/library` 根断开或变得不可读 | 标记受影响媒体库离线，保留索引，不执行陈旧清理 |
| `/app/data` 不可写 | readiness 失败，拒绝需要写入的操作，记录安全错误 |
| 磁盘不足 | 停止派生任务并保护数据库；不能删除原媒体或静默损坏索引 |
| 扫描中容器退出 | 下次启动标记任务中断，新完整扫描负责收敛 |
| 缩略图缓存损坏 | 标记派生数据待重建，保留媒体索引 |
| 数据库迁移失败 | 不进入业务就绪，提示从备份恢复或修复环境 |

## 发布前部署门槛

- amd64/arm64 镜像通过相同集成与媒体 fixture 测试。
- 非 root、`:ro` 媒体、capabilities 丢弃和只读根文件系统组合经过验证。
- Compose 中的镜像、端口、UID/GID、healthcheck 和环境变量全部为真实值。
- 完成数据库备份/恢复、升级失败和磁盘已满演练。
- SBOM、第三方许可证和镜像漏洞扫描纳入发布流程；候选自动化已经建立，但只有最终
  双架构 digest 的全阻断扫描、notices/provenance 与签署完成后才满足此项。
- 明确支持的宿主机、CPU/内存建议和目标库规模，不用未验证的数字宣传。
- 以约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器完成主要容量验收；具体吞吐与延迟只能引用可重复 benchmark。
- 单管理员初始化、登录、会话、退出和 CSRF 测试通过；无认证开发构建不会被文档引导直接暴露到互联网。
