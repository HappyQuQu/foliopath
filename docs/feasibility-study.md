# FolioPath 可行性研究

## 结论与状态

FolioPath 在既定范围内具备实现可行性，但当前结论仍是 **有条件推进（Conditional Go）**：
FS-01 已通过 Linux/arm64 `openat2` 同/跨设备与 self-bind mount 边界和真实 HTTP test
harness 子范围；FS-02 已通过当前 SQLite/generation 正确性范围；FS-03 已取得 Darwin
FFmpeg 媒体矩阵局部证据；FS-04 的 Linux/arm64 四核、4 GiB 扫描/索引子范围已通过目标
数据档。FS-01、FS-03/04 的完整门槛与 FS-05 仍未通过。只可继续验证性 spike、契约/生成
护栏和不扩大信任边界的实验脚手架，尚未获准产品功能开发或发布承诺。

本文依据已接受的 [ADR-0001](adr/0001-go-react-sqlite.md)、[ADR-0002](adr/0002-library-path-model.md) 和 [ADR-0003](adr/0003-scan-consistency.md)。用户已于 2026-07-23 确认[需求确认清单](requirements-checklist.md)中的全部 A 方案；规模、格式和认证已经是验证目标，而不是未决产品选项。

文中使用四种证据等级：

- **已知事实**：已接受的项目约束，或可直接从当前仓库确认的状态。
- **已验证**：报告中列出的命令在明确环境执行成功；结论只适用于报告声明的 scope。
- **假设**：合理但尚未由用户需求、原型或测试证明的前提。
- **需 spike**：必须用小型可运行实验收集证据；本文不提供或虚构 benchmark 数字。

## 分项评估

| 维度 | 已知事实 | 假设与未知 | 需要的验证 | 初步判断 |
| --- | --- | --- | --- | --- |
| 产品 | 文件夹优先、多媒体库、递归浏览、只读原文件和 RQ-001～014 已成为基线；主要容量档为 10 万媒体/1 万目录/4 GiB/四核 | 相对现有工具的实际使用价值和细节可用性尚无真实用户证据 | 用可点击原型或最小纵向切片验证创建媒体库、定位目录、递归浏览和离线恢复 | 产品范围已冻结，仍需可用性验证 |
| 技术架构 | 已决定采用 Go 模块化单体、嵌入 React SPA、SQLite、govips/libvips 与 FFmpeg；离线契约检查、确定性 TypeScript 类型、唯一 Web API client、摘要锁、真实 PR 语义兼容比较、strict TypeScript 与原生双架构 CI 已通过 | 尚无应用入口、生产 HTTP handler、React 产品应用、媒体 adapter、静态资源嵌入或生命周期；`sqlc` 生成也未建立 | 建立最小服务，贯通迁移、API、后台任务、静态资源嵌入和优雅退出 | 后端边界与 S1 wire 契约已有局部证据，整体集成仍未验证 |
| 扫描与性能 | generation、流式遍历、短事务批量写入和故障保留已有自动化正确性证据；FS-04 在受限 Linux/tmpfs 上完成 10 万媒体/1 万目录、最大深度 32 的混合扫描/索引档，并单测 1,000 层 SQLite rollup | 代表性 NAS 存储、媒体探测/缩略图、FTS、正式 HTTP、全局队列、RSS 和前端交互仍无完整实测 | 在代表性设备继续测量扫描、搜索、缩略图、HTTP 与前端并发，固定预算 | 当前扫描/索引子范围通过；叶到根批量 rollup 已去掉 O(D×A) 和 O(A×depth) 路径并对损坏拓扑失败关闭，完整 FS-04 仍 Conditional |
| 媒体格式 | 确认 JPEG/PNG/WebP/GIF 图片与 MP4/MOV/MKV 视频；FS-03 已验证 Darwin/arm64 与 Linux/amd64 QEMU 环境的合成 FFmpeg 矩阵、视频封面、动画 GIF 与截断视频拒绝 | libvips/govips 图片链路、动画策略、原生双架构最终镜像、浏览器直放和生产任务隔离尚无实测 | 对固定矩阵继续做 libvips 缩略图、方向、色彩、动画、超时和浏览器播放测试，并运行原生 CI | FS-03 局部证据通过，完整矩阵仍 Conditional |
| SQLite | 真实文件数据库、Goose 迁移、WAL 返回值、外键、busy timeout、SQLite 3.53.3、generation 故障路径和原子 finalize 已验证 | 目标规模 FTS5/并发、磁盘满、真实强杀、长期 checkpoint、SMB/NFS 与备份恢复未知 | FS-04 压力/容量测试；FS-05 Linux 容器、强杀、磁盘满和恢复演练 | FS-02 当前 scope 通过；`/app/data` 仍禁止放在 SMB/NFS |
| 安全 | 已定义 `/library` 边界；Darwin 与原生 Linux amd64/arm64 路径矩阵、Linux `openat2` 同/跨设备和 self-bind mount 拒绝、HTTP test harness 与错误脱敏已有自动化证据 | 生产 handler 对权威 OpenAPI 的实现、认证/错误 envelope、只读发布 volume、运行期 unmount、解析器资源上限和认证尚未验证 | 完成 FS-01 生产 HTTP/发布容器剩余矩阵，再验证媒体炸弹/超时、管理员/会话/CSRF 和容器权限 | FS-01 为 Conditional；仍是发布阻断项 |
| 运维 | 目标为一个容器、一个端口、一个本地持久数据目录；缓存可重建 | 健康检查、磁盘配额、升级失败、数据库恢复和观测字段尚未实现 | 构建镜像并演练启动、停机、备份、恢复、迁移、磁盘满和挂载离线 | 部署简单，但恢复流程必须先演练 |
| 跨架构 | Go 支持 amd64/arm64；目标镜像包含原生 libvips 与 FFmpeg | govips 的 CGO、系统库、编解码能力和不同架构镜像内容可能不一致 | 在 linux/amd64 与 linux/arm64 构建并运行同一媒体 fixture 与容器冒烟测试 | 可行但不是“纯 Go”交叉编译，发布链路有明显风险 |
| 许可证 | 项目采用 AGPL-3.0-or-later，仓库已有许可证文本 | Go/npm 依赖、libvips、FFmpeg 构建选项、编解码库及分发地区的义务尚未形成清单 | 生成依赖与镜像 SBOM，核对许可证兼容性、源码提供义务及 FFmpeg 构建配置；必要时寻求法律意见 | 原则上可行，发布前必须完成合规审查 |

