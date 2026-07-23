# FolioPath 部署与运维草案

## 状态

本文描述首个可用版本的目标部署方式。FS-05 probe 已以 UID/GID `65532:65532` 验证非 root、
健康检查、只读挂载和恢复语义，但当前仍没有正式应用或发布镜像；示例不可直接用于生产。

## 部署目标

- 一个容器、一个应用进程、一个 HTTP 端口。
- 用户通常只映射一个只读媒体根目录和一个可写数据目录。
- 新增 `/library` 下的媒体库只需在 Web 设置中操作，不修改 Compose。
- 媒体原件始终只读；数据库、设置、任务状态和缓存全部位于 `/app/data`。
- 目标发布 `linux/amd64` 与 `linux/arm64` 镜像，并用不可变版本标签升级。

## 目标 Compose

```yaml
services:
  foliopath:
    image: ghcr.io/YOUR_GITHUB_USERNAME/foliopath:VERSION
    container_name: foliopath
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - /mnt/photos:/library:ro
      - ./foliopath-data:/app/data
    environment:
      TZ: Asia/Shanghai
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
```

首个镜像发布时必须替换占位仓库、固定 `VERSION`、记录镜像内非 root UID/GID，并加入经过真实镜像验证的 healthcheck。生产部署不建议使用会静默漂移的 `latest`。

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

- 最终镜像以固定非 root 用户运行；发布前需文档化 UID/GID。
- 该用户对 `/library` 及所选普通子目录需要读取和目录遍历权限，对 `/app/data` 需要读写、创建、重命名和同步权限。
- 媒体映射必须使用 `:ro`。应用报“只读”不能替代 Docker 层的只读保护。
- 建议在首次启动前创建数据目录并设置精确所有者，不使用全局可写 `0777` 作为长期方案。
- SELinux 主机可能需要专用 volume label；准确选项应在对应发行版验证后加入平台说明。

应用需要临时文件时，应使用受控的 `/app/data/tmp` 或受限 tmpfs。若启用只读容器根文件系统，需先确认 libvips、FFmpeg、时区和证书在该模式下均通过集成测试。

## 数据目录布局

目标布局：

```text
/app/data/
├── foliopath.db          SQLite 主文件
├── foliopath.db-wal      运行时 WAL，可能存在
├── foliopath.db-shm      运行时共享内存文件，可能存在
├── cache/                可重建缩略图与视频封面
└── tmp/                  可清理的受控临时文件
```

具体文件名在实现迁移前可调整，但所有持久状态必须留在 `/app/data`。不得把数据库放入媒体 volume，也不得把原媒体复制进缓存目录。

SQLite WAL 依赖正确的锁和同步语义。`/app/data` 必须使用宿主机本地文件系统或明确验证兼容的存储；MVP 不支持直接把活动数据库放在 SMB/NFS 上。单一 `/library` 根可以来自经过验证的 NAS 挂载，但不得在其后代再嵌套其他挂载；断连时应用应把受影响的库标为离线并保留索引。

## 首次启动

目标流程：

1. 容器验证 `/app/data` 可写、数据库可打开且迁移可以安全执行。
2. 服务启动 API 和前端；未完成初始化时只开放受限的单管理员创建流程，不存在默认密码。
3. 管理员登录后进入设置，在 `/library` 的安全目录选择器中创建一个或多个媒体库。
4. 创建成功后异步启动首次完整扫描，页面显示阶段、跳过统计与计数；应用启动时校准，之后默认每 24 小时完整扫描。
5. 媒体库可浏览后继续在后台生成所需缩略图；默认 10 GiB 配额达到水位时按 LRU 清理可重建缓存。

首个稳定版必须包含内建单管理员、会话和退出登录。认证未完成的开发预览版只能绑定回环地址或位于可信认证反向代理后，不属于可公网部署版本。

## 健康检查与可观察性

计划提供：

