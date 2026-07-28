# FolioPath 可行性研究

## 结论与状态

FolioPath 在既定范围内具备实现可行性，**Stage 0 Gate 已通过并允许进入 Stage 1**。
FS-01 路径边界、FS-02 SQLite/generation、FS-03 原生双架构媒体链路、FS-04 Stage 0
容量范围、FS-05 原生双架构运行/恢复，以及 source/npm/image SBOM 与关键许可证识别均有
可复现证据。完整媒体、浏览器、生产容量、认证、真实升级和最终发布镜像仍由各自后续 Gate
阻断；本结论只授权后端优先的 Stage 1，不是发布承诺。

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
| 扫描与性能 | FS-04/S2-106/S4-003 已验证 100k/10k 扫描、搜索与恢复；S5-005 又在本机 arm64 和指定原生 amd64 候选完成正式 HTTP、100k 全量派生、cache 水位、内存/DB/cache 及三引擎 FPS/RSS | 数字是代表性设备回归护栏而非用户 SLA；未承诺任意 NAS 文件系统或网络盘性能 | 发布前在目标版本复跑冻结预算；超限必须修复或正式缩小支持边界 | `S5-005` Go；目标容量风险已关闭 |
| 媒体格式 | 确认 JPEG/PNG/WebP/GIF 图片与 MP4/MOV/MKV 视频；S3-004/005 已实现 production adapter、durable queue 和缓存保护；S3-006 已固定输入/像素/维度/工具输出、native/FFmpeg 并发与进程树取消，并以真实 tmpfs 验证 `ENOSPC` 清理恢复 | 进程内 libvips 无法在任意 C 调用中间抢占或隔离 native crash；ICC、浏览器直放、代表性存储峰值和最终镜像尚无完整实测 | 当前限制 native concurrency 并在求值前拒绝超限输入；媒体 UI 与 Release Gate 验证浏览器、存储和最终镜像，改变进程隔离边界须 ADR | FS-03、S3-004～006 范围满足；浏览器/发布矩阵仍 Conditional |
| SQLite | 真实文件数据库、Goose migration 1～10、WAL、generation、原子 finalize、FTS5/规范搜索键旧库回填、100k 扫描并发查询、启动 integrity/rebuild/cancel/fail-closed、离线备份/恢复、重复 migration 与满盘/损坏失败关闭已验证 | 真实强杀、长期 checkpoint、在线备份、代表性存储和真实已发布版本升级未知 | Release Gate 继续故障、长期运行和升级演练 | FS-02/05 与 S4-003 对应范围通过；`/app/data` 仍禁止放在 SMB/NFS |
| 安全 | 已定义 `/library` 边界；Darwin 与原生 Linux amd64/arm64 路径矩阵、Linux `openat2` mount 拒绝、HTTP harness 与脱敏已有自动化证据 | 生产 handler/OpenAPI、认证 envelope、只读发布 volume、运行期 unmount、解析器资源上限和认证尚未验证 | 在首个受保护 API Backend Gate 完成生产 HTTP/auth，在 FS-05/Release Gate 完成发布容器，再验证媒体炸弹与认证 | FS-01 Stage 0 范围通过；后续安全要求仍分别阻断 Backend/Release Gate |
| 运维 | FS-05 已验证单容器 probe、非 root、health、优雅退出、离线恢复和故障关闭 | 正式应用观测、缓存配额、在线备份、真实升级与 NAS 断连未实现 | 在 S1/对应 Backend/Release Gate 用正式应用复验 | Stage 0 运行模式可行，不等于可发布 |
| 跨架构 | 相同 govips/FFmpeg fixture 与 FS-05 Dockerfile 已在原生 amd64/arm64 通过 | 最终 manifest、发布 digest 和持续安全更新仍待验证 | Release Gate 对最终 digest 重跑矩阵 | 可行但不是“纯 Go”交叉编译 |
| 许可证 | source/npm/image SPDX 可重复生成；libvips 为 LGPL-2.1，Debian FFmpeg 启用 GPL、libx264/libx265 | 最终 notices、source offer、漏洞和签署尚未完成 | 对最终双平台 digest 附 SBOM/provenance 并完成合规签署 | Stage 0 未发现换栈阻断，发布前仍必须审查 |

## 当前 spike 状态