## 当前 spike 状态

| Spike | 状态 | 范围与产物 | 通过条件 / 剩余门槛 |
| --- | --- | --- | --- |
| [FS-01 路径边界](spikes/fs-01-path-boundary.md) | **Conditional** | Darwin 与原生 Linux amd64/arm64 安全目录/文件打开；Linux `openat2` 同/跨设备及 self-bind mount 拒绝；真实 HTTP test harness 覆盖 ID、Range、条件请求、攻击矩阵与脱敏 | 已验证子范围通过；生产 handler、认证/错误 envelope、只读发布 volume、运行期 unmount 与长期 churn 仍待验证 |
| [FS-02 SQLite 与扫描](spikes/fs-02-sqlite-generation.md) | **Passed（当前正确性 scope）** | 真实文件 SQLite、嵌入迁移、generation、批量 upsert、故障/取消/离线/重启保留和原子成功清理 | 当前 scope 通过；磁盘满、真实 `SIGKILL`、长期 WAL 压力、SMB/NFS 与备份恢复属于后续门槛 |
| [FS-03 媒体矩阵](spikes/fs-03-media-matrix.md) | **Conditional** | 运行时合成 JPEG/PNG/WebP/GIF、MP4/MOV/MKV/FFV1 与损坏视频；Darwin/arm64、QEMU amd64 及原生 Linux amd64/arm64 CI 的合成 FFmpeg 矩阵通过 | libvips/govips、任务超时/隔离、浏览器直放及双架构最终镜像仍待验证 |
| [FS-04 容量基线](spikes/fs-04-capacity-baseline.md) | **Conditional（扫描/索引子范围通过）** | Linux/arm64、4 核/4 GiB、10 万媒体/1 万目录混合宽度/32 层深度；另有 1,000 层 finalize 档；generation 扫描、聚合正确性、并发索引读取、heap 和 DB 大小有可复现结果 | 代表性磁盘、RSS、媒体/缩略图、FTS/keyset、HTTP/前端并发和预算趋势仍待验证 |
| FS-05 镜像与恢复 | 未开始 | 为目标架构构建单容器镜像，验证非 root、只读媒体、健康检查、SQLite 备份/恢复与升级迁移 | 两个目标架构行为一致，恢复演练成功，镜像依赖和许可证可追溯 |

Spike 应提交可复现命令、fixture 说明、执行环境、未执行项和结论。若硬件、数据集或构建选项不同，不得横向比较为同一 benchmark。FS-04 的 Docker Desktop/tmpfs 数据只描述当前扫描/索引子范围，不能泛化为代表性 NAS、完整产品吞吐或发布预算。

SQLite WAL 只适用于具备正确共享内存、锁和同步语义的同机文件系统；WAL 模式必须检查 `PRAGMA journal_mode=WAL` 的实际返回值，且 `-wal` 文件属于持久状态的一部分。当前实现和后续运维验证均以 [SQLite WAL documentation](https://sqlite.org/wal.html) 为基线。

## Go / No-Go 门槛

### 允许进入 MVP 实施

- [需求确认清单](requirements-checklist.md)中的 RQ-001～RQ-014 已全部确认 A，并已同步为产品基线。
- FS-01 与 FS-02 均满足其完整门槛。当前 FS-02 在定义的正确性 scope 通过，FS-01 只完成
  Linux/openat2 与 HTTP harness 子范围并仍为 Conditional，因此本条尚未满足。
- FS-03 产出首版媒体支持矩阵，README 与实现使用同一口径。
- 性能目标、目标硬件和数据规模已经定义；FS-04 没有发现必须改写架构的阻断问题。
- FS-05 至少证明目标发布架构可构建、可启动、可备份和可恢复。
- 依赖许可证与 FFmpeg 构建配置没有未处理的分发阻断项。

### 必须暂停、缩减范围或新增 ADR

- 无法可靠阻止 `/library` 或媒体库根目录之外的读取。
- SQLite 在已确认目标规模上无法同时满足数据正确性与可接受交互，且调优后仍失败。
- 目标架构无法提供一致的 libvips/FFmpeg 能力或安全更新路径。
- 认证未实现且部署方案要求直接公网暴露。
- 核心依赖存在无法接受的许可证、专利、供应链或未修复高危漏洞问题。

触发 No-Go 不等同于放弃项目。优先缩小格式、规模、架构或网络暴露范围；若需要引入外部数据库、独立 worker 或改变路径模型，必须先新增 ADR。