- `GET /health/live`：进程事件循环可响应，不检查媒体库是否在线。
- `GET /health/ready`：数据库已打开、迁移完成且服务可以接受业务请求；一个媒体库离线不应让整个实例不就绪。
- `GET /api/v1/status`：认证后返回版本和安全的能力/初始化状态。

健康端点不得泄露路径、数据库错误或版本依赖细节。容器 healthcheck 的命令、间隔和启动宽限需要用最终镜像验证，避免长扫描导致误杀。

最低可观察信息包括扫描状态、最近成功时间、安全错误码、队列深度、缓存占用和可写数据目录的磁盘余量。日志输出到标准输出/错误；轮转由容器平台负责。默认不记录宿主机路径、认证信息或媒体工具完整 stderr。

## 网络、反向代理与认证

- 默认端口绑定 `127.0.0.1`；内建认证完成并通过测试前不得改成局域网或公网监听。
- 对局域网或公网提供访问时，使用 TLS 与可信认证反向代理；不得直接暴露无认证端口。
- 应用只能信任显式配置的代理来源提交的 `Forwarded`/`X-Forwarded-*` 头。
- 反向代理需要允许图片和视频响应流式传输、Range、较长读取和客户端取消；不应缓冲整个视频到内存。
- 稳定版应用内 Cookie 会话必须配置 HTTPS、安全 Cookie、CSRF 防护和会话失效流程。

认证产品决策和威胁要求见[安全模型](security.md)与[需求确认清单](requirements-checklist.md)。

## 备份与恢复

### 备份范围

必须备份：

- SQLite 中的媒体库配置、管理员凭据、应用设置，以及未来其他不可重建的用户数据。

可选备份：

- 索引、缩略图和视频封面。这些数据可重建，但大型媒体库重建成本可能较高。

原媒体不属于 FolioPath 数据备份；用户仍需独立备份 `/mnt/photos`。

### MVP 安全备份流程

在没有内建在线备份命令前，推荐：

1. 停止 FolioPath 容器并确认进程退出。
2. 备份整个 `/app/data` 到另一个存储位置，保留权限和时间信息。
3. 重新启动容器并检查 readiness 与媒体库状态。

不要在运行时只复制 `foliopath.db` 而忽略 `-wal`/`-shm`。未来如提供在线备份，必须使用 SQLite backup API 或受控 checkpoint，并通过恢复演练证明。

### 恢复演练

1. 停止容器并保留当前数据目录，不直接覆盖唯一副本。
2. 把备份恢复到一个新的空数据目录并校正所有者。
3. 使用与备份 schema 兼容的镜像启动。
4. 验证迁移、媒体库配置、扫描历史和抽样媒体浏览。
5. 若宿主机媒体位置改变，先恢复相同的容器内 `/library` 布局；数据库不应依赖宿主机绝对路径。

发布前必须自动化或至少记录一次真实恢复演练。

## 升级与回滚

推荐升级流程：

1. 阅读版本说明，确认格式、迁移和平台支持变化。
2. 停止服务并完成可恢复备份。
3. 拉取明确版本标签或 digest。
4. 以相同 `/library` 与 `/app/data` 映射启动新版本。
5. 检查迁移、readiness、媒体库离线状态和抽样浏览，再清理旧镜像。

数据库只做向前迁移。升级后直接运行旧镜像可能不安全，因此“回滚”必须同时恢复升级前数据备份，除非版本说明明确保证 schema 向后兼容。迁移失败时服务应停止进入业务就绪，而不是带着部分 schema 继续运行。

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
- SBOM、第三方许可证和镜像漏洞扫描纳入发布流程。
- 明确支持的宿主机、CPU/内存建议和目标库规模，不用未验证的数字宣传。
- 以约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器完成主要容量验收；具体吞吐与延迟只能引用可重复 benchmark。
- 单管理员初始化、登录、会话、退出和 CSRF 测试通过；无认证开发构建不会被文档引导直接暴露到互联网。
