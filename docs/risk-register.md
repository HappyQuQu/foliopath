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
| R-005 | 10 万媒体/1 万目录目标超出四核、4 GiB 环境的扫描、查询或内存能力 | 高 | 高 | 队列持续增长、首次扫描不可接受、列表延迟或 RSS 超预算 | 历史 [S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 证据保留；2026-09-01 已将完整矩阵验证的有序扫描纳入 production repository，本机 arm64 强制 100k 基线 keyset P95 为 133.637 ms、零预算违规，见[容量回归复审](gates/MVP-2026-07-23/s4-search-capacity-regression-2026-08-31.md)。最终模型联合负载与 native amd64/arm64 仍未验收 | 保持 250 ms 预算和搜索语义；在同一候选 SHA 上完成最终模型联合负载及原生双架构复跑，失败则回退内部查询策略并阻断最终容量 Gate | 技术负责人 | 缓解中 |
| R-006 | 畸形媒体触发 libvips/FFmpeg 崩溃或资源耗尽 | 中 | 严重 | 解析超时、OOM、子进程堆积或相同文件无限重试 | [FS-03](spikes/fs-03-media-matrix.md)、[S3-004](gates/MVP-2026-07-23/s3-media-processing.md)、[S3-005](gates/MVP-2026-07-23/s3-media-jobs-cache.md) 与 [S3-006](gates/MVP-2026-07-23/s3-media-resource-safety.md) 已验证截断/像素炸弹拒绝、256 MiB 图片、100 MP/32,768 px、govips concurrency/cache、FFmpeg 单线程/进程组取消/输出 cap、2-worker/3-attempt 退避；[2026-08-06 修复](changes/FIX-2026-08-06-large-video-probe-diagnostics.md)以 1 TiB 视频源上限、probe/poster 各 60 秒和结构化失败分类替代会误报损坏的 4 GiB/共享 15 秒边界。进程内 libvips native crash 隔离和代表性峰值仍由 Release Gate/未来 ADR 复审 | 隔离失败资产；临时禁用受影响格式或视频封面 | 媒体处理负责人 | 缓解中 |
| R-007 | 已确认的 JPEG/PNG/WebP/GIF 或 MP4/MOV/MKV 契约在目标架构/浏览器表现不一致 | 高 | 中 | 同扩展名行为不一致；海报可生成但浏览器播放失败 | [FS-03](spikes/fs-03-media-matrix.md) 的相同 govips/FFmpeg fixture 已在原生双架构通过；仍区分服务端可处理与浏览器可直放，并在查看器 Gate 验证浏览器 | 明确标记不可播放；MVP 不加入隐式转码 | 产品负责人 | 缓解中 |
| R-008 | amd64 与 arm64 的 govips/FFmpeg 构建或功能不一致 | 中 | 高 | 某架构构建失败、缺少 codec、fixture 输出或崩溃行为不同 | 原生双架构 govips/FFmpeg fixture 与 [FS-05](spikes/fs-05-runtime-recovery.md) 同 Dockerfile runtime/recovery smoke 已通过；[S5-002](gates/MVP-2026-07-23/s5-compose-candidate-matrix.md) 又在本机原生 arm64 和操作者指定的原生 amd64 服务器通过完整候选媒体/Compose/恢复矩阵，并审阅相同嵌入 SPA 与运行包闭包 | 若需求允许先缩减发布架构；否则阻断发布 | 发布负责人 | 已关闭 |
| R-009 | 10 GiB 缩略图缓存、临时文件或 WAL 仍耗尽磁盘 | 中 | 高 | 可用空间持续下降、LRU 无法维持余量、临时文件残留或写事务失败 | [S3-005](gates/MVP-2026-07-23/s3-media-jobs-cache.md) 与 [S3-006](gates/MVP-2026-07-23/s3-media-resource-safety.md) 已验证 90%→80% LRU、512 MiB 余量和真实 `ENOSPC` 恢复；[S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 又在 100k 全量派生后把在线降配缓存精确收敛至 3,355,440 B 低水位，0 pending deletion、应用持续健康并关闭淘汰写竞争/写放大 | 暂停派生任务并清理可重建缓存，优先保护数据库 | 运维负责人 | 已关闭 |
| R-010 | 经认证的 LAN HTTP 被误暴露到公网或不可信网络 | 高 | 严重 | 路由器端口转发、云安全组放行、认证/CSRF 缺陷或错误代理信任范围 | 内建单管理员认证、业务 API 默认认证、CSRF、SameSite/HttpOnly Cookie；直连模式清除转发头并按 peer 限流；显式代理模式保持严格单跳 HTTPS、Secure Cookie 与 HSTS。文档明确 LAN HTTP 不提供链路机密性，公网由部署者提供 TLS/ACL | 撤销外部端口映射，收窄 `FOLIOPATH_BIND_ADDRESS`，必要时置于外部 TLS 代理后 | 安全负责人 | 缓解中 |
| R-011 | 自动迁移、升级或备份恢复导致配置丢失 | 中 | 高 | 迁移中断、只复制主 DB、恢复版本不兼容 | 只追加迁移；[S5-004](gates/MVP-2026-07-23/s5-recovery-failure-smoke.md) 已在原生 amd64/arm64 以不同前一候选 image ID 验证向前升级、旧镜像＋升级前离线备份配对回滚、完整 SQLite family 恢复、WAL 强杀恢复、缓存重建及满盘/损坏失败关闭 | 回到备份对应版本恢复，再从原媒体重建派生数据 | 发布负责人 | 已关闭 |
| R-012 | Unicode、大小写和文件系统差异造成重复路径或搜索错误 | 中 | 高 | macOS/Linux 结果不同、唯一约束冲突、不可定位文件 | S2-007 已确认媒体库显示名 NFC、唯一键 NFKC + Unicode full case folding、locale-neutral numeric sort key 与组件级根比较；[S4-002](gates/MVP-2026-07-23/s4-search-keyset.md)已用组合字符、sharp-s、全角、中文、保留变音符号、短词和 `%/_` fixture 固定语义，[S4-003](gates/MVP-2026-07-23/s4-search-backend-ready.md)又在 macOS/Linux 100k 档验证同一派生键、FTS 和 fallback 路径。真实多种文件系统仍留给 Release Gate | 保留规范显示名并拒绝歧义库，搜索实现不满足 profile 时停止 Backend Gate；必要时缩小受支持文件系统范围 | 后端负责人 | 缓解中 |
| R-013 | 多媒体库共享任务队列时出现饥饿或请求被后台工作拖慢 | 中 | 高 | 一个大库占满 worker；API 延迟随扫描显著上升 | S2/S3 已固定公平/资源边界；[S5-005](gates/MVP-2026-07-23/s5-release-capacity-candidate.md) 关闭 session touch、unchanged 派生和 cache 淘汰写放大，原生 100k 扫描并发浏览 P95 降至 6.116 ms，94,234 个后台任务调度期间重扫 140.190 s，全部 100,000 个媒体任务最终成功，三引擎 FPS/RSS 通过 | 暂停/限速后台任务，改为手动扫描模式 | 技术负责人 | 已关闭 |
| R-014 | 依赖许可证、FFmpeg 构建选项或 codec 分发义务不清 | 中 | 高 | SBOM 出现不兼容/未知许可证；无法提供对应源码或 notice | [供应链审查](supply-chain-review.md) 的 Stage 0 基线确认 Debian FFmpeg 启用 GPL、libx264/libx265；[S5-007D](gates/MVP-2026-07-23/s5-minimal-ffmpeg-runtime.md) 已以关闭 GPL/x264/网络的最小 LGPL 2.1+ 构建替换它并保留许可证。最终双平台 digest 仍须附 SBOM/provenance、notices、漏洞结果和合规签署 | 替换依赖、关闭相关 codec 或停止分发受影响镜像 | 合规负责人 | 缓解中 |
| R-015 | 可选虚拟瀑布流破坏键盘顺序、焦点或大列表稳定性 | 中 | 中 | DOM 顺序与视觉顺序不一致、焦点丢失、滚动跳动 | 默认使用规则网格；[Stage 3 Integrated Done](gates/MVP-2026-07-23/s3-browse-integrated-done.md) 已验证 grid/masonry 记忆、DOM 顺序、固定占位、虚拟容量和返回焦点，S5-005 已通过三引擎 100k DOM/FPS/RSS，S5-006A 又执行查看器键盘/焦点矩阵；五个桌面自动化项目已通过 200% 等效重排的焦点、控件、溢出与 axe 护栏，[S5-006B Chrome](evidence/s5-006b/README.md)在真实 Chrome 151 原生 200%、[Firefox 物理证据](evidence/s5-006c/README.md)在 Firefox 153.0.1 原生 200%/400% 下验证扫描、浏览、预览、Viewer 与快捷键；读屏、物理触控、移动设备和 Safari 缩放签署仍待完成 | 保留默认网格并临时禁用未达标的瀑布流 | 前端负责人 | 缓解中 |
| R-016 | 取消自动 CI 后漏跑本地检查或干净环境发生漂移 | 高 | 高 | 合并前未记录本地验证；生成文件与源不一致；目标架构发布失败 | `Makefile` 保留格式、架构、生成、lint、单元、集成与 E2E 统一入口；开发者在合并和发布前本地执行并记录结果，Docker Hub workflow 只负责 Release/手动双架构构建；POST-MVP-5 新增只读、手动、禁止 QEMU 的原生 amd64/arm64 evidence workflow，但只有同一 source SHA 的实际成功 artifact 才计证据；[2026-08-31 远端审计](evidence/int-001/native-remote-readiness-audit-2026-08-31.md)确认当前远端尚无该 workflow/候选提交/匹配 artifact；历史双架构、mount、媒体、runtime/recovery 与供应链证据继续保留 | 阻止合并和发布，先在干净工作树恢复并完成本地验证基线；不得把 workflow 存在、历史 run 或失败 artifact 记为通过 | QA 负责人 | 缓解中 |
| R-017 | 发布镜像的原生依赖闭包含未处置的 Critical/High 漏洞 | 高 | 严重 | 最终 SBOM/扫描出现 Critical/High；漏洞数据库变化 | [S5-007A/G](gates/MVP-2026-07-23/s5-supply-chain-candidate.md) 已固定 Syft/Trivy digest和完整报告；最小 libvips/FFmpeg、内建健康检查、无 shell distroless、固定源码 Expat 与修复来源 GLib 2.88.3 将本机 arm64 候选从 15 Critical / 136 High 降至 `0 / 0`，并移除 GLib 的 libmount/blkid 间接闭包。干净候选提交 `5c3b3c7` 的原生 arm64 已以相同结果生成 smoke、SPDX、notices、provenance 和不可变 digest 证据；原生 amd64 配对复扫及安全/合规签署仍缺失 | 若任一最终 digest 重新出现发现，升级/移除依赖或停止发布 | 安全负责人 | 缓解中 |
| R-018 | 视频故事板 backfill、重复 seek 或 hover 生命周期造成 CPU、I/O、缓存、请求或前端资源失控 | 中 | 高 | grid/poster 等待变长；浏览 P95 上升；队列/缓存持续增长；快速掠过产生请求风暴；虚拟卡片回收后 timer/动画仍活动 | [VSP-S2 Backend Evidence Ready](gates/POST-MVP-1/vsp-s2-backend-evidence-ready.md)已以双架构生产 FFmpeg、单并发/低优先级、128 项 admission、真实 cache repair 和 Linux 100k/10k 档关闭后端 Gate；[VSP-S3 Consumer/UI Ready](gates/POST-MVP-1/vsp-s3-consumer-ui-ready.md)又以 300ms intent、按需 decode、单活动动画、生命周期回收、六种浏览器/输入 profile 和 100-video 三引擎容量关闭前端 Gate；[VSP-301](gates/POST-MVP-1/vsp-301-product-vertical.md)已贯通生产镜像真实全链，剩余风险由 [VSP-302](gates/POST-MVP-1/vsp-302-target-platform.md) 原生候选复验与 VSP-S4 最终签署阻断 | 依次降低 worker/尺寸/质量、改为有界按需 admission；仍不满足时禁用 storyboard 并保留现有 poster | 媒体处理与前端性能负责人 | 缓解中 |
| R-019 | 文件事件丢失、乱序、overflow、网络盘不转发、watch 资源不足或删除误判导致内容延迟或索引损失 | 高 | 严重 | `IN_Q_OVERFLOW`/`ENOSPC`、watch 被移除、事件后目录不一致、挂载掉线时出现批量 delete、dirty 队列持续满或页面长期不更新 | [FTR-SCN-001](features/automatic-library-discovery.md)要求事件只作不可信提示；新增/修改/删除均经 `internal/files` 安全定向校准确认，掉盘/权限/overflow 保留可靠索引并合并安排完整扫描；每目录而非每文件 watch，队列/并发/revision 轮询有界；实现前必须通过 WCH-S0 ADR 与 Linux 100k/10k burst/恢复证据 | 先扩大合并并降低定向并发，再按库进入 degraded；仍不可靠时全局禁用自动发现，保留创建/启动/手动/定时完整扫描 | 扫描与运维负责人 | 开放 |
| R-020 | 任务中心全量重建造成无界 admission、队列饥饿、磁盘耗尽，或先清缓存导致可用预览丢失 | 中 | 高 | rebuild 一次登记全部资产；日常 poster/grid 或浏览延迟持续恶化；取消后队列继续增长；ENOSPC 后旧 ready 不可用 | [FTR-OPS-001](features/task-center.md)要求 parent run、asset keyset 小批 admission、最低后台优先级、active coalesce、磁盘安全余量、停止 admission 的协作取消和新文件成功后才替换旧缓存；OPS-003 必须在 100k/10k 档冻结上限 | 只保留 missing backfill，禁用 all rebuild；必要时完全隐藏批量入口并继续现有按需 self-heal | 媒体处理与性能负责人 | 开放 |
| R-021 | 原型、共享 token、生产页面和视觉基线各自演进，导致跨页漂移或为追求像素一致破坏真实功能、可访问性和大列表能力 | 高 | 高 | 同一控件出现多份样式；批量接受截图变化；页面只在 1440px 正常；为匹配原型改用 mock、全量客户端过滤或嵌套滚动 | [FTR-UIF-001](features/frontend-prototype-fidelity.md)固定视觉/功能双真相与唯一 shared owner；[UIF-S4](gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md)已接受 12 页共同 1280、12 页 × 4 断点、Linux 基线、真实 API、三浏览器/axe/输入、100k/10k 和跨文档收敛证据；没有以 mock、无界加载或嵌套滚动换取截图一致 | 保留已验证生产页面和机器 reference manifest；任何基线变化须解释来源并重跑真实链，不能以静态原型、mock 或批量 snapshot 更新替代 | 前端、设计系统与 QA 负责人 | 已关闭 |
| R-022 | root runtime 扩大应用或媒体解析漏洞的容器内影响 | 中 | 严重 | 进程可写非持久根、获得默认 capabilities、访问未授权挂载或修改宿主 bind 内容 | [ADR-0012](adr/0012-root-runtime-bind-data.md)只为零初始化 `/app/data` 接受 root；继续强制 `/library:ro`、锚定 `openat2`、认证、输入上限和有界媒体工具；权威 Compose 保留只读根、cap-drop 与 `no-new-privileges` | 收窄到受信 LAN，使用权威 Compose；若出现越界写或媒体工具逃逸则停止发布并恢复非 root/降权启动器方案 | 安全与发布负责人 | 已接受 |
| R-023 | 大规模标签关联或频繁收藏写入导致查询放大、分页漂移、丢失更新或原媒体边界混淆 | 中 | 高 | tag join 随集合增长退化；旧 cursor 混入新 revision；多标签替换部分提交；界面把标签表现成目录 | [FTR-CUR-001](features/favorites-and-tags.md)固定复合索引、稳定 keyset、全局 curation revision、ETag、20 标签上限、单事务替换和独立快速访问；CUR-S2 已通过 migration/SQLite/HTTP/真实 composition 与原媒体 hash/mtime 不变证据 | 暂停标签写入并保留只读收藏列表；若容量不达标则缩小筛选组合或延后标签 UI，不修改原媒体 | Curation、前端与性能负责人 | 缓解中 |
| R-024 | 本地 AI 推理在 4 GiB 环境造成 OOM、长期占满 CPU 或拖慢浏览 | 高 | 严重 | RSS 超 3.2 GiB；浏览 P95 退化超过 20%；backfill 无法在预算内结束 | INT-001 固定并发 1、延迟加载、输入/批量上限，并在 100k 档与浏览并发测量 | 只保留手动/夜间处理或图片语义；仍超限则 feature No-Go | 推理运行时与性能负责人 | 开放 |
| R-025 | 模型、权重、tokenizer 或向量引擎许可/供应链不允许产品分发或引入高危 native 闭包 | 高 | 严重 | 权重只有研究/非商用授权；来源/hash 不清；候选依赖未选定的 SentencePiece/tokenizer runtime；双架构 SBOM 出现未处置发现 | 模型 manifest、哈希、独立权重许可审查、tokenizer 可执行合同、原生双架构 SBOM/provenance；ADR-0013 接受前不得选型 | 更换许可清晰模型/runtime；无法替换则删除对应能力 | 合规、安全与发布负责人 | 开放 |
| R-026 | 10 万媒体向量查询、构建、备份或索引损坏恢复不满足容量/一致性 | 高 | 高 | P95/P99 超预算；索引文件成为唯一事实；强杀后无法恢复；DB/备份膨胀 | 比较 exact/SQLite extension/可重建 ANN；SQLite 保存权威派生行，索引临时构建原子激活 | exact 达标则不用 ANN；否则缩小容量或停止语义切片，不引入未经批准服务 | Semantic、SQLite 与性能负责人 | 开放 |
| R-027 | 人脸聚类错误合并不同人物，用户将模型建议误认为真实身份 | 高 | 严重 | 核心簇 precision 低于 99.5%；已命名人物被后台合并；UI 使用“识别”措辞 | 核心/边缘分层、用户命名、named person 禁止自动合并、人工 assignment/cannot-link 优先、代表性套图评测 | 降级为 pair/小组建议或删除人脸范围；不降低 precision 门槛换取上线 | 人脸、产品、隐私与 QA 负责人 | 开放 |
| R-028 | 人脸 embedding、人物名称、查询或裁剪泄露，或清除/备份语义不完整 | 中 | 严重 | 日志/诊断包含敏感数据；禁用/删除后残留；未告知即默认分析；备份丢人工人物关系 | 默认关闭、管理员告知、API 不返 embedding、诊断脱敏、派生/应用状态分类、清除/备份/恢复集成测试 | 全局禁用人脸并清除派生数据；保留核心浏览，必要时延后人物库 | 安全、隐私与运维负责人 | 开放 |
| R-029 | 模型升级、源变化或自动重聚类覆盖人工人物关系，造成不可逆整理损失 | 中 | 严重 | manual assignment 消失；cannot-link 再次合并；两代向量混排；重建先删可靠结果 | 人工关系独立应用状态、generation 隔离、并行构建原子激活、约束优先、合并审计和配对备份恢复 | 回滚到上一 generation；禁用自动重聚类，只保留人工人物库 | 人脸、数据与恢复负责人 | 开放 |
| R-030 | 模型下载源、部署镜像或 `/models` 映射被替换、投毒、滥用为 SSRF，或直接来源丢失使查询不可恢复 | 高 | 严重 | API 可提交 URL/路径；重定向到内网；hash/manifest 不匹配仍加载；目录可写/symlink/mount crossing；项目宣称但未运营国内镜像；直接模型消失后静默换代 | 签名发行清单、部署预配置 origin、重定向/地址策略、并发 1 与临时空间、allowlist/hash/license/arch 校验、`/models:ro` 固定安全边界、托管复制默认、direct 每次加载复核、provenance 与真实镜像 Gate | 删除在线/镜像入口，仅保留离线托管复制；direct 失效则 model unavailable，保留旧索引和人工状态，不自动换模型 | AI 模型管理、安全、发布与镜像运营负责人 | 开放 |

### POST-MVP-5 风险处置责任与最迟 Gate

下表分配的是推动决策的 owner 角色，不是假定已经有具体人员值守。Frozen scope 必须把纳入切片的角色
落实到可执行负责人；风险保持开放不影响本表关闭“无 owner/无 Gate/无 fallback”的治理缺口。
`POST-MVP-5` revision 2 已纳入 A～E，因此 `R-024～030` 全部适用。A+B 保持当前 Gate；人脸相关
`R-027～029` 立即恢复为 E 的准入风险，但产品纳入不等于风险已接受或隐私已签署。

| 风险 | Owner 角色 | 最迟决策 Gate | 后续复验证据 | 触发时强制 fallback |
| --- | --- | --- | --- | --- |
| `R-024` | 推理运行时 + 性能 | `INT-S0` 冻结 model/runtime/lifecycle 与资源布局 | 对应 slice `S2` 后端容量、`S4` 全进程双架构 | 只保留手动/夜间或图片语义；仍超限则删除智能切片 |
| `R-025` | 合规 + 安全 + 发布 | `INT-S0` 批准每个纳入 runtime/权重/vector 许可与来源 | `S4` 最终双架构 digest、SBOM、VEX、notices、provenance | 替换依赖；无法替换则删除对应能力并停止分发 |
| `R-026` | Semantic + SQLite + 性能 | `INT-S0` 冻结 exact/存储/量化与 100k 空间布局 | `S2` 真实 embedding/recovery、`S4` backup/full-process | exact 达标则拒绝 ANN；仍失败则降容量或删除语义切片 |
| `R-027` | 人脸 + 产品 + 隐私 + QA | `INT-S0` 决定 face slice 纳入/降级/删除 | `S2` 合法真实 core precision、`S3/S4` 纠错文案与完整链 | 降为 pair/小组建议；不达 99.5% 则删除整组建库/人脸 slice |
| `R-028` | 安全 + 隐私 + 运维 | `INT-S0` 接受数据分类、告知、清除与备份边界 | `S1` 合同、`S2/S4` 删除/诊断/备份恢复 | 默认并全局禁用人脸，清除派生数据并延期人物库 |
| `R-029` | 人脸 + 数据 + 恢复 | `INT-S1` 冻结 generation/manual/cannot-link 事务语义 | `S2` 并发/强杀、`S4` 升级/回滚/配对恢复 | 回滚旧 generation，关闭自动重聚类，只保留人工人物库 |
| `R-030` | AI 模型管理 + 安全 + 发布 + 镜像运营 | `INT-S0` 冻结实际来源；无 owner 即删除在线/镜像范围 | `S2` 下载/映射实现、`S4` 真实 TLS/key/mirror/final image | 只保留 `/models:ro` 离线托管复制；direct 失效即 unavailable |

任何 owner 角色未落实、最迟 Gate 无证据或 fallback 不能实际执行时，对应 slice 自动保持 No-Go；不得
把风险改成“已接受”来绕过。

INT-001 在 2026-08-25 的初步证据没有关闭 `R-024～R-030`：YuNet/SFace macOS CPU 随机张量循环显示
资源量级可继续研究，但 SFace RSS 有约 0.56 MiB 单调增长，Linux C API 泄漏仍未判定；100k × 512
float32 SQLite 已达 410,619,904 bytes，强化 `R-026`，float16/int8 仅有随机向量 payload/recall 证据；
OpenCV 候选已固定 revision/hash，Chinese-CLIP 许可仍不清且 InsightFace 公共权重非商用；一次 SFace
中断下载可断点续传不等于完整供应链/SSRF/原子激活验证。详见 [INT-001 evidence](evidence/int-001/README.md)。
同日 ANN 比较进一步说明 `R-026` 不能通过“引入 HNSW”直接关闭：测试候选默认 recall 不合格，高预算
10k build 已超过 50 秒，导入存在未封顶分配审计缺口且导出不具 byte determinism。exact 路径在当前本机
100k 基线仍更简单；只有受限 Linux 并发 exact 失败后才应重新评估 ANN，避免无收益扩大恢复与供应链面。
SigLIP 2 F32 离线合成 smoke 观测约 1.171 GB RSS 和 58.414 ms 图片 P95，说明 `R-024` 可以继续验证，
但 1.539 GB 模型包、PyTorch 开发运行时、非代表性图片和缺少并发浏览使其不能关闭资源风险。模型还依赖
权重之外的 tokenizer/config/preprocessor；单文件 hash/catalog 无法证明整个可执行包身份，进一步强化
`R-025` 与 `R-030` 的 signed multi-file manifest、原子激活和离线恢复要求。4 帧合成代理中 mean pooling
优于 max-frame，只是 INT-011 的实验方向，不降低真实视频质量 Gate。
较小的 SigLIP 1 对照把观测 RSS 降至约 724 MB，但图片 P95 仍约 60.373 ms，并新增 SentencePiece native
闭包；这给 `R-024` 提供了资源备选，却没有关闭 `R-025`，也没有代表性质量证据支持选型。
multi-file catalog v2 与 no-replace package publish spike 缩小了 `R-030` 的技术未知：缺失、额外、篡改、
symlink 文件可按整包失败，发布不覆盖已有 generation。但 package digest 不是签名，Linux `openat2` 测试
只有双架构交叉编译而没有 native 执行，下载 staging/配额/取消和数据库 active-generation 事务也未贯通，
因此 `R-025`/`R-030` 继续开放。

2026-08-26 的 10 张 Wikimedia Commons Cosplay 公开许可 pilot 让两个 SigLIP 候选在真实人物照片上
出现了可见差异：SigLIP 2 中英文 Recall@1 均为 1.0，SigLIP 1 中文为 0.917，且两者宽泛多相关查询
都存在 Top-3 漏召回。但小样本、人工明确查询和缺少困难负例不足以关闭质量风险。直接解码最高
5,388×3,592 原图后，SigLIP 2/SigLIP 1 RSS 分别约 1.71/1.27 GB；这强化 `R-024` 的输入边界要求，
不作为 3.2 GiB 验收。正式路径必须以现有安全媒体链生成的有界缩放输入，在完整进程、浏览并发、
100k backfill 与 Linux 双架构下重测；此前不得选择模型或进入生产实现。
同日 512px WebP/quality 82 Pillow surrogate 三次新进程复测未改变 24 条查询的 Top-1/首个相关项
排名，并将 SigLIP 2/SigLIP 1 RSS 中位数相对 direct-original 约降低 31%/43%。这支持以有界派生图
缓解 `R-024`，但 Pillow 不等于生产 libvips，10 图也无法测浏览退化或 backfill；风险继续开放。
当前生产 govips adapter 又在 Linux/amd64 QEMU 生成同一 10 图，查询排名保持稳定，进一步缩小输入
变换未知；但 QEMU 不是原生双架构，无法关闭 `R-024`。完整 Dockerfile 两次因 Docker Hub manifest
`EOF` 失败、改用本地缓存才离线通过，强化 `R-030` 的多来源/离线包要求，不构成项目镜像已运营证据。
随后以相同 digest 的 mirror 缓存完成未修改 Dockerfile 的原生 arm64 `libvips-test`，arm64 native 与
amd64 QEMU 的 10 个 govips WebP byte-identical，缩小 `R-024` 的 transform 跨架构未知；原生 amd64、
ONNX embedding、整进程 browse/backfill 和 1,000 图仍未完成，`R-024`/`R-030` 继续开放。
同日隔离下载 failure matrix 证明固定 catalog origin/ETag/size/hash、精确同源重定向、配额预检、
传输中取消后续传、错 hash 不发布和 no-replace 可以组合实现；macOS/arm64 与原生 Linux/arm64 均
通过，arm64 受限 tmpfs 的真实 `ENOSPC` 也未发布 candidate、未改变 active，缩小 `R-030` 的状态机
未知；真实子进程在 partial 后被 `SIGKILL`，新进程也能以固定 ETag/Range 恢复并精确发布。但 DNS
单次解析固定 IP、特殊/私网/混合 answer 整体拒绝和禁用环境 proxy 随后在两套 arm64 环境通过，
进一步缩小 rebinding 面。真实 catalog TLS/CDN 轮换与 resolver 失败、其他发布阶段磁盘满/容器强杀、
active generation、原生 Linux/amd64 和真实镜像 owner 仍未覆盖。Ed25519 primitive 已在两套 arm64
环境拒绝未知 key、篡改、过期、rollback 和同序号换内容，但使用临时 key，未证明生产 key 托管/轮换/
撤回或 durable checkpoint。隔离 SQLite registry 随后证明 checkpoint 与 active pointer 单事务及
注入失败整体回滚、重启持久化；原生 Linux anchored reconciliation 又证明精确 orphan 不自动激活，
missing/corrupt/restore 只改变 availability 并保留 active/checkpoint，未知目录不删除。真实只读
bind mount 进一步验证 direct source kind 不可变、消失/精确 remount 恢复且不复制/删除/切 active。
S1 已冻结 production migration 计划/事务 owner、direct 部署/API、备份/配额与 unavailable 查询语义；
实际实现和 Release evidence 仍未完成。
因此 `R-030` 继续开放且仍不承诺在线/国内镜像。

同日原生 Linux/arm64 的 4 CPU/4 GiB/no-network 三次 100k×512 float32 联合向量负载进一步收敛
`R-026`：exact search P95 229.843～258.371 ms、取消/重启正确，但 keyset browse proxy 相对退化
16.25～23.22 倍，未通过 20% Gate；SQLite 文件 410,619,904 bytes 尚未包含视频帧、人脸、WAL 与
备份。由此拒绝 float32 作为 combined final layout。真实 embedding 的 float16/降维 recall、生产
browse path、native amd64 和最终空间闭包未完成，`R-026` 保持开放；若失败按既定 fallback 缩减范围，
不通过放宽门槛或直接引入已拒绝的 HNSW 来关闭风险。
同矩阵 float16 DB 为 136,880,128 bytes、search P95 138.854～156.912 ms，说明容量路径可继续；但
随机向量不构成真实模型质量或跨架构容差证据，因此不降低 `R-026` 等级。

同日的合成人脸状态机缩小 `R-027/R-029` 的所有权未知：只有 core 可批量创建/并入人物，edge 必须
逐脸确认；named/manual faces 在换模型重聚类时完全排除，cannot-link 与已有同人物关系冲突时失败关闭。
这证明状态语义可实现，但没有合法真实 ground truth、99.5% core precision、生产事务/并发/恢复证据，
因此两项风险继续开放，不允许把合成通过写成“人脸识别可用”。
2026-09-01 的授权 `coser` 扩大样本又暴露单链 core bridge：986 个候选曾经通过少量中间脸形成跨八个
来源的 834-member 巨簇。聚类 owner 已改为 smallest-ID anchor coherence，定向 bridge、race 与
100k×512 容量通过，同一输入重算不再出现巨簇。该修复降低 `R-027` 的算法传播风险，但没有正式身份
ground truth 或 99.5% precision，故风险仍开放且整组操作继续失败关闭。

2026-08-26 的上游 ONNX 来源复核强化 `R-025/R-030`：Google SigLIP 2 仓库可见 ONNX 仅位于未合并
PR ref，ONNX Community 转换仓库虽由 HF Staff 提交，但为单一贡献者、多变体 11.4 GB 第三方闭包。
两者均不进入受信下载源。必须从固定 Google revision 以固定 exporter/opset/shapes 自行导出、做双架构
数值验证/SBOM/hash 并由项目签名；完成前 `R-025/R-030` 保持开放。
同日固定 toolchain/opset/shape 的自导出已三次 byte-identical，macOS arm64 与原生 Linux/arm64
四输出均通过 `1e-4` 数值容差，缩小了第三方转换与 arm64 兼容未知。但单语义 session 的 Linux 峰值
约 2.19 GiB，export tracer 仅允许当前固定形状，且 amd64、统一 production runtime、SBOM/漏洞/notices、
项目签名和再分发签署仍缺；因此 `R-024/R-025/R-030` 都保持开放，不能因成功导出批准模型。
固定形状 split export 随后把 image/text 图拆为 371.7/1,129.3 MB，双次导出一致且 arm64 public pilot
排名不变；Linux P95 相对单体同形状分别下降约 20.7%/78.1%，据此拒绝单体运行布局。但顺序关闭 image
再加载 text 的进程 HWM 仍约 2.03 GiB，Python allocator/native session 卸载后仍保留约 527～530 MiB。
split 只能作为 `R-024` 下一轮候选，必须用 production adapter、完整浏览/backfill 和长期切换验证。
30 轮 split session 切换随后失败：末轮 RSS 比首轮高 213,004 KiB，超过 128 MiB retained-growth
门槛，峰值约 2.20 GiB。虽然早期跃升后进入平台、未证明无限 leak，但 `R-024` 不能假设 session close
回收全部内存；需明确 allocator/session owner、复用或隔离策略，并在完整进程复测，不能放宽门槛结案。
2026-08-27 的受控复测保留该失败与原门槛，关闭 ORT CPU memory arena 后 30 轮 cycle RSS 增长为
0 KiB、峰值降至 2,046,084 KiB，三次 public pilot 排名和延迟未回退。该配置缩小 `R-024`，但只有
arm64 Python wrapper 证据；production adapter 必须强制 arena off，且 amd64、长周期和完整进程仍阻断。
同日三次 4 CPU/4 GiB/no-network arm64 组合负载让该 image encoder 与 synthetic 100k float16
backfill/search/browse/cancel/restart 并行：容器峰值 1.29～1.33 GB、search P95 165.443～186.532 ms、
恢复完整，但 browse 相对退化 5.32～12.74 倍，原 20% Gate 失败。它排除了一个即时 OOM 假设，不能
关闭 `R-024`：生产 HTTP/catalog、真实 100k embedding、人脸并发、长周期和 native amd64 仍缺。
随后 100 轮 arena-off 双图 load/infer/close 的 retained process RSS 只增加 28 KiB，但 cgroup peak
达到 3,719,651,328 bytes，超过 3.2 GiB 门槛；反复完整模型 reload 生命周期因此被拒绝。进程 HWM 与
cgroup peak 的差异推测含大量 file cache，下一轮必须比较有界常驻 session/隔离 worker并持续采样
`memory.stat`，不能只看 `/proc/self/status`。
双 session 常驻对照也失败：100 轮交替推理虽稳定，但 cgroup current/peak 达约 3.56/4.01 GB，且
`memory.stat` 约 1.90 GB anon + 1.65 GB file；在完整应用负载加入前已无余量。当前 SigLIP 2 的 reload
和 resident 两种 lifecycle 都不满足 4 GiB，`R-024` 缓解方向必须转为更小模型或正式缩减需求。
较小 SigLIP 1 随后给出可继续的资源方向：split 双次导出一致，双 session 100 轮 peak 2.18 GB；叠加
synthetic 100k float16 三次 peak 2.364～2.370 GB、search/restart 通过。但 browse 相对 Gate 仍失败，
且 10 图中文 pilot 有一次 Top-1 miss；因此 `R-024` 继续开放，不能用资源改善替代 1,000 图质量、
production full-process、native amd64 和真实 browse 验证。
真实 10k 目录/100k 文件生产 catalog 容量配对进一步显示：ordinary recursive browse 仅 +11.2%，说明
micro-proxy 的数倍退化不代表产品路径；但 cgroup peak 从 1.604 GB 升至 3.590 GB，超过 3.2 GiB，
global search/storyboard browse 单次为 +25.3%/+20.3%。两轮无 AI/有 AI 的 search-keyset 都超过既有
250 ms 预算，故当前环境没有干净全预算基线。`R-024` 仍开放，下一轮先降模型内存再重复配对。
dynamic-QInt8 虽把两图压到约 281 MB、Linux cgroup peak 降至约 811 MB，却令 native Linux 中文
Recall@1 从 0.917 跌至 0.25，且 macOS/Linux 仅 8/24 Top-3 相同；该配置已拒绝，不得作为 `R-024`
fallback。不同 runtime minor 是额外混杂因素，后续候选必须固定同版再作双架构判断。
float16-internal 候选保留了 24/24 pilot 排名，双 session/生产 100k cgroup peak 降至 1.614/2.906 GB，
ordinary recursive browse +14.5%，首次把该 arm64 proxy 压回 3.2 GiB 内；但 CPU 推理变慢且 global
search 单次 +41.6%，只一组配对、baseline keyset 失败、不同 runtime minor 和小质量集仍阻断，故
`R-024` 继续开放，float16 仅为下一轮资源优先候选。
两组追加配对令三次 float16 生产 peak 固定在 2.860～2.951 GB，ordinary recursive browse 退化
6.0%～14.5%，首轮 global +41.6% 未复现；但 storyboard browse 一轮为 +20.34%，且六次 no-AI/AI
search-keyset 均失败既有 250 ms 预算。arm64 内存/普通浏览子风险缩小，完整 `R-024` 仍不能关闭。
无 AI 的原生 Linux/arm64 10k/100k component timing 进一步把两页 358.623 ms 拆成服务第一页/第二页
167.755/190.937 ms、repository count 66.987 ms 和 list 106.750/130.316 ms。失败不是模型引入，也不只是
两页合并口径；重复 count 与 broad list 都需处理。后续 query-plan 已证明 FTS candidate scan/rowid
回表以及 first/second list 临时 B-tree 排序，第二页 keyset 未改变执行形态；但候选修复和原预算复测
尚未完成，禁止通过放宽 250 ms 门槛掩盖该债务，`R-024` 继续开放。
float16 split 的基础失败关闭矩阵在 macOS/arm64 ORT 1.29 与原生 Linux/arm64 ORT 1.28 均拒绝
空、随机、截断模型和 image/text 错误 shape/dtype，子进程无超时/信号崩溃且错误后推理恢复。这缩小
畸形输入造成 runtime 失稳的子风险，但 protobuf-valid hostile graph、输入字节 admission、生产 C/Go
adapter 的 timeout/cancel、native amd64 和统一 runtime 仍未验证，`R-024` 保持开放。
parser-valid hostile graph 追加验证同目录 external-data 控制图可推理，而 `../` external-data、未知算子
和循环图在两套 arm64 runtime 均拒绝且无 hang/crash。runtime 行为不能成为文件边界：生产 package
首版不需要该能力，故候选收敛为 graph 内嵌全部 initializer、发行校验拒绝 external-data，runtime
结果只作纵深防御；oversized allocation 与生产 adapter 隔离当时仍缺，
因此不关闭 `R-024/R-030`。
6 GB 固定输出 graph 在 Linux/arm64 的 1/1.5/2 GiB 有效 child address-space 档均返回
`RuntimeException`，没有 timeout/signal；512 MiB 因安全控制图也失败而不计。该结果缩小 runtime
异常分支未知，但 child RLIMIT 无法隔离同一 Go 进程，仍必须依靠批准图 exact hash、固定 shape、发行
校验和生产 adapter 超时/取消；`R-024/R-030` 继续开放。
官方 ORT 1.28 C API 的 isolated Go/cgo harness 在 native arm64 通过 100 轮取消/恢复和 30 轮 race
build，两个 RSS 增长均低于 128 MiB；但 raw runtime error 确实包含模型路径，进一步确认 production
adapter 只能返回稳定码。amd64、最终镜像 ABI/漏洞扫描、context/admission owner 和完整并发仍缺，
因此 `R-024/R-030` 保持开放。
隔离 distroless arm64 final-stage 已把“builder 可链接但 final image 遗漏 SONAME”的真实 exit 127
失败保留为回归证据；补齐后在 non-root/read-only/no-network/cap-drop 下通过 100 轮。`base-nossl`
最小闭包进一步移除了未使用 OpenSSL 及 7 个 High 发现，但 Grype 仍报告 glibc 1 Critical/2 High；
Debian 的 minor/no-dsa 分类必须经安全 owner 的 VEX/可达性评审或改用修复基座，不能自动 suppress。
实际 arm64 harness/ORT ELF 对三项受影响 API 无直接 import/精确字符串命中，但该负面证据不能排除
glibc 内部间接路径、动态查找或未来生产 composition，只能附给 VEX 评审。
CycloneDX 未把裸 ORT `.so` 识别为 package component；现已补充通过官方 1.6 Schema 结构校验的
arm64 显式 component，绑定 version/library/archive/source/license/notices，但 composition 明确为
incomplete。它尚未并入最终镜像 SBOM；native amd64、生产 composition、漏洞/VEX 关联与 signed
provenance 仍缺，`R-024/R-025/R-030` 继续开放。
SentencePiece 0.2.1 与 ORT 1.28 的同一 no-SSL distroless arm64 闭包随后在严格 runtime profile 下
完成 31 条端到端文本 parity，模型文件保持外部只读挂载，并固定 SentencePiece archive/library/
license/tag commit 与 incomplete CycloneDX component。固定 archive 随后逐文件对照官方 commit 的 258 个
raw Git blob 与 7 个 executable mode，全部 exact；这关闭
source-content 对应关系，但不提供 exact distribution URL 或上游签名。后续固定 Grype 0.116.1 与有效
2026-08-26 DB、关闭自动更新后的 exact arm64/QEMU amd64 扫描均确认 glibc 1 Critical/2 High 且无 fixed
version；native amd64、最终签名 SBOM、显式 native-library advisory correlation、VEX/fixed base、signed
provenance 和 production composition 继续阻断。
旧的 harness/ORT 两文件检查未发现受影响 glibc import，但扩展到 app/ORT/SentencePiece/libstdc++/
libgcc_s 后，两架构 exact libstdc++ 均直接导入 `ungetwc`，调用点属于 wide-character
`stdio_sync_filebuf`。这不证明应用触发特殊重叠编码漏洞条件，却否定了 closure-wide“无直接导入”的
简化豁免路径；安全 owner 完成调用路径/运行时配置评审或更换修复基座前不得签 VEX。
双架构 `LD_PRELOAD` tripwire 的 control probe 均真实拦截 `ungetwc` 并 exit 86；同一 tripwire 下固定
31-case tokenizer→text encoder 链路未触发且保持数值 parity。该动态证据只覆盖 default locale 的固定
推理路径，不覆盖其他 libstdc++ wide-stream 使用、alternate locale/encoding、生产 HTTP composition 或
native amd64，因此不足以降低风险等级或自行签署 VEX。
后续 Docker Scout 1.24.0 已能扫描 exact arm64 text-runtime，但只识别 17 个 package，并漏掉 ORT/
SentencePiece；两次扫描还把相同文件挂到不同 Debian package。隔离合并器现扁平化该不稳定归属、合并
重复依赖并补入两个显式 native component；两次规范化输出 byte-identical，且无 dangling ref。该结果
仍标记 incomplete。相同流程随后在 QEMU 构建的 amd64 镜像及其架构专属 ORT/SentencePiece component
上两次得到 byte-identical 输出，但这不是 native amd64 扫描。以上只关闭“是否存在可复现合并路径”，
不降低最终双架构 SBOM、漏洞、VEX、签名或生产 composition 的 Gate，`R-025/R-030` 保持开放。
两次约 300 秒的官方 SentencePiece 在线获取路径（curl 与 BuildKit remote ADD）均未完成，证明 release
compile 不能依赖实时境外连接。隔离 Dockerfile 现从两个本地 reviewed context 读取 SentencePiece/ORT
归档并在解包前强制 hash，离线重建成功；归档的获取/镜像与签名仍须由更早的受控供应链步骤负责。
重复构建保持 runnable platform manifest/config，但自动 provenance 改变外层 local index，发布必须签
exact published index，不能用某次本地 index digest 冒充跨构建不变的内容标识。
供应链复核已固定 ORT license/notices 与三项候选权重的 revision/hash，但 Docker Scout 因未登录未
产生报告，Grype 裸包扫描又同时得到 0 match/0 component，不能作为干净结果；最终镜像 SBOM/双架构
扫描仍缺。SFace 目录的 Apache-2.0 声明不足以关闭精确权重训练数据
来源及商业/再分发澄清，当前 production hold；未获合规签署时只能替换候选或延期人脸 slice，不能
通过用户侧下载或镜像源转移项目责任。`R-025/R-030` 保持开放。

Stage 4 媒体内容风险更新：S4-005B 已用真实认证 composition、poisoned catalog path、
source fingerprint 变化、missing/offline、Range/取消/有界 admission 和 Linux arm64
`openat2` nested-mount fixture 缓解 R-002/R-006/R-012/R-016。amd64 QEMU 因缺少所需
`openat2` 能力按设计失败关闭；这不是 native amd64 通过证据，仓库 billing 恢复后必须重跑
PR native job，Stage 5 仍阻断正式只读 volume、运行期 unmount、浏览器与发布镜像。

2026-08-12 的[媒体处理韧性与诊断纠偏](changes/FIX-2026-08-12-media-processing-resilience.md)
继续缓解 R-006/R-018：按真实格式限制非 JPEG 100 MP，真实 JPEG 只在 100～180 MP 使用
shrink-on-load；空文件、像素超限、截断数据和未知工具故障保持不同稳定诊断；退出 0 的有效
产物不再被超长 stderr 误判。故事板无帧只搜索 `±1s` 邻帧，再执行 10→4 帧，4K fallback
120 秒且全流程不超过 5 分钟，耗尽后永久失败，防止相同输入循环占用队列。external libdav1d
只用于 AV1 派生解码，不扩大浏览器播放承诺。该修复不关闭两项风险；最终候选镜像、代表性
存储/媒体矩阵和 VSP-302/S4 证据仍适用。

同日的 [JPEG 有界容错与 MPEG-TS 派生兼容](changes/FIX-2026-08-12-tolerant-jpeg-mpegts-derivation.md)
继续缓解 `R-006/R-018`，但也扩大 `R-008/R-014` 的候选复验面：JPEG 容错只对
真实、≤100 MP 且命中窄错误 allowlist 的输入执行一次，有效产物以 ready/succeeded
交付并只保留内部 attempt warning；其他损坏仍失败。MPEG-TS 仅对既有视频候选派生，
不新增 `.ts` 或播放承诺。最小 FFmpeg `7.1.5-4` 的 mpegts demuxer 必须进入最终双架构
SBOM、notice、漏洞复扫、实际派生 smoke 和不可变 digest 证据；未完成前不降低相关风险等级。

## 发布阻断风险

以下风险在首个可用镜像发布前不得保持“开放”且无证据：

- R-002 路径边界、R-003 扫描清理和 R-006 不可信媒体处理。
- R-010 网络暴露，以及已确认单管理员认证的实现与验证。
- R-011 备份/恢复和迁移。
- R-008 中实际承诺发布的架构。
- R-014 许可证与分发义务。
- R-017 最终原生双架构镜像尚未从干净提交完成全阻断复扫与安全签署。
- R-022 root runtime 的剩余影响必须按 ADR-0012 由产品与安全接受，并完成新候选 smoke。

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
键盘、触摸、forced-colors 与 reduced-motion 自动化适用复验；后续 S5-006B 已取得
Firefox 原生 200%/400% 缩放证据，真实读屏、OS 高对比、物理触摸/移动设备和 Safari
缩放仍未完成；UIF-405 又在最新共享集合上通过三引擎 100k
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
该快照早于 2026-08-01 的 ADR-0012；root runtime 变更新增已接受的 R-022，并重新打开
AF-012 镜像身份与 bind-data smoke。新的 root 候选证据完成前继续 No-Go。

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