| Spike | 状态 | 范围与产物 | 通过条件 / 剩余门槛 |
| --- | --- | --- | --- |
| [FS-01 路径边界](spikes/fs-01-path-boundary.md) | **Passed（Stage 0 范围）** | Darwin 与原生 Linux amd64/arm64 安全目录/文件打开；Linux `openat2` mount 拒绝；HTTP harness 覆盖 ID、Range、条件请求、攻击矩阵与脱敏 | 生产 handler/auth 转入首次受保护 API Backend Gate；发布 volume/unmount 转入 FS-05/Release Gate |
| [FS-02 SQLite 与扫描](spikes/fs-02-sqlite-generation.md) | **Passed（当前正确性 scope）** | 真实文件 SQLite、嵌入迁移、generation、批量 upsert、故障/取消/离线/重启保留和原子成功清理 | 当前 scope 通过；磁盘满、真实 `SIGKILL`、长期 WAL 压力、SMB/NFS 与备份恢复属于后续门槛 |
| [FS-03 媒体矩阵](spikes/fs-03-media-matrix.md) | **Passed（Stage 0 范围）；完整范围 Conditional** | FFmpeg 合成矩阵和原生双架构 govips/libvips 元数据、方向、alpha、动画首帧、有界缩略图与截断拒绝 | 任务超时/隔离、更多敌意输入、浏览器直放及最终镜像由后续 Gate 强制 |
| [FS-04 容量基线](spikes/fs-04-capacity-baseline.md) | **Passed（扫描与搜索后端范围）；完整范围 Conditional** | Linux/arm64、2 CPU/4 GiB、10 万媒体/1 万目录混合宽度/32 层深度；另有 1,000 层 finalize、Linux RSS、三档趋势；S4-003 已补扫描并发搜索、FTS/短词/全局/keyset、取消与 rebuild 强制预算 | 代表性磁盘/最终镜像、生产媒体队列、HTTP/前端并发继续按 [S0-106](gates/MVP-2026-07-23/s0-106-capacity-gate-order.md)由后续 Gate 强制 |
| [FS-05 镜像与恢复](spikes/fs-05-runtime-recovery.md) | **Passed（Stage 0 范围）** | 原生双架构同 Dockerfile；非 root/只读、health、退出、离线恢复、重复 migration 和满盘/损坏失败关闭 | 正式应用、在线备份、真实版本升级、NAS 断连和最终 manifest 由后续 Gate 强制 |

Spike 应提交可复现命令、fixture 说明、执行环境、未执行项和结论。若硬件、数据集或构建选项不同，不得横向比较为同一 benchmark。FS-04 的 Docker Desktop/tmpfs 数据只描述当前扫描/索引子范围，不能泛化为代表性 NAS、完整产品吞吐或发布预算。

SQLite WAL 只适用于具备正确共享内存、锁和同步语义的同机文件系统；WAL 模式必须检查 `PRAGMA journal_mode=WAL` 的实际返回值，且 `-wal` 文件属于持久状态的一部分。当前实现和后续运维验证均以 [SQLite WAL documentation](https://sqlite.org/wal.html) 为基线。

## Go / No-Go 门槛

### 允许进入 MVP 实施

- [需求确认清单](requirements-checklist.md)中的 RQ-001～RQ-014 已全部确认 A，并已同步为产品基线。
- FS-01 与 FS-02 均满足其 Stage 0 范围；生产 HTTP/auth 和发布容器证据按
  [S0-105](gates/MVP-2026-07-23/s0-105-gate-order.md)分别由后续 Backend/Release Gate 强制。
- FS-03 Stage 0 媒体支持矩阵与原生双架构链路通过；完整产品证据已分配后续 Gate。
- 性能目标、目标硬件和数据规模已经定义；FS-04 没有发现必须改写架构的阻断问题。
  Stage 0 容量范围已通过；生产与代表性设备证据按
  [S0-106](gates/MVP-2026-07-23/s0-106-capacity-gate-order.md) 分配到后续 Gate。
- FS-05 已证明目标架构可构建、可启动、可备份和可恢复。
- Stage 0 供应链审查未发现换栈阻断项；最终分发义务继续阻断 Release Gate。

### 必须暂停、缩减范围或新增 ADR

- 无法可靠阻止 `/library` 或媒体库根目录之外的读取。
- SQLite 在已确认目标规模上无法同时满足数据正确性与可接受交互，且调优后仍失败。
- 目标架构无法提供一致的 libvips/FFmpeg 能力或安全更新路径。
- 认证未实现且部署方案要求直接公网暴露。
- 核心依赖存在无法接受的许可证、专利、供应链或未修复高危漏洞问题。

触发 No-Go 不等同于放弃项目。优先缩小格式、规模、架构或网络暴露范围；若需要引入外部数据库、独立 worker 或改变路径模型，必须先新增 ADR。
