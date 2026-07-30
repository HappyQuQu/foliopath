# FolioPath 风险登记册

## 使用方式

本表用于开发前和每个阶段出口评审。概率与影响是当前缺少实现证据时的定性判断，不是统计结果。RQ-001～RQ-014 已全部确认 A；新的产品变更统一进入[需求确认清单](requirements-checklist.md)，验证计划见[可行性研究](feasibility-study.md)。

状态含义：`开放` 表示尚未验证；`缓解中` 表示已有实施或 spike；`已接受` 必须有明确决策者和剩余影响；`已关闭` 必须附验证证据。Owner 为负责推动决策的角色，不代表单人独立完成全部工作。

| ID | 风险 | 概率 | 影响 | 触发信号 | 缓解措施 | Fallback | Owner 角色 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| R-001 | MVP 需求持续变化导致架构和 UI 返工 | 中 | 高 | 已确认基线被重新打开；同一行为在文档中定义不一 | RQ-001～014 已全部确认 A；变更必须留新决策记录，纵向切片前先做 spike | 缩减为创建媒体库、扫描和目录浏览的最小闭环 | 产品负责人 | 缓解中 |
| R-002 | 路径遍历、编码绕过或符号链接导致越界读取 | 中 | 严重 | 安全测试能读取媒体库或 `/library` 之外路径；路径校验散落多个包 | 所有真实路径访问集中到 `internal/files`；[FS-01](spikes/fs-01-path-boundary.md) 与[媒体库 Backend Ready](gates/MVP-2026-07-23/s2-library-backend-ready.md)已验证 Darwin/Linux amd64/arm64、`openat2`、认证 path/create handler、恶意编码、symlink、同/跨设备与 self-bind mount、TOCTOU/ABA、权限及错误脱敏。扫描/媒体读取 handler、只读发布 volume 和运行期 unmount 仍由后续 Gate 验证 | 停止发布并禁用相关入口；不以 UI 隐藏代替修复 | 安全负责人 | 缓解中 |
| R-003 | 离线、权限失败或中断扫描误清理索引 | 中 | 严重 | 失败代次删除旧记录；挂载消失后资产数骤降 | [FS-02](spikes/fs-02-sqlite-generation.md)、[S2-103](gates/MVP-2026-07-23/s2-directory-counts.md)、[S2-104](gates/MVP-2026-07-23/s2-media-convergence.md)、[S2-105](gates/MVP-2026-07-23/s2-scan-recovery.md) 与 [S2-106](gates/MVP-2026-07-23/s2-scan-capacity.md) 已验证 generation 仅成功时原子清理；失败、取消、offline、部分不可读、nested mount、根替换、重启和 SQLite 满页保留可靠代次，后续完整扫描可恢复收敛。生产强杀与代表性存储故障仍待 Release Gate | 禁用自动陈旧清理，保留索引并要求完整重扫 | 后端负责人 | 缓解中 |
| R-004 | `/app/data` 位于 SMB/NFS，SQLite WAL 锁或同步失效 | 中 | 高 | `SQLITE_BUSY`、损坏、checkpoint 异常或恢复后不一致 | 文档强制本地文件系统；[FS-02](spikes/fs-02-sqlite-generation.md) 已验证 WAL/checkpoint，[FS-05](spikes/fs-05-runtime-recovery.md) 已验证完整卷离线备份/恢复；SMB/NFS 继续明确为不支持，不以网络盘实验放宽边界 | 停机迁移数据目录到本地盘后重建派生索引 | 运维负责人 | 缓解中 |
| R-005 | 10 万媒体/1 万目录目标超出四核、4 GiB 环境的扫描、查询或内存能力 | 高 | 高 | 队列持续增长、首次扫描不可接受、列表延迟或 RSS 超预算 | [S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 已在本机原生 arm64 与指定原生 amd64 4 CPU/4 GiB 候选完成 100k/10k 扫描、查询、重扫、取消/offline、100k 全量派生、内存/DB/cache 及三引擎 FPS/RSS，并冻结回归护栏，所有指标通过 | 降低并发/缩略图规格，声明实测支持上限；架构变更须 ADR | 技术负责人 | 已关闭 |
| R-006 | 畸形媒体触发 libvips/FFmpeg 崩溃或资源耗尽 | 中 | 严重 | 解析超时、OOM、子进程堆积或相同文件无限重试 | [FS-03](spikes/fs-03-media-matrix.md)、[S3-004](gates/MVP-2026-07-23/s3-media-processing.md)、[S3-005](gates/MVP-2026-07-23/s3-media-jobs-cache.md) 与 [S3-006](gates/MVP-2026-07-23/s3-media-resource-safety.md) 已验证截断/像素炸弹拒绝、256 MiB/4 GiB/100 MP/32,768 px 上限、govips concurrency/cache、FFmpeg 单线程/进程组取消/输出 cap、2-worker/3-attempt 退避。进程内 libvips native crash 隔离和代表性峰值仍由 Release Gate/未来 ADR 复审 | 隔离失败资产；临时禁用受影响格式或视频封面 | 媒体处理负责人 | 缓解中 |
| R-007 | 已确认的 JPEG/PNG/WebP/GIF 或 MP4/MOV/MKV 契约在目标架构/浏览器表现不一致 | 高 | 中 | 同扩展名行为不一致；海报可生成但浏览器播放失败 | [FS-03](spikes/fs-03-media-matrix.md) 的相同 govips/FFmpeg fixture 已在原生双架构通过；仍区分服务端可处理与浏览器可直放，并在查看器 Gate 验证浏览器 | 明确标记不可播放；MVP 不加入隐式转码 | 产品负责人 | 缓解中 |
| R-008 | amd64 与 arm64 的 govips/FFmpeg 构建或功能不一致 | 中 | 高 | 某架构构建失败、缺少 codec、fixture 输出或崩溃行为不同 | 原生双架构 govips/FFmpeg fixture 与 [FS-05](spikes/fs-05-runtime-recovery.md) 同 Dockerfile runtime/recovery smoke 已通过；[S5-002](gates/MVP-2026-07-23/s5-compose-candidate-matrix.md) 又在本机原生 arm64 和操作者指定的原生 amd64 服务器通过完整候选媒体/Compose/恢复矩阵，并审阅相同嵌入 SPA 与运行包闭包 | 若需求允许先缩减发布架构；否则阻断发布 | 发布负责人 | 已关闭 |
| R-009 | 10 GiB 缩略图缓存、临时文件或 WAL 仍耗尽磁盘 | 中 | 高 | 可用空间持续下降、LRU 无法维持余量、临时文件残留或写事务失败 | [S3-005](gates/MVP-2026-07-23/s3-media-jobs-cache.md) 与 [S3-006](gates/MVP-2026-07-23/s3-media-resource-safety.md) 已验证 90%→80% LRU、512 MiB 余量和真实 `ENOSPC` 恢复；[S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 又在 100k 全量派生后把在线降配缓存精确收敛至 3,355,440 B 低水位，0 pending deletion、应用持续健康并关闭淘汰写竞争/写放大 | 暂停派生任务并清理可重建缓存，优先保护数据库 | 运维负责人 | 已关闭 |
| R-010 | 经认证的 LAN HTTP 被误暴露到公网或不可信网络 | 高 | 严重 | 路由器端口转发、云安全组放行、认证/CSRF 缺陷或错误代理信任范围 | 内建单管理员认证、业务 API 默认认证、CSRF、SameSite/HttpOnly Cookie；直连模式清除转发头并按 peer 限流；显式代理模式保持严格单跳 HTTPS、Secure Cookie 与 HSTS。文档明确 LAN HTTP 不提供链路机密性，公网由部署者提供 TLS/ACL | 撤销外部端口映射，收窄 `FOLIOPATH_BIND_ADDRESS`，必要时置于外部 TLS 代理后 | 安全负责人 | 缓解中 |
| R-011 | 自动迁移、升级或备份恢复导致配置丢失 | 中 | 高 | 迁移中断、只复制主 DB、恢复版本不兼容 | 只追加迁移；[S5-004](gates/MVP-2026-07-23/s5-recovery-failure-smoke.md) 已在原生 amd64/arm64 以不同前一候选 image ID 验证向前升级、旧镜像＋升级前离线备份配对回滚、完整 SQLite family 恢复、WAL 强杀恢复、缓存重建及满盘/损坏失败关闭 | 回到备份对应版本恢复，再从原媒体重建派生数据 | 发布负责人 | 已关闭 |
| R-012 | Unicode、大小写和文件系统差异造成重复路径或搜索错误 | 中 | 高 | macOS/Linux 结果不同、唯一约束冲突、不可定位文件 | S2-007 已确认媒体库显示名 NFC、唯一键 NFKC + Unicode full case folding、locale-neutral numeric sort key 与组件级根比较；[S4-002](gates/MVP-2026-07-23/s4-search-keyset.md)已用组合字符、sharp-s、全角、中文、保留变音符号、短词和 `%/_` fixture 固定语义，[S4-003](gates/MVP-2026-07-23/s4-search-backend-ready.md)又在 macOS/Linux 100k 档验证同一派生键、FTS 和 fallback 路径。真实多种文件系统仍留给 Release Gate | 保留规范显示名并拒绝歧义库，搜索实现不满足 profile 时停止 Backend Gate；必要时缩小受支持文件系统范围 | 后端负责人 | 缓解中 |
| R-013 | 多媒体库共享任务队列时出现饥饿或请求被后台工作拖慢 | 中 | 高 | 一个大库占满 worker；API 延迟随扫描显著上升 | S2/S3 已固定公平/资源边界；[S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 关闭 session touch、unchanged 派生和 cache 淘汰写放大，原生 100k 扫描并发浏览 P95 降至 6.116 ms，94,234 个后台任务调度期间重扫 140.190 s，全部 100,000 个媒体任务最终成功，三引擎 FPS/RSS 通过 | 暂停/限速后台任务，改为手动扫描模式 | 技术负责人 | 已关闭 |
| R-014 | 依赖许可证、FFmpeg 构建选项或 codec 分发义务不清 | 中 | 高 | SBOM 出现不兼容/未知许可证；无法提供对应源码或 notice | [供应链审查](supply-chain-review.md) 的 Stage 0 基线确认 Debian FFmpeg 启用 GPL、libx264/libx265；[S5-007D](gates/MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md) 已以关闭 GPL/x264/网络的最小 LGPL 2.1+ 构建替换它并保留许可证。最终双平台 digest 仍须附 SBOM/provenance、notices、漏洞结果和合规签署 | 替换依赖、关闭相关 codec 或停止分发受影响镜像 | 合规负责人 | 缓解中 |
| R-015 | 可选虚拟瀑布流破坏键盘顺序、焦点或大列表稳定性 | 中 | 中 | DOM 顺序与视觉顺序不一致、焦点丢失、滚动跳动 | 默认使用规则网格；[Stage 3 Integrated Done](gates/MVP-2026-07-23/s3-browse-integrated-done.md) 已验证 grid/masonry 记忆、DOM 顺序、固定占位、虚拟容量和返回焦点，S5-005 已通过三引擎 100k DOM/FPS/RSS，S5-006A 又执行查看器键盘/焦点矩阵；最终物理辅助功能签署仍待 S5-006B | 保留默认网格并临时禁用未达标的瀑布流 | 前端负责人 | 缓解中 |
| R-016 | 缺少 CI、生成漂移检查和可复现 fixture 导致回归 | 高 | 高 | 开发者本地通过而干净环境失败；生成文件与源不一致 | 真实 PR CI 已覆盖双架构 Go/race、生成/兼容、mount、govips/FFmpeg、runtime/recovery 与 SBOM/license；媒体库 Backend Gate、S2-102～106 worker、目录、媒体收敛、故障恢复和容量均有分层证据，容量主档另由 Linux amd64/arm64 受限资源 job 强制；每个后续生产切片继续增加集成/E2E 和已许可 fixture | 阻止合并和发布，先恢复最小验证基线 | QA 负责人 | 缓解中 |
| R-017 | 发布镜像的原生依赖闭包含未处置的 Critical/High 漏洞 | 高 | 严重 | 最终 SBOM/扫描出现 Critical/High；漏洞数据库变化 | [S5-007A/G](gates/MVP-2026-07-23/s5-supply-chain-candidate.md) 已固定 Syft/Trivy digest和完整报告；最小 libvips/FFmpeg、内建健康检查、无 shell distroless、固定源码 Expat 与修复来源 GLib 2.88.3 将本机 arm64 候选从 15 Critical / 136 High 降至 `0 / 0`，并移除 GLib 的 libmount/blkid 间接闭包。干净候选提交 `5c3b3c7` 的原生 arm64 已以相同结果生成 smoke、SPDX、notices、provenance 和不可变 digest 证据；原生 amd64 配对复扫及安全/合规签署仍缺失 | 若任一最终 digest 重新出现发现，升级/移除依赖或停止发布 | 安全负责人 | 缓解中 |
| R-018 | 视频故事板 backfill、重复 seek 或 hover 生命周期造成 CPU、I/O、缓存、请求或前端资源失控 | 中 | 高 | grid/poster 等待变长；浏览 P95 上升；队列/缓存持续增长；快速掠过产生请求风暴；虚拟卡片回收后 timer/动画仍活动 | [VSP-S2 Backend Evidence Ready](gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md)已以双架构生产 FFmpeg、单并发/低优先级、128 项 admission、真实 cache repair 和 Linux 100k/10k 档关闭后端 Gate；[VSP-S3 Consumer/UI Ready](gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md)又以 300ms intent、按需 decode、单活动动画、生命周期回收、六种浏览器/输入 profile 和 100-video 三引擎容量关闭前端 Gate；[VSP-301](gates/POST-MVP-1/vsp-301-product-vertical.md)已贯通生产镜像真实全链，剩余风险由 [VSP-302](gates/POST-MVP-1/vsp-302-target-platform.md) 原生候选复验与 VSP-S4 最终签署阻断 | 依次降低 worker/尺寸/质量、改为有界按需 admission；仍不满足时禁用 storyboard 并保留现有 poster | 媒体处理与前端性能负责人 | 缓解中 |
| R-019 | 文件事件丢失、乱序、overflow、网络盘不转发、watch 资源不足或删除误判导致内容延迟或索引损失 | 高 | 严重 | `IN_Q_OVERFLOW`/`ENOSPC`、watch 被移除、事件后目录不一致、挂载掉线时出现批量 delete、dirty 队列持续满或页面长期不更新 | [FTR-SCN-001](features/automatic-library-discovery.md)要求事件只作不可信提示；新增/修改/删除均经 `internal/files` 安全定向校准确认，掉盘/权限/overflow 保留可靠索引并合并安排完整扫描；每目录而非每文件 watch，队列/并发/revision 轮询有界；实现前必须通过 WCH-S0 ADR 与 Linux 100k/10k burst/恢复证据 | 先扩大合并并降低定向并发，再按库进入 degraded；仍不可靠时全局禁用自动发现，保留创建/启动/手动/定时完整扫描 | 扫描与运维负责人 | 开放 |
| R-020 | 任务中心全量重建造成无界 admission、队列饥饿、磁盘耗尽，或先清缓存导致可用预览丢失 | 中 | 高 | rebuild 一次登记全部资产；日常 poster/grid 或浏览延迟持续恶化；取消后队列继续增长；ENOSPC 后旧 ready 不可用 | [FTR-OPS-001](features/task-center.md)要求 parent run、asset keyset 小批 admission、最低后台优先级、active coalesce、磁盘安全余量、停止 admission 的协作取消和新文件成功后才替换旧缓存；OPS-003 必须在 100k/10k 档冻结上限 | 只保留 missing backfill，禁用 all rebuild；必要时完全隐藏批量入口并继续现有按需 self-heal | 媒体处理与性能负责人 | 开放 |
| R-021 | 原型、共享 token、生产页面和视觉基线各自演进，导致跨页漂移或为追求像素一致破坏真实功能、可访问性和大列表能力 | 高 | 高 | 同一控件出现多份样式；批量接受截图变化；页面只在 1440px 正常；为匹配原型改用 mock、全量客户端过滤或嵌套滚动 | [FTR-UIF-001](features/frontend-prototype-fidelity.md)固定视觉/功能双真相与唯一 shared owner；[UIF-S4](gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md)已接受 12 页共同 1280、12 页 × 4 断点、Linux 基线、真实 API、三浏览器/axe/输入、100k/10k 和跨文档收敛证据；没有以 mock、无界加载或嵌套滚动换取截图一致 | 保留已验证生产页面和机器 reference manifest；任何基线变化须解释来源并重跑真实链，不能以静态原型、mock 或批量 snapshot 更新替代 | 前端、设计系统与 QA 负责人 | 已关闭 |

Stage 4 媒体内容风险更新：S4-005B 已用真实认证 composition、poisoned catalog path、
source fingerprint 变化、missing/offline、Range/取消/有界 admission 和 Linux arm64
`openat2` nested-mount fixture 缓解 R-002/R-006/R-012/R-016。amd64 QEMU 因缺少所需
`openat2` 能力按设计失败关闭；这不是 native amd64 通过证据，仓库 billing 恢复后必须重跑
PR native job，Stage 5 仍阻断正式只读 volume、运行期 unmount、浏览器与发布镜像。

## 发布阻断风险

以下风险在首个可用镜像发布前不得保持“开放”且无证据：

- R-002 路径边界、R-003 扫描清理和 R-006 不可信媒体处理。
- R-010 网络暴露，以及已确认单管理员认证的实现与验证。
- R-011 备份/恢复和迁移。
- R-008 中实际承诺发布的架构。
- R-014 许可证与分发义务。
- R-017 最终原生双架构镜像尚未从干净提交完成全阻断复扫与安全签署。

R-005、R-008、R-009、R-011 与 R-013 已由 Stage 5 容量、双架构和恢复证据关闭；
R-002～004、R-006～007、R-010、R-012、R-014～016 仍处于缓解中。但其余项目对应的
报告列出的生产 HTTP/认证错误边界、只读发布挂载与运行期 unmount、真实强杀与生产磁盘
策略、媒体任务隔离、浏览器、代表性存储、最终镜像 SBOM/attestation、真实版本升级与发布
签署仍未完成；原生双架构 CI、真实 PR 基线兼容比较、TypeScript 生成和摘要锁降低了工程
风险，但不等于发布风险关闭。
不得把产品确认或局部实现误写成完整技术可行性证明。

R-002～R-016 的逐项 Stage 0 Owner、Fallback 和最迟阻断 Gate 见
[S0-109 风险复审](gates/MVP-2026-07-23/s0-109-risk-review.md)。
R-017 是 Stage 5 候选扫描新增风险，其 owner、fallback 与 Release Candidate 阻断证据
由 [S5-007 候选镜像供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)
维护。
R-018 属于 `Post-MVP/1` 的
[FTR-VID-001](features/video-storyboard-preview.md)，不改变当前 MVP RC 判断；它在未来版本
中阻断 `VSP-S4`。后端容量、前端生命周期与 VSP-301 产品纵向证据已完成；在 VSP-302
原生双架构候选复验和最终 Gate 签署前不得作为稳定能力发布。
R-019 属于已冻结 `Post-MVP/2` 的
[FTR-SCN-001](features/automatic-library-discovery.md)，不改变当前 MVP RC 判断；在 watcher
ADR、增量任务/删除资格合同、Linux overflow/掉盘/强杀/容量证据和 `WCH-S4` 签署前保持开放。
R-020 属于 `Post-MVP/3` scope proposed 的
[FTR-OPS-001](features/task-center.md)，不改变当前 MVP RC 判断；它阻断 `OPS-S0/S2/S4`，
在 100k admission、优先级、取消、盘满、重启和旧 ready 保留证据完成前保持开放。
R-021 属于当前 MVP revision 4 的
[FTR-UIF-001](features/frontend-prototype-fidelity.md)；UIF-S1/S2/S3 已依次用真实合同、
账户/目录/cache 后端、机器 reference manifest、四档状态矩阵、三浏览器/可访问性与 100k
有界 DOM 关闭对应子风险；UIF-401/402/403 又完成逐页同状态比较、Linux-owned 基线和真实
容器纵向链，并验证 cache cleanup 与媒体路径/SHA-256 不变；UIF-404 已通过三引擎、axe、
键盘、触摸、forced-colors 与 reduced-motion 自动化适用复验，并把真实读屏、缩放、OS
高对比和物理触摸明确留给 S5-006B；UIF-405 又在最新共享集合上通过三引擎 100k
滚动/DOM/FPS/RSS，并以 10k/100k 完整扫描期浏览/搜索并发和跨库 worker 顺序复验后端
容量边界；UIF-406 的 fmt、architecture、generation、lint、unit、integration 与生产容器
E2E 七项完整验证全部通过；UIF-407 又把 PRD、UI、流程、API/data/security/testing、
deployment、traceability、风险、README 和 release 状态统一到实际证据，并明确 12 页
共同 1280 逐页比较与四档响应式矩阵不是同一证据；UIF-408 随后补齐 12 页 × 4 断点的
48 张原型图、48 张真实生产图和 12 张成对审阅图，并重跑受影响的浏览器、容量、容器和 RC
readiness 检查。[UIF-S4](gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md)据此为 Go，
R-021 当前 feature 风险关闭；未来改动继续由 reference manifest 和视觉回归防止重新引入。

2026-07-28 的 [S5-009A 当前 RC 判断](gates/MVP-2026-07-23/s5-release-candidate-current.md)
已把八个前置 Gate 与八项发布阻断风险聚合到
[`MVP-2026-07-23-rc-readiness.json`](releases/MVP-2026-07-23-rc-readiness.json)。
当前六项发布风险为“缓解中”、R-008/R-011 为“已关闭”，没有“开放”或“已接受”项，但
缓解中风险仍未达到发布关闭条件，因此
Release Candidate 明确为 No-Go。

同日对当前已提交 HEAD 的 GitHub Actions run `30314930003` 复核发现，全部 job 因账户
付款失败或 spending limit 在 runner 分配前失败，没有任何测试 step。`S5-001C` 已为后续
原生候选 job 增加绑定 commit、run、architecture、digest 和 smoke result 的成对 JSON
证据检查，降低后续 R-008/R-016 的结果误配风险。操作者已指定原生 amd64 服务器替代本轮
计费阻断的 CI 运行，因此 `R-008` 由实际双架构矩阵关闭；Billing 仍应恢复，但不再作为
本轮产品运行风险。

## 评审规则

- 阶段入口检查新风险、Owner 和触发信号；阶段出口检查缓解证据与剩余风险。
- 概率或影响上升到“严重”时，立即停止相关范围扩张并由技术/产品负责人共同决定继续、降级或 No-Go。
- 风险关闭需链接测试、spike 结果、ADR 或发布演练记录；只有口头判断不能关闭风险。
- 接受剩余风险必须写明适用版本、部署范围和复审条件，不能永久泛化。
