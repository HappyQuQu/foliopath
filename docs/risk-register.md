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
| R-010 | 认证未完成的预览版或失效的稳定版认证被暴露到不可信网络 | 高 | 严重 | Compose 改为全网绑定、代理绕过、会话/CSRF 缺陷或公开扫描发现端口 | 预览版默认回环；认证已 Integrated Done。S5-003 又固定显式可信 CIDR、非回环 require-proxy、严格单跳 HTTPS transport、伪造/链式头失败关闭、Secure Cookie/Origin/HSTS 与客户端限流共享 transport。实际 Compose 仍须用回环端口/私有网络/firewall 阻止代理旁路，并由 S5-001B/002 重复验证 | 立即撤销公网暴露，只允许本机/可信隧道访问 | 安全负责人 | 缓解中 |
| R-011 | 自动迁移、升级或备份恢复导致配置丢失 | 中 | 高 | 迁移中断、只复制主 DB、恢复版本不兼容 | 只追加迁移；[S5-004](gates/MVP-2026-07-23/s5-recovery-failure-smoke.md) 已在原生 amd64/arm64 以不同前一候选 image ID 验证向前升级、旧镜像＋升级前离线备份配对回滚、完整 SQLite family 恢复、WAL 强杀恢复、缓存重建及满盘/损坏失败关闭 | 回到备份对应版本恢复，再从原媒体重建派生数据 | 发布负责人 | 已关闭 |
| R-012 | Unicode、大小写和文件系统差异造成重复路径或搜索错误 | 中 | 高 | macOS/Linux 结果不同、唯一约束冲突、不可定位文件 | S2-007 已确认媒体库显示名 NFC、唯一键 NFKC + Unicode full case folding、locale-neutral numeric sort key 与组件级根比较；[S4-002](gates/MVP-2026-07-23/s4-search-keyset.md)已用组合字符、sharp-s、全角、中文、保留变音符号、短词和 `%/_` fixture 固定语义，[S4-003](gates/MVP-2026-07-23/s4-search-backend-ready.md)又在 macOS/Linux 100k 档验证同一派生键、FTS 和 fallback 路径。真实多种文件系统仍留给 Release Gate | 保留规范显示名并拒绝歧义库，搜索实现不满足 profile 时停止 Backend Gate；必要时缩小受支持文件系统范围 | 后端负责人 | 缓解中 |
| R-013 | 多媒体库共享任务队列时出现饥饿或请求被后台工作拖慢 | 中 | 高 | 一个大库占满 worker；API 延迟随扫描显著上升 | S2/S3 已固定公平/资源边界；[S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 关闭 session touch、unchanged 派生和 cache 淘汰写放大，原生 100k 扫描并发浏览 P95 降至 6.116 ms，94,234 个后台任务调度期间重扫 140.190 s，全部 100,000 个媒体任务最终成功，三引擎 FPS/RSS 通过 | 暂停/限速后台任务，改为手动扫描模式 | 技术负责人 | 已关闭 |
| R-014 | 依赖许可证、FFmpeg 构建选项或 codec 分发义务不清 | 中 | 高 | SBOM 出现不兼容/未知许可证；无法提供对应源码或 notice | [供应链审查](supply-chain-review.md) 的 Stage 0 基线确认 Debian FFmpeg 启用 GPL、libx264/libx265；[S5-007D](gates/MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md) 已以关闭 GPL/x264/网络的最小 LGPL 2.1+ 构建替换它并保留许可证。最终双平台 digest 仍须附 SBOM/provenance、notices、漏洞结果和合规签署 | 替换依赖、关闭相关 codec 或停止分发受影响镜像 | 合规负责人 | 缓解中 |
| R-015 | 可选虚拟瀑布流破坏键盘顺序、焦点或大列表稳定性 | 中 | 中 | DOM 顺序与视觉顺序不一致、焦点丢失、滚动跳动 | 默认使用规则网格；[Stage 3 Integrated Done](gates/MVP-2026-07-23/s3-browse-integrated-done.md) 已验证 grid/masonry 记忆、DOM 顺序、固定占位、虚拟容量和返回焦点，S5-005 已通过三引擎 100k DOM/FPS/RSS，S5-006A 又执行查看器键盘/焦点矩阵；最终物理辅助功能签署仍待 S5-006B | 保留默认网格并临时禁用未达标的瀑布流 | 前端负责人 | 缓解中 |
| R-016 | 缺少 CI、生成漂移检查和可复现 fixture 导致回归 | 高 | 高 | 开发者本地通过而干净环境失败；生成文件与源不一致 | 真实 PR CI 已覆盖双架构 Go/race、生成/兼容、mount、govips/FFmpeg、runtime/recovery 与 SBOM/license；媒体库 Backend Gate、S2-102～106 worker、目录、媒体收敛、故障恢复和容量均有分层证据，容量主档另由 Linux amd64/arm64 受限资源 job 强制；每个后续生产切片继续增加集成/E2E 和已许可 fixture | 阻止合并和发布，先恢复最小验证基线 | QA 负责人 | 缓解中 |
| R-017 | 发布镜像的原生依赖闭包含未处置的 Critical/High 漏洞 | 高 | 严重 | 最终 SBOM/扫描出现 Critical/High；漏洞数据库变化 | [S5-007A](gates/MVP-2026-07-23/s5-supply-chain-candidate.md) 已固定 Syft/Trivy digest、保存完整报告并拒绝已有修复版本的发现；最小 libvips/FFmpeg、内建健康检查、无 shell distroless final stage 与固定源码 Expat 2.8.2 将发现从 15 Critical / 136 High 降至 1 / 8。剩余发现全部保留为 RC 阻断；发布前升级/移除受影响依赖或对具体 CVE 完成限时正式接受，并在最终双架构 digest 上执行全阻断扫描 | 跟踪 GLib/blkid/mount 修复；无法证明安全时停止发布 | 安全负责人 | 开放 |

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
- R-017 候选镜像未处置的 Critical/High 漏洞。

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

2026-07-28 的 [S5-009A 当前 RC 判断](gates/MVP-2026-07-23/s5-release-candidate-current.md)
已把八个前置 Gate 与八项发布阻断风险聚合到
[`MVP-2026-07-23-rc-readiness.json`](releases/MVP-2026-07-23-rc-readiness.json)。
当前五项发布风险为“缓解中”、R-008/R-011 为“已关闭”、R-017 为“开放”，没有“已接受”项，因此
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
