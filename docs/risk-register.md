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
| R-005 | 10 万媒体/1 万目录目标超出四核、4 GiB 环境的扫描、查询或内存能力 | 高 | 高 | 队列持续增长、首次扫描不可接受、列表延迟或 RSS 超预算 | [FS-04](spikes/fs-04-capacity-baseline.md) 与 [S2-106](gates/MVP-2026-07-23/s2-scan-capacity.md) 已覆盖混合深度/宽度主档、1,000 层 finalize、并发索引读、RSS/heap/DB 预算和三档趋势；正式 256 batch 的 10k/100k 本地档无预算超限，Linux amd64/arm64 独立容量 job 已接入 CI。生产媒体、FTS/keyset、HTTP/前端和代表性存储仍按 S0-106 分配到后续 Gate | 降低并发/缩略图规格，声明实测支持上限；架构变更须 ADR | 技术负责人 | 缓解中 |
| R-006 | 畸形媒体触发 libvips/FFmpeg 崩溃或资源耗尽 | 中 | 严重 | 解析超时、OOM、子进程堆积或相同文件无限重试 | [FS-03](spikes/fs-03-media-matrix.md) 与 [S3-004](gates/MVP-2026-07-23/s3-media-processing.md) 已验证 production adapter 的截断输入拒绝、稳定错误、FFmpeg 超时/取消、输入/输出初始上限和原子派生发布；像素炸弹、原生调用不可中断区间、全局并发、退避与 durable 任务隔离仍由 S3-005/006 阻断 | 隔离失败资产；临时禁用受影响格式或视频封面 | 媒体处理负责人 | 缓解中 |
| R-007 | 已确认的 JPEG/PNG/WebP/GIF 或 MP4/MOV/MKV 契约在目标架构/浏览器表现不一致 | 高 | 中 | 同扩展名行为不一致；海报可生成但浏览器播放失败 | [FS-03](spikes/fs-03-media-matrix.md) 的相同 govips/FFmpeg fixture 已在原生双架构通过；仍区分服务端可处理与浏览器可直放，并在查看器 Gate 验证浏览器 | 明确标记不可播放；MVP 不加入隐式转码 | 产品负责人 | 缓解中 |
| R-008 | amd64 与 arm64 的 govips/FFmpeg 构建或功能不一致 | 中 | 高 | 某架构构建失败、缺少 codec、fixture 输出或崩溃行为不同 | 原生双架构 govips/FFmpeg fixture 与 [FS-05](spikes/fs-05-runtime-recovery.md) 同 Dockerfile runtime/recovery smoke 已通过；S3-004 production adapter 的 tagged govips 与真实 FFmpeg fixture 已接入原生依赖 CI，正式构建必须启用 `libvips` tag；最终发布 digest 继续运行相同矩阵 | 若需求允许先缩减发布架构；否则阻断发布 | 发布负责人 | 缓解中 |
| R-009 | 10 GiB 缩略图缓存、临时文件或 WAL 仍耗尽磁盘 | 高 | 高 | 可用空间持续下降、LRU 无法维持余量、临时文件残留或写事务失败 | 可配置 10 GiB 默认配额、清理水位、原子写入、启动清理、磁盘观测和磁盘满测试 | 暂停派生任务并清理可重建缓存，优先保护数据库 | 运维负责人 | 开放 |
| R-010 | 认证未完成的预览版或失效的稳定版认证被暴露到不可信网络 | 高 | 严重 | Compose 改为全网绑定、代理绕过、会话/CSRF 缺陷或公开扫描发现端口 | 预览版默认回环；S1-101～106 已完成冻结契约、密码/原子 setup、高熵摘要会话、绝对期限/撤销、真实 handler、Origin/CSRF、默认拒绝、防缓存、直连 peer 限流、错误/并发/时间安全矩阵和 Backend Ready。可信代理、非回环网络发布与浏览器 E2E 继续由 Stage 5 验收 | 立即撤销公网暴露，只允许本机/可信隧道访问 | 安全负责人 | 缓解中 |
| R-011 | 自动迁移、升级或备份恢复导致配置丢失 | 中 | 高 | 迁移中断、只复制主 DB、恢复版本不兼容 | 只追加迁移；[媒体库 Contract Ready](gates/MVP-2026-07-23/s2-library-contract-ready.md) 已验证真实 version 2→3 升级、事务回滚和约束；[FS-05](spikes/fs-05-runtime-recovery.md) 已验证停机后完整数据卷备份/恢复、重复 migration、只读/磁盘满/损坏 DB 失败关闭；真实已发布版本升级与在线备份仍由 Release Gate 演练 | 回到备份对应版本恢复，再从原媒体重建派生数据 | 发布负责人 | 缓解中 |
| R-012 | Unicode、大小写和文件系统差异造成重复路径或搜索错误 | 中 | 高 | macOS/Linux 结果不同、唯一约束冲突、不可定位文件 | S2-007 已确认媒体库显示名 NFC、唯一键 NFKC + Unicode full case folding、locale-neutral numeric sort key 与组件级根比较通过组合字符、sharp-s、全角、中文、无效 UTF-8、控制字符和并发碰撞测试；文件名排序、搜索及真实跨文件系统语义仍由后续切片验证 | 保留规范显示名并拒绝歧义库，缩小受支持文件系统范围 | 后端负责人 | 缓解中 |
| R-013 | 多媒体库共享任务队列时出现饥饿或请求被后台工作拖慢 | 中 | 高 | 一个大库占满 worker；API 延迟随扫描显著上升 | [S2-102](gates/MVP-2026-07-23/s2-bounded-scan-worker.md) 与 [S2-106](gates/MVP-2026-07-23/s2-scan-capacity.md) 已固定容量 1 wake signal、2 个全局 worker、available/created/ID 公平领取、256 active/256 batch，并以真实 SQLite queue 验证三个媒体库只运行两个、释放后继续，以及目标档扫描期间索引读取延迟。正式 HTTP catalog 延迟仍由 Stage 3 Gate 验证 | 暂停/限速后台任务，改为手动扫描模式 | 技术负责人 | 缓解中 |
| R-014 | 依赖许可证、FFmpeg 构建选项或 codec 分发义务不清 | 中 | 高 | SBOM 出现不兼容/未知许可证；无法提供对应源码或 notice | [供应链审查](supply-chain-review.md) 已生成 source/npm/image SPDX，确认 Debian FFmpeg 启用 GPL、libx264/libx265 并按 GPL 组合处理；最终双平台 digest 仍须附 SBOM/provenance、notices、漏洞结果和合规签署 | 替换依赖、关闭相关 codec 或停止分发受影响镜像 | 合规负责人 | 缓解中 |
| R-015 | 可选虚拟瀑布流破坏键盘顺序、焦点或大列表稳定性 | 中 | 中 | DOM 顺序与视觉顺序不一致、焦点丢失、滚动跳动 | 默认使用规则网格；对可选瀑布流验证固定占位、稳定游标、DOM 顺序和无障碍 | 保留默认网格并临时禁用未达标的瀑布流 | 前端负责人 | 开放 |
| R-016 | 缺少 CI、生成漂移检查和可复现 fixture 导致回归 | 高 | 高 | 开发者本地通过而干净环境失败；生成文件与源不一致 | 真实 PR CI 已覆盖双架构 Go/race、生成/兼容、mount、govips/FFmpeg、runtime/recovery 与 SBOM/license；媒体库 Backend Gate、S2-102～106 worker、目录、媒体收敛、故障恢复和容量均有分层证据，容量主档另由 Linux amd64/arm64 受限资源 job 强制；每个后续生产切片继续增加集成/E2E 和已许可 fixture | 阻止合并和发布，先恢复最小验证基线 | QA 负责人 | 缓解中 |

