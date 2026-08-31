# INT-S0：本地智能媒体发现 Architecture Ready

## 当前判断

**Go（2026-08-27，仅 POST-MVP-5 revision 1 的 Slice A+B）**。授权进入 A+B 的 S1 合同设计；不授权
生产后端、migration 或 UI。C 标签、D 视频和 E 人脸不在 revision 1，不能从本 Gate 偷渡开工。

2026-08-27 已停止继续扩张本地测试面。现有技术探索按
[INT-S0 收口与阻塞清单](int-s0-closeout-and-blockers.md)冻结；后续只接受产品决定、外部资源到位或
候选/合同发生变化后能直接改变决策的验证。不得用更多合成排列代替真实数据、amd64 或合规签署。

## 进入依据

- Feature：[FTR-INT-001](../../features/intelligent-media-discovery.md)
- Change Record：[CR-2026-021](../../changes/CR-2026-021-intelligent-media-discovery.md)
- 技术方案：[智能媒体发现技术架构](../../architecture/intelligent-media-discovery.md)
- Proposed ADR：[ADR-0013](../../adr/0013-local-ai-runtime-and-derived-vector-index.md)
- Spike：[INT-001](../../spikes/int-001-ai-feasibility.md)
- 人脸隐私评审草案：[INT-015](int-015-face-privacy-review.md)
- 模型分发/存储评审草案：[INT-019/021](int-019-021-model-distribution-review.md)
- 收口与阻塞：[INT-S0 closeout](int-s0-closeout-and-blockers.md)
- Frozen scope：[POST-MVP-5 revision 1](../../releases/POST-MVP-5-scope.md)

## Go 条件

- [x] 产品负责人接受 A+B 范围、非目标、用户流程、隐私边界和降级路径。
- [x] target version、16/32 工程周 scope budget、role owners 与 Frozen revision 1 已确定。
- [x] 对 revision 1 A+B 冻结候选架构、owner、最迟质量/双架构/许可 Gate 和失败时删减路径；
  不要求在 S0 重复执行最终发布矩阵。
- [x] Workstream B 已将 SQLite float16 exact 保留为 100k/损坏恢复候选并拒绝当前 HNSW；真实 embedding
  Recall、完整进程和 amd64 归 Slice B Backend/Release Gate。
- [x] Slice E 不进入 revision 1；未来只有合法人脸数据、核心簇 precision、隐私和许可通过后才能以新
  scope revision 开工。
- [x] ADR-0013 已对 A+B 接受 ORT C API、SigLIP 1 float16-internal、SQLite float16 exact、CPU-only
  与单进程边界；最终质量和发布 Gate 不因接受 ADR 而跳过。
- [x] revision 1 模型获取冻结为 `/models:ro` 离线基线、托管复制与严格直接读取；不承诺项目在线源
  或国内镜像，未来新增必须另开 scope revision。
- [x] A+B 的派生数据分类、清除、模型升级和失败降级已确认；未来人工人物关系默认另行确认，不进入
  revision 1 数据合同。
- [x] `R-024～R-030` 已有 owner 角色、最迟决策 Gate、复验证据和强制 fallback；风险仍开放，
  Frozen scope 仍须把纳入切片的角色落实到可执行负责人。
- [x] S1 需要更新的 PRD、user flow、UI、architecture/ADR、data/migration、API/generated client、
  security、deployment、testing 与 traceability 文件已在 scope proposal 逐项列明；当前未提前改合同。
- [x] A+B 上限 16 单人工程周，POST-MVP-5 总上限 32 周；本版本不占用 MVP/RC scope。

## 当前缺口

下表记录 A+B Backend/Release Gate 或未来 C/D/E scope 仍必须完成的验证，不再作为继续扩张 S0
合成测试的队列，也不改变本 Gate 只授权 S1 合同设计的边界：