## 发布阻断风险

以下风险在首个可用镜像发布前不得保持“开放”且无证据：

- R-002 路径边界、R-003 扫描清理和 R-006 不可信媒体处理。
- R-010 网络暴露，以及已确认单管理员认证的实现与验证。
- R-011 备份/恢复和迁移。
- R-008 中实际承诺发布的架构。
- R-014 许可证与分发义务。

R-002～R-008、R-011、R-014 与 R-016 已因实验实现、契约或 spike 证据转为“缓解中”，但对应
报告列出的生产 HTTP/认证错误边界、只读发布挂载与运行期 unmount、真实强杀与生产磁盘
策略、媒体任务隔离、浏览器、代表性存储、最终镜像 SBOM/attestation、真实版本升级与发布
签署仍未完成；原生双架构 CI、真实 PR 基线兼容比较、TypeScript 生成和摘要锁降低了工程
风险，但不等于发布风险关闭。
R-009 的 10 GiB LRU 策略只有产品约束
而无实现证据，仍保持开放。不得把产品确认或局部 spike 误写成完整技术可行性证明。

R-002～R-016 的逐项 Stage 0 Owner、Fallback 和最迟阻断 Gate 见
[S0-109 风险复审](gates/MVP-2026-07-23/s0-109-risk-review.md)。

## 评审规则

- 阶段入口检查新风险、Owner 和触发信号；阶段出口检查缓解证据与剩余风险。
- 概率或影响上升到“严重”时，立即停止相关范围扩张并由技术/产品负责人共同决定继续、降级或 No-Go。
- 风险关闭需链接测试、spike 结果、ADR 或发布演练记录；只有口头判断不能关闭风险。
- 接受剩余风险必须写明适用版本、部署范围和复审条件，不能永久泛化。