| 范围 | 已有证据 | 后续切片/发布仍需 |
| --- | --- | --- |
| 语义模型与质量 | SigLIP 2/较小 SigLIP 1 float32 因容量拒绝，dynamic-QInt8 因质量拒绝；SigLIP 1 float16-internal 24/24 pilot ranking 保持；三组生产 100k AI peak 2.860～2.951 GB、ordinary browse +6.0%～14.5%；arm64 hostile graph/6 GB 输出失败关闭；isolated Go/C API、distroless SONAME/受限加载均通过 100 轮取消恢复 | 一轮 storyboard browse +20.34%，六次 baseline/AI keyset 均未过既有绝对预算；无 AI component timing 与 query plan 已确认 count/list 成本来自 FTS candidate scan、rowid 回表及 list 临时排序。order-first 候选在 arm64 100k 的 26 个首页面/22 个第二页 scope/filter/sort/cursor 组合保持 ID 顺序，单页最坏约 203 ms；当前最慢 kind、递归名称排序和 broad modified-window 各 20 次的首页/第二页 P95 最坏 174.591/135.115 ms。三次生产修改窗口约 9.44～19.00/9.55～19.00 s，plan 证明 `assets_modified` 范围扫描驱动、逐行 FTS 探测与临时排序。单库 80k image/10k video/10k animated、image+video 组合，以及两个 50k 库的 mixed-media/date/sparse global tie-break 均已覆盖；仍缺其他组合 kind、真实分布/选择性、全矩阵及 hydration P95、planner threshold 与 amd64，修复属于独立维护 Gate；还缺 1,000 图/100 视频、统一 runtime/native amd64、生产 adapter context/admission/composition、真实 embedding backfill及完整 HTTP/browser/face |
| 许可证与供应链 | ORT 1.28.0 官方 archive/license/notices 已固定 hash；arm64 `cc`/`base-nossl` 最终闭包已生成 CycloneDX 并扫描，no-SSL 对照移除 7 个未使用 OpenSSL High；扫描器漏识别的 ORT `.so` 已补显式 CycloneDX 1.6 component；SigLIP 1 source/两份派生图、YuNet、SFace 也已补 5-component BOM，两份均通过官方 Schema 结构校验且 composition=incomplete；实际 harness/ORT ELF 无剩余三项 glibc 受影响符号的直接 import/精确字符串命中；HNSW 技术候选已拒绝 | ELF 结果只是 VEX 输入；no-SSL arm64 镜像仍有 glibc 1 Critical/2 High，Debian minor/no-dsa 不是自动豁免，需安全 VEX/修复基座；显式组件尚未并入最终镜像/模型 SBOM，仍缺 native amd64/生产镜像扫描、签名 provenance、notice 打包与合规签署；SigLIP 1/YuNet 仍 compliance pending，SFace 精确权重训练数据与商业/再分发澄清未获采信答复，production hold |
| 人脸 | YuNet/SFace load/pipeline smoke；pair/cluster/constraint 合成 scorer；匿名 core/edge → 创建人物/合并 core/单脸确认/cannot-link/换代不覆盖人工关系的合成状态机；隐私草案已列数据分类、默认关闭、访问/清除/备份/诊断/分享与真实数据 Gate；manifest v2 已自动拒绝缺少用途/角色/保留/删除/授权/隐私引用或试图以公开许可替代生物特征授权的数据；隔离 diagnostic DTO 已用封闭字段和危险值拒绝测试证明设计可行 | 合法真实 ground truth、真实授权与 privacy/legal 签署、受控存储/删除演练、detector recall、ROC、核心簇 99.5% precision、生产 transaction/concurrency/删除恢复、生产日志/API/诊断泄露测试、跨库/禁用/五种清除/备份/合法依据 |
| 向量 | exact/SQLite 10k/50k/100k、量化与过滤基线；HNSW 候选拒绝；SQLite generation/active 强杀恢复；arm64 SigLIP 1 + synthetic float16 search/restart 正确；float16-internal 已作三组生产 catalog 100k AI/无 AI 配对，AI peak 2.860～2.951 GB、ordinary browse 均低于 20% 退化门槛；无 AI keyset 已拆分到 count/list，plan 证明 FTS scan/rowid 回表/临时排序 | float32 因 410.6 MB 拒绝；float16 尚缺代表性真实 embedding Recall/双架构容差；既有 keyset 修复设计与原预算复测尚未完成；一轮 storyboard browse +20.34%；仍需真实 embedding backfill、完整 HTTP/browser/face 联合负载、Linux/amd64 和最终 500 MiB 闭包 |
| 模型供应链 | multi-file catalog v2、原子 publish；arm64 下载/DNS/ENOSPC/SIGKILL/Ed25519、SQLite activation；私有 CA TLS/SNI/未知 CA 拒绝、固定地址集显式故障回退与每地址 timeout；package rename/fsync 三阶段强杀与完整包后真实 ENOSPC；anchored managed reconciliation 与真实只读 direct 消失/恢复/source-kind 边界；决策草案已拆分来源/存储并列明 unavailable/backup/upgrade/cleanup/diagnostic | 公网 CA/CDN/DNS TTL 与 resolver/retry 策略、发布 key/checkpoint 运营、host power loss、selected package quota/撤回、生产 migration/direct 配置与 API/UI/备份、native amd64、SBOM/合规、真实在线源/镜像 owner |
| 产品与治理 | feature、CR、技术方案、proposed ADR、静态原型、完整任务拆分及 scope manifest proposal draft 1；草案按 A～E 切片并设置 S0 后 32 单人工程周停损 | 产品签署、实际纳入切片、具名/可执行 owners、隐私决定、ADR 接受、严重风险处置及 Frozen revision 1 |

机器结果和局限见 [INT-001 evidence](../../evidence/int-001/README.md)，可勾选的子项见
[任务清单 S0 已完成证据子项](../../features/intelligent-media-discovery-task-list.md#s0-已完成证据子项)。
这些进展足以支持产品做范围与停损决定，不构成最终模型质量、双架构、合规或生产开发授权。

## 获准下一步

执行 A+B 的 `INT-101～113` S1 合同任务。S1 可以更新权威需求、架构、数据、安全、部署、测试和
OpenAPI source，但不得实现 production backend、migration、AI package 或 UI；这些必须等待
INT-S1 Contract Ready。
