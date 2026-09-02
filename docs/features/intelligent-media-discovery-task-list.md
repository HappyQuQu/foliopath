# FTR-INT-001 本地智能媒体发现开发任务清单

## 当前状态

### 进度总表（截至 2026-09-02）

统计只计算各阶段的 `INT-xxx` 主任务；`INT-xxxA/B` 证据子项、说明段落和静态原型不重复计数。
百分比表示清单勾选率，不等于质量、发布或 Gate 通过率。`make arch-check` 会从主任务复算并校验
本表及两个汇总数字；勾选任务时必须同步更新这里。

| 阶段 | 主任务完成 | 清单进度 | Gate / 授权状态 | 当前结论或下一步 |
| --- | ---: | ---: | --- | --- |
| S0 可行性审计 | 10 / 23 | 43% | A+B 为 **Go**；探索已收口 | 未勾选项作为真实数据、双架构、合规和最终镜像证据台账保留，不再追加同类合成测试 |
| S1 A+B 权威合同 | 13 / 13 | 100% | Contract Ready **Go** | 合同已冻结，允许 S2A 后端实现 |
| S1R2 C+D+E 合同扩展 | 7 / 7 | 100% | Contract Ready **Go** | 合同已冻结；S2B 与 fail-closed S2C 后端均已获实现授权 |
| S2A 模型管理与图片语义搜索后端 | 16 / 16 | 100% | **Backend Ready / Release No-Go** | semantic v2、模型生命周期、搜索/任务/故障矩阵与本地 100k 容量完成；最终模型/native/供应链归 S4 |
| S2B 标签与视频搜索 | 8 / 8 | 100% | **Backend Ready / Release No-Go** | C/D repository、worker、HTTP、ranking/scorer 和失败矩阵完成；governed 质量与最终模型归 S4 |
| S2C 人脸与人物库 | 11 / 11 | 100% | **Backend Ready / Release No-Go** | fail-closed backend、授权 `coser` 功能矩阵、100k×512 与隐私边界完成；最终质量/合规/native 归 S4 |
| S3 消费者与 UI | 11 / 11 | 100% | **Consumer/UI Ready / Release No-Go** | `INT-301～311` 完成；最终模型、质量、双架构、供应链与批准仍归 S4 |
| S4 纵向、容量与发布 | 3 / 11 | 27% | **In Progress / Release No-Go** | 恢复、原件只读、发布文档及原生 baseline 完成；最终模型、质量、联合容量、浏览器/真机及签署仍阻塞 |

汇总口径：

- 当前工作阶段：**S4 In Progress / Release No-Go；S2A/S2B/S2C 保持 Backend Ready，S3 保持
  Consumer/UI Ready**。按
  [CR-2026-022](../changes/CR-2026-022-s2-backend-release-gate-separation.md)，最终审核模型、governed
  semantic/tag/video/face 质量、native 双架构、联合容量、供应链和发布签署仍由 S4 失败关闭。
- revision 2 全 S2 主线台账（S0 + S1 + S1R2 + S2A + S2B + S2C）：**65 / 78（83%）**。
  S0 的未完成项被有意保留，所以该比例不能解释为“产品完成 45%”。
- 全路线主任务（含 S3/S4）：**79 / 100（79%）**。
- 静态交互原型 `INT-000P`：**1 / 1**，只证明产品复核覆盖，不计入生产进度。
- 当前发布判断：**不能发布 AI 功能，也不授权未经 S4 Gate 的发行 UI**。S2 Backend Ready 只允许继续
  S3 合同消费者实现；reviewed catalog 保持为空，face route/runtime 继续缺席。
- 2026-09-01 最终逐项复核完成全部 S2 后端任务；真实模型/数据/native/供应链/签署输入已按 CR-2026-022
  归入 S4，五个最终 verifier 继续对缺失输入失败关闭发布，详见
  [S2 最终阻塞审计](../gates/POST-MVP-5/int-s2-final-blocker-audit-2026-09-01.md)。
- `make verify-intelligent-media-supply-chain` 已把最终 catalog/model package、三类必需 runtime/model
  component、双架构完整 SBOM、签名 provenance、notices、漏洞/VEX 和四方批准绑定到真实文件哈希；
  当前没有这些最终外部产物或签署，所以验收入口完成不改变 Release No-Go 或 S4 任务分子。

2026-08-29 产品用户明确决定“都纳入”，据此冻结
[POST-MVP-5 revision 2](../releases/POST-MVP-5-scope-r2.md)，把 C 标签、D 视频与 E 人脸纳入正式范围，
并接受最多 32 单人工程周的 scope-budget exception。安全、隐私、质量、双架构与供应链 Gate 不降低；
C/D/E 的 `INT-114～120` S1R2 合同工作已完成并通过 Contract Ready；S2A/S2B/S2C 后端随后均已
Backend Ready。汇总分母按 revision 2 范围计算。

2026-08-31 产品用户决定当前执行顺序先跳过人脸测试。该决定只把 S2C 实现/测试暂缓到隐私、法务、
合法 ground truth 与模型许可准入之后，不从 revision 2 范围或最终验收分母删除 E，也不把 S2C 的
No-Go 解释为完成；当前继续推进 S2A/S2B 和非人脸容量、双架构、供应链证据。

2026-08-29 已完成
[ADR-0014 接受审计](../gates/POST-MVP-5/adr-0014-acceptance-audit-2026-08-29.md)：六类门槛中
special-token 失败关闭合同通过，其余五类为 Partial/Blocked，故 ADR 保持提议。该复审没有新的
`INT-xxx` 主任务完成证据，上表进度不变。

2026-09-01 产品授权全部 S2 后端继续并保持发布 fail-closed，ADR-0014 随之接受其 production architecture。
semantic v2 parser、SentencePiece FD adapter、ORT text session、generation-bound text owner、activation smoke
与搜索 composition 已完成，关闭 `INT-202/203/207/208/214` 的实现要求；空 reviewed catalog、质量、原生
双架构容量与供应链签署仍由 `INT-209/210/215` 和 Gate 独立失败关闭。记录见
[semantic v2 production tokenizer 与文本推理闭环](../changes/FIX-2026-09-01-semantic-v2-production-runtime.md)。

同日又复核 Debian 官方安全状态和当前 distroless 双架构子镜像：二者仍指向
`libc6 2.41-12+deb13u3`，三项 glibc finding 仍为 vulnerable，没有可采用的 trixie 修复基座。
证据见 [glibc 安全状态刷新](../evidence/int-001/glibc-security-status-refresh-2026-08-29.md)；这只刷新外部
阻塞事实，不完成新的 `INT-xxx` 主任务，进度总表与 S2A **No-Go** 保持不变。

2026-08-31 再次复核 Debian 官方 trixie package/source 页面，`libc6` 仍为
`2.41-12+deb13u3`，没有可替换当前 runtime closure 的新稳定包；见
[状态刷新](../evidence/int-001/glibc-security-status-refresh-2026-08-31.md)。该时间敏感事实继续维持
供应链 No-Go，不改变清单分子。

同日复核 OpenCV Zoo 官方 SFace README 与 exact-weight issue #313：README 仍声明目录文件使用
Apache-2.0，但 issue 仍开放且没有 maintainer 对训练数据来源、商业推理和权重再分发问题作出答复；Debian
trixie 的 `libc6` 同样仍为 `2.41-12+deb13u3`。见
[外部阻塞刷新](../evidence/int-001/s2-external-blocker-refresh-2026-08-31.md)。最终权重与供应链 Gate 不变。

2026-08-28 的合同消费者维护已恢复并通过 `make test-web-e2e`（7 passed、4 skipped），真实 Docker
后端、Chromium / Pixel 5 项目及媒体只读 sentinel 均通过。该结果只维护现有纵向回归，不完成任何新的
`INT-xxx` 主任务，因此当时的进度数字与 S2A **No-Go** 均不变。证据见
[真实浏览器 E2E harness 与当前界面合同同步](../changes/FIX-2026-08-28-browser-e2e-harness.md)。

目标版本是已冻结的 [`POST-MVP-5` revision 2](../releases/POST-MVP-5-scope-r2.md)。当前
[INT-S0](../gates/POST-MVP-5/int-s0-architecture-ready.md) 已对 Slice A+B 为 **Go**；A+B 的 S1 合同
`INT-101～113` 与 C/D/E 的 S1R2 合同 `INT-114～120` 均已完成。S2A/S2B 已实现其当前获授权的生产
后端范围，并在 CR-2026-022 后签署 Backend Ready / Release No-Go；S3 可消费合同，但发行仍须等待 S4 Gate。
下面所有 `[ ]` 都表示没有完成证据，不能因已有方案文档改成 `[x]`。

S0 本地探索已于 2026-08-27 收口，当前权威下一步是
[INT-S0 收口与阻塞清单](../gates/POST-MVP-5/int-s0-closeout-and-blockers.md)。未完成主任务保留为证据
台账，不再逐项追加合成测试；真实数据、amd64、合规与最终镜像验证已归回对应切片 Gate。

粗略规模：单人 5～8 个月；S0 4～8 周。建议拆成四个可独立删减的生产 slice：

```text
S0 可行性
  → S1 合同与模型管理
  → S2A 图片语义搜索
  → S2B 标签建议 + 视频代表帧搜索
  → S2C 人脸聚类与人物库
  → S3 UI
  → S4 目标容量与发布
```

- [x] `INT-000P` 建立不进入生产依赖图的静态交互原型，覆盖语义搜索、标签审核、后台匿名聚类、
  创建人物、匿名组并入人物、单 face 归类、按库设置、模型下载/离线发现/直接读取和清除边界。
  - 证据：[发现原型](../../prototypes/apple-redesign/12-ai-features.html)、
    [智能设置原型](../../prototypes/apple-redesign/13-settings-ai.html)；它们只用于产品复核，
    不证明模型、API、数据或性能可行。

## 执行规则

- 后端优先；S0 Go 前不得改生产合同，S1 Go 前不得实现生产后端，S2 对应 slice Go 前不得接 UI。
- 每项完成必须同时有实现、自动化验证、文档/风险同步和明确证据链接。
- 原媒体只读，测试只用合成/公开许可/授权 fixture，不读开发者真实媒体库。
- 若模型、权重、运行时或索引许可证不清，对应任务停止；不以“能下载”视为可分发。
- 若人脸核心聚类精度不达标，执行已定义降级，不调低 precision 门槛来宣布完成。
- 不以“UI 有下载按钮”视为镜像已存在。国内镜像需要运营 owner、签名清单、可用性、撤回与成本证据；
  否则发布范围只保留 `/models:ro` 离线安装。
- 同一风险已有足够的通过/失败关闭证据后停止追加测试；新测试必须能直接改变范围、候选或架构决定。
- 没有合法真实数据、原生 amd64 或合规签署时登记为外部阻塞，不用更大合成集替代。

## S0 已完成证据子项

下面的 `[x]` 只表示对应隔离证据已经落地，不表示所属 `INT-xxx` 主任务或 INT-S0 Gate 完成。

- [x] `INT-004A` 合成语义 manifest、生成器、严格校验和 Recall@1/3、MRR scorer smoke。
- [x] `INT-004B` 10 张 Wikimedia Commons 公开许可 Cosplay pilot：固定 revision/作者/许可/hash、
  临时下载复核器、24 条中英文 relevance judgment 和多相关 Top-3 recall。
- [ ] `INT-004C` 至少 1,000 张代表性合法图片、100 视频及困难负例/失败分类验收集。
- [x] `INT-005A` 人脸 pair ROC、cluster/core precision/recall、cannot-link/manual assignment 合成 scorer。
- [x] `INT-005B1` 非合成评测数据强制使用 intake manifest v2；真实人脸数据自动校验限定用途/访问角色、保留期限、
  删除方式、不可再分发和不透明授权/隐私评审引用；拒绝把公开图片许可当作生物特征处理授权。仓库中的
  文件只是占位模板，不构成真实授权、隐私签署或质量数据。证据：[manifest v2 validator](../evidence/int-001/dataset-governance-manifest-v2-2026-08-27.md)。
- [ ] `INT-005B` 经过隐私评审的合法真实人脸 ground truth。
- [x] `INT-015A` 隔离 diagnostic snapshot 使用封闭字段，只允许模型 ID/version/hash、稳定状态/错误码、
  固定计数与资源指标；精确 JSON shape 和危险原始值已有回归测试。它尚未接生产日志、API 或诊断包，
  不关闭 `INT-015/406`。证据：[diagnostic privacy contract](../evidence/int-001/ai-diagnostic-privacy-contract-2026-08-27.md)。
- [x] `INT-006A` 固定 SigLIP 2/SigLIP 1 revision、权重 hash、完整运行包文件和初步许可来源。
- [x] `INT-006B` 两候选离线合成集与公开许可 pilot 的中英文质量/吞吐/RSS 对照。
- [x] `INT-006C` 512px WebP/quality 82 Pillow surrogate 三次新进程复测；排名稳定、RSS 回落。
- [x] `INT-006D` 当前生产 govips adapter 在 Linux/amd64 QEMU 生成同一 pilot 输入；尺寸与查询排名
  稳定，并记录相对 Pillow 的像素差；这是开发证据，不是 native Gate。
- [x] `INT-006E1` 未修改 Dockerfile 的原生 Linux/arm64 `make test-libvips` 通过；同一 pilot 输出与
  amd64 QEMU 逐文件 byte-identical。
- [x] `INT-006E1A` ONNX 来源复核：Google 仓库仅有未合并 PR ref；ONNX Community 为第三方转换，
  不进入受信下载源。生产候选必须由固定 Google revision + 固定 exporter/opset/shapes 自行导出、验证、
  SBOM、hash 和项目签名。
- [x] `INT-006E1B` 固定 Google revision、完整 exporter 依赖与 opset 18 自导出三次 byte-identical；
  macOS arm64 ORT 1.29.0 和原生 Linux/arm64/no-network ORT 1.28.0 的四输出均在 `1e-4` 容差内。
  单语义 session Linux 峰值约 2.19 GiB，且 tracer 只支持批准固定 224×224/length-64 形状；三次
  100-call runtime smoke 还覆盖四类错误输入、执行中取消、取消后恢复和 warm-up 后 RSS 增长上限。
- [x] `INT-006E1C` 固定形状 image/text split graph 两次 byte-identical，均匹配 PyTorch；原生
  Linux/arm64 public pilot 三次保持排名，image/text P95 优于单体图。单体图因每次执行两支被拒绝；
  split 仅进入下一轮 session 调度/卸载/完整容量验证。
- [x] `INT-006E1D` 原生 Linux/arm64 运行 30 轮 image/text session load→infer→close；输出正常但
  cycle RSS 相对首轮保留增加 213,004 KiB，超过 131,072 KiB smoke 门槛，明确判定失败，不放宽门槛。
- [x] `INT-006E1E` 保留 E1D 失败和原门槛，关闭 ORT CPU memory arena 后重跑 30 轮：cycle RSS
  增长 0 KiB、峰值降至 2,046,084 KiB；三次 public pilot 排名不变、延迟无明显回退。后续 arm64
  spike 必须固定 arena off，仍不等于 production runtime 选型。
- [x] `INT-006E1F` 原生 Linux/arm64、4 CPU/4 GiB/no-network 三次让真实 split image encoder 与
  100k float16 SQLite backfill/search/browse/cancel/restart 同时运行；容器峰值 1.29～1.33 GB、search
  P95 165.443～186.532 ms、恢复均到 100,000 行，但 browse 相对退化 5.32～12.74 倍，保持失败。
- [x] `INT-006E1G` arena-off split image/text load→infer→close 延长到 100 轮；进程 cycle RSS 仅增
  28 KiB，但容器 `memory.peak` 3,719,651,328 bytes 超过 3.2 GiB 门槛，拒绝反复完整模型 reload 策略。
- [x] `INT-006E1H` arena-off image/text 双 session 常驻并交替推理 100 轮；cgroup current 稳定约
  3.56 GB、peak 4,008,951,808 bytes，超过 3.2 GiB 且几乎耗尽 4 GiB，拒绝双 session 常驻策略。
- [x] `INT-006E1I` 较小 SigLIP 1 候选 split 两次 byte-identical 且 PyTorch `1e-4`、macOS/Linux
  arm64 Top-3、float16 ranking 通过；双 session 100 轮 peak 2.18 GB，叠加 100k float16 三次 peak
  2.364～2.370 GB、search/restart 通过，但中文 pilot 有一次 Top-1 miss 且 browse 相对 Gate 失败。
- [x] `INT-006E1J` SigLIP 1 双 session 旁路运行当前生产 10k 目录/100k 文件 catalog capacity：普通
  recursive browse 相对基线 +11.2%，否定 micro-proxy 数倍退化的代表性；但容器 peak 3.590 GB 超过
  3.2 GiB，global/storyboard browse 单次对照超过 20%，且无模型 search-keyset 自身也未过既有预算。
- [x] `INT-006E1K` SigLIP 1 dynamic-QInt8 MatMul/Gemm 两次 byte-identical、总图约 281 MB；但
  macOS/Linux 仅 8/24 Top-3 一致，Linux 中文 Recall@1 从 0.917 跌至 0.25，质量失败，候选拒绝。
- [x] `INT-006E1L` SigLIP 1 float16-internal/float32-I/O 两次转换一致，24/24 pilot Top-3 跨
  float32/float16-model 与 macOS/Linux 相同；双 session 100 轮 peak 1.614 GB，生产 10k/100k peak
  2.906 GB且 ordinary browse +14.5%，但 global search 单次 +41.6%、baseline keyset 仍失败。
- [x] `INT-006E1M` 再补两组 fresh no-AI/float16-AI 生产 10k/100k 配对：三次 AI peak
  2.860～2.951 GB、ordinary browse +6.0%～14.5% 均通过；global outlier 未复现，但一轮 storyboard
  browse +20.34%，且六次 baseline/AI search-keyset 均未过既有 250 ms 绝对预算。
- [x] `INT-006E1N` 在无 AI runtime 的原生 Linux/arm64 生产 10k/100k 基线中拆分 search-keyset：
  两页合计 P95 358.623 ms，第一页/第二页为 167.755/190.937 ms；repository count 为 66.987 ms，
  list 为 106.750/130.316 ms。既有 250 ms 门槛未改；重复 count 与 list 都是实质成本，单一 SQL
  根因在本子项尚未由 query plan 证明，后续由 `INT-006E1O` 关闭该诊断缺口。
- [x] `INT-006E1O` 同一生产 query shape 的原生 Linux/arm64 `EXPLAIN QUERY PLAN` 证明 count/list
  均由 FTS 虚表扫描候选并按 rowid 回表，首/次页 list 都使用临时 B-tree 排序；第二页 keyset 未改变
  执行形态，现有 folder/name expression index 不产生搜索顺序。只加普通索引不足；未改生产 SQL/预算。
- [x] `INT-006E1P` benchmark-only broad-query order-first SQL 复用既有 browse expression index，原生
  Linux/arm64 100k 下前后两页各 101 个 ID 与生产结果完全一致；两次扩展运行的最坏 ID-selection P95
  33.990/32.451 ms、完整 hydration 33.668 ms、稀疏 `asset-099` 110.061 ms，取消 2.332 ms，计划均无
  临时排序。但未覆盖全部 scope/filter/sort/order、混合阈值和 amd64，只保留为
  独立 maintenance Gate 候选，不改生产实现。
- [x] `INT-006E1Q` order-first correctness matrix 在原生 Linux/arm64 10k/100k 覆盖 library/global/
  root-recursive/direct/subtree、name/modified/size 双向组合、非空 image/video/animated、image+video
  组合、短词、修改时间窗口及游标；26/26 首页面和 22/22 第二页与生产 repository ID 顺序一致，
  候选单次页面最坏 202.268/155.230 ms。
  三次生产修改窗口首/次页达 9.443～18.996/9.545～19.002 s；plan 证明 `assets_modified` 范围扫描
  驱动后逐行探测 FTS 并临时排序。矩阵含约 80k 图片/10k 视频/10k 动图，但仍为单库，其他组合
  kind、单次时延、矩阵全 hydration、真实选择性、amd64 与 planner threshold 仍缺，生产不变。
- [x] `INT-006E1R` 两个不重叠媒体库各 5k/50k，经真实 library service + scanner 建库后验证 global
  library-ID tie-break；加入不同 image/video/animated 比例、image+video、选择性日期和约 2% 稀疏词后，
  name/modified/size 双向等 11/11 首页与 11/11 第二页顺序一致，候选最坏 82.253 ms。真实分布和选择性、
  单次时延/full hydration/amd64 仍缺。
- [x] `INT-006E1S` 对当前最慢 kind、递归名称排序和 broad modified-window 三个候选代表场景的首页/
  第二页各重复 20 次；原生 Linux/arm64 100k 的 P95 最坏 174.591/135.115 ms，均低于未变的 250 ms
  隔离页预算。其余矩阵仍是单次观测，不能据此宣称完整矩阵 P95 或修改生产 SQL。
- [ ] `INT-006E2` 原生 amd64 govips、ONNX 双架构 embedding 容差、完整进程与浏览并发比较。
- [x] `INT-007A` 固定 YuNet/SFace revision/hash，完成随机张量循环与临时许可图片 pipeline smoke。
- [ ] `INT-007B` 合法真实数据的 detector recall、pair ROC、聚类 precision/recall 和 cosplay 失败分类。
- [x] `INT-008A` SigLIP 1 float16 split 在 macOS/arm64 与原生 Linux/arm64 拒绝空/随机/截断模型及
  图片/文本错误 shape/dtype，子进程无超时或信号崩溃，错误后正常推理恢复；不覆盖有效恶意图、
  native amd64、C/Go adapter、输入字节 admission 或 adapter 取消。
- [x] `INT-008B` macOS/arm64 与原生 Linux/arm64 对 protobuf-valid fixture 均允许同目录 external-data
  控制模型推理，同时拒绝 `../` external-data、未知算子和循环图，子进程无超时/信号崩溃；runtime
  结果只作纵深防御；据此收敛首版 graph 必须内嵌 initializer，发行校验拒绝全部 external-data。
- [x] `INT-008C` 原生 Linux/arm64 在 1/1.5/2 GiB 子进程地址空间上限下保持控制图正常，并让固定
  6 GB 输出请求返回 `RuntimeException`，无 timeout/signal；512 MiB 因控制也失败而明确排除。RLIMIT
  不是生产方案，正式边界仍是发行图校验、固定 shape 与 exact hash。
- [x] `INT-008D` 独立 Go/cgo C API harness 在原生 Linux/arm64 完成 control/错误码、100 轮
  cancel→recovery 和 30 轮 race build；取消 P95 6.56/6.69 ms，RSS 增长 17,404/80,364 KiB，均低于
  128 MiB。ORT 原始错误含模型路径，harness 只输出稳定码；不等于生产 adapter 或 native amd64。
- [x] `INT-008E` 固定 digest 的 distroless arm64 镜像补齐 ORT SONAME 后，在 non-root/read-only/
  no-network/cap-drop/no-new-privileges 下完成 100 轮取消恢复；`base-nossl` 最小闭包同样通过并移除 7 个
  OpenSSL High 发现。Grype 仍报 glibc 1 Critical/2 High；扫描器漏掉的 ORT `.so` 后续已补 arm64
  显式 component，但尚未合并最终镜像 SBOM。因此只关闭 arm64 ABI/受限加载子问题，不通过发布 Gate。
- [x] `INT-012A` 合成状态机覆盖后台匿名 core/edge、从 core 创建人物、匿名 core 并入人物、单 face
  确认、cannot-link、换代重聚类不覆盖人工关系和稳定匿名组 ID；edge 不随批量操作写入人物库，
  stale plan/cannot-link 冲突不会留下空人物或部分 assignment。
- [x] `INT-009A` 10k/50k/100k exact、SQLite BLOB、过滤、稳定 tie-break 和 float16/int8 随机向量基线。
- [x] `INT-009B1` SQLite/WAL replacement generation + active pointer 同事务真实强杀；macOS/arm64 与
  原生 Linux/arm64 重启后旧代完整、新代不可见、integrity ok，并可重新原子构建/切换。
- [x] `INT-009B2` 原生 Linux/arm64、4 CPU/4 GiB/no-network 三次 100k×512 float32 bounded backfill +
  exact search + keyset browse proxy + cancel/restart 测量；随后同矩阵 float16 三次。两者 search P95 通过
  但 browse 相对退化 Gate 失败；float32 410.6 MB 被拒绝，float16 136.9 MB 仅保留为待真实质量验证路径。
- [x] `INT-009B3` 生产 govips 输入的 10 图/24 查询 public pilot 在 macOS 与原生 Linux arm64 ONNX
  排名一致；float16 图片向量保留全部 24 个 Top-3 和评分指标。样本过小且查询偏简单，不关闭真实
  embedding Recall Gate；单体 ONNX 每次执行图文两支，不能作为最终 runtime layout。
- [ ] `INT-009B` 真实 embedding、Linux 双架构、并发浏览、强杀/恢复和最终空间预算。
- [x] `INT-010A` `coder/hnsw` 10k 三档 recall/build/query、截断拒绝与重建比较；当前候选被拒绝。
- [x] `INT-010B` 条件分支未触发并关闭：Linux/arm64 100k float16 exact search P95 已在 250 ms 内，
  当前 HNSW 候选又因 recall/build/确定性被拒绝；在真实 embedding 或 amd64 证明 exact 不达标前，
  不引入第二套 ANN/native 供应链。
- [x] `INT-011A` 合成 4 帧故事板 mean/max 聚合代理比较。
- [x] `INT-011B1` 原生 Linux/arm64 使用项目固定 FFmpeg 7.1.5，经现有生产
  `videoffmpeg.Processor.Storyboard` 跑通 4/10 帧计划、统一结果校验和源 hash 不变；证明可复用唯一
  media adapter，不证明真实内容质量。
- [ ] `INT-011B2` 至少 100 个合法代表性真实视频的抽帧/语义检索质量、聚合策略、双架构与联合负载。
- [ ] `INT-011B` 真实视频 4/10 帧、抽帧质量和复用现有 FFmpeg admission。
- [x] `INT-014A` ONNX Runtime、SigLIP、YuNet/SFace、Chinese-CLIP、InsightFace 和 HNSW 初步许可筛选。
- [x] `INT-014B1` arm64 `cc`/`base-nossl` 最终闭包生成 CycloneDX/Grype 报告；no-SSL 删除 7 个
  OpenSSL High。实际 harness/ORT ELF 对剩余三项 glibc 受影响符号无直接 import/精确字符串命中，
  但这只是 VEX 输入，不是安全审批，也不覆盖生产 composition/native amd64。
- [x] `INT-014B2` 为扫描器漏识别的官方 ORT 1.28.0 Linux/arm64 `.so` 生成显式 CycloneDX 1.6
  component，绑定 library/archive/source/license/notices hash，并通过官方 Schema 结构校验；composition
  明确为 incomplete。它尚未并入最终镜像 SBOM、覆盖 amd64、关联漏洞/VEX 或签名 provenance。
- [x] `INT-014B3` 为 SigLIP 1 源权重、两份派生 float16 ONNX、YuNet 和 SFace 生成 5-component
  CycloneDX 1.6 清单并通过官方 Schema 结构校验；SigLIP 1/YuNet 明确为 compliance pending，SFace
  明确为 production hold，composition=incomplete。它不等于权重再分发批准或最终签名模型 SBOM。
- [x] `INT-014B4` 为 exact arm64 text-runtime 镜像实现隔离、失败关闭的 CycloneDX 合并器；补入扫描器
  漏识别的 ORT/SentencePiece，扁平化 Scout 不稳定的 package→file 嵌套，合并重复 dependency，并拒绝
  dangling ref/重复组件/错架构。两次 fresh scan 规范化后均为 1,267 components/18 dependency rows，
  输出 SHA-256 同为 `6e340ca7…`。同一流程及 amd64 显式组件在 QEMU 构建镜像的两次 fresh scan 上也
  byte-identical（1,267/18，`da490242…`）；两架构 composition 均为 incomplete，amd64 仍不是 native
  证据，因此不等于最终双架构签名 SBOM。
- [x] `INT-014B5` 使用固定 Grype 0.116.1 与关闭自动更新的有效 2026-08-26 DB 扫描 exact arm64/amd64
  text-runtime 镜像；两边均为 15 findings，且 `libc6` 2.41-12+deb13u3 同为 1 Critical/2 High、无可用
  fixed version。扫描器仍漏识别 ORT/SentencePiece；不得据此 suppress，amd64 仍仅 QEMU package evidence，
  `INT-014B` 与发布 Gate 保持未勾选。
- [x] `INT-014B6` 将旧 harness/ORT ELF 检查扩展到 app、ORT、SentencePiece、libstdc++、libgcc_s 的
  external non-glibc closure。两架构均未直接导入受影响 scanf/DNS debug 函数，但 exact libstdc++ 均直接
  导入 `ungetwc`，调用点位于 `stdio_sync_filebuf<wchar_t>` 的 underflow/pbackfail。故旧“无直接 affected
  import”不能覆盖 `CVE-2026-5928`；未签 VEX，发布 Gate 继续失败。
- [x] `INT-014B7` 构建双架构 `LD_PRELOAD` `ungetwc` tripwire；专用 control probe 在 arm64/amd64 均
  打印 marker 并 exit 86，证明拦截生效。相同受限 profile 下固定 31-case tokenizer→text encoder 在
  两架构均通过且未触发 tripwire；只证明当前固定路径/default locale，不能覆盖未来生产 composition、
  其他 locale/编码或签署 VEX，发布 Gate 不变。
- [ ] `INT-014B` 入选 runtime/权重/引擎的 SBOM、漏洞、notices、再分发与正式合规签署。
  - 2026-08-31：新增 `make verify-intelligent-media-supply-chain`，实际复算 catalog、最终 model package、
    component notices、native amd64/arm64 complete SBOM、provenance、signature verification、漏洞报告和
    可选 VEX 文件的 SHA-256；拒绝路径逃逸/symlink、缺架构、不完整 SBOM、未验证签名、缺再分发批准，
    以及存在 Critical/High 却没有 VEX + security approval 的输入。当前没有最终产物与四方签署，因此
    入口不等于 `INT-014B` 完成。
- [x] `INT-018A` 依据已明确产品行为形成 `POST-MVP-5` scope manifest proposal draft 1，拆分
  A～E 独立切片、32 单人工程周停损、owner/合同/Frozen 条件和 face/mirror 删除 fallback；草案未签署，
  不等于 `INT-018` 或 scope 冻结。
- [x] `INT-018B` scope proposal 已逐项列明 S1 必须更新的 PRD、user flow、UI、architecture/ADR、
  data/migration、security、deployment、testing、OpenAPI/generated client、traceability 文件与 task；
  当前只登记影响，不提前修改生产合同。
- [x] `INT-019A/021A` 已形成[模型分发、存储与恢复决策草案](../gates/POST-MVP-5/int-019-021-model-distribution-review.md)：
  拆分来源/运行存储，拒绝内置权重和任意 URL，冻结候选 managed/direct unavailable、backup、upgrade、
  cleanup、diagnostic/provenance 语义；真实在线源 owner、quota、撤回和生产合同仍未签署。
- [x] `INT-020A` multi-file catalog schema v2、整包 digest/校验、Linux `openat2` scanner 骨架、
  SFace 断点续传和 macOS no-replace package publish。
- [x] `INT-020B1` 隔离审核源下载状态机：固定 origin/ETag/size/hash、稳定及传输中取消后续传、
  跨 origin 重定向拒绝、loopback 默认拒绝、配额、错 hash 不发布、no-replace；macOS/arm64 与
  原生 Linux/arm64 `httptest` 通过；原生 arm64 真实 `ENOSPC` 不发布候选，真实子进程 `SIGKILL`
  后由新进程续传并精确发布。
- [x] `INT-020B2A` catalog transport 单次解析并固定已审核公网 IP，公私混合/CGNAT/特殊网段整体拒绝，
  IPv4-mapped IPv6 归一化，禁用环境 proxy，resolver 错误/空 answer 单次失败且不 dial；macOS/arm64
  与原生 Linux/arm64 通过。
- [x] `INT-020B2A1` 私有测试 CA + `models.example.test` 证书完成真实 TLS/SNI 握手，未知 CA 失败关闭；
  固定 DNS 地址集合首地址显式失败后回退第二地址并完成 ETag/size/hash 发布。macOS/arm64 与受限
  Linux/arm64 通过；阻塞 dial 还证明每地址 5 秒/更短外层 deadline 后回退。公网 CA/CDN/DNS TTL
  与外层重试仍未覆盖。
- [x] `INT-020B2B1` Ed25519 domain-separated 签名 catalog、严格 envelope/大小/时间窗、未知 key、
  篡改、rollback 和同序号不同内容拒绝；macOS/arm64 与原生 Linux/arm64 通过。
- [x] `INT-020B2B2A` 真实子进程分别在 package rename 前、rename 后/parent fsync 前、fsync 后强杀；
  macOS/arm64 与原生 Linux/arm64 重启后只出现完整 staging 或完整 final，可安全重试/对账。
- [x] `INT-020B2B2B` 原生 Linux/arm64 在 package 完整落盘后将 128 KiB tmpfs 填至真实 `ENOSPC`；
  same-filesystem no-replace rename + parent fsync 仍原子发布完整 final，active 不变；测试同时约束失败闭合分支。
- [ ] `INT-020B2B2` 公网 CA/CDN/DNS TTL 与 resolver/retry 策略、发布 key
  托管/轮换/撤回与 durable
  checkpoint 运营、host power-loss durability、生产 active-generation owner/
  FS-DB reconciliation 及原生 Linux/amd64 完整矩阵。
- [x] `INT-021A` 隔离 package generation staging → no-replace 原子发布 primitive。
- [x] `INT-021B1` 隔离 SQLite published-generation registry；catalog checkpoint 与 active pointer 单事务，
  仅接受 Ed25519 verifier 的内部认证结果，注入失败整体回滚、旧 active 保留、幂等与重启持久化；
  macOS/arm64 与原生 Linux/arm64 通过。
- [x] `INT-021B2A` 原生 Linux/arm64 anchored scan → registry reconciliation：精确 orphan 只登记不激活，
  缺失/损坏标 unavailable 但保留 active/checkpoint，精确恢复后重新 available，未知目录不删除。
- [x] `INT-021B2B1` 原生 Linux/arm64 真实只读 bind mount direct 生命周期：source kind 不可变，消失
  只标 unavailable，精确 remount 恢复，不复制/删除/自动切 active；仅测试容器临时使用 mount capability。
- [ ] `INT-021B2B2` 生产 migration/事务 owner、部署期 direct 配置与运行期异常、key/checkpoint 运营、
  回滚、配额、备份和 unavailable 查询/API/UI 状态所有权及原生 Linux/amd64。

证据索引：[INT-001 evidence](../evidence/int-001/README.md)。

## S0 审计台账（本地探索已收口）

> 本节是完整审计台账，不再代表逐项执行队列。当前执行入口只有
> [已接受的五项产品决定](../gates/POST-MVP-5/int-s0-closeout-and-blockers.md#产品用户决定已接受)。

- [x] `INT-001` 产品负责人复核并接受 revision 1 只纳入 A+B；C/D/E 改为后续独立 scope revision，
  全部非目标保持。
  - Owner：产品负责人；证据：签署后的 Change Record/Gate。
- [x] `INT-002` 决定未来人物库为实例级，face observation/匿名组/资产关系按库隔离，仅允许用户显式
  跨库合并人物；E 不进入 revision 1。
  - Owner：产品 + face owner；依赖：INT-001。
- [x] `INT-003` semantic 与未来 face 按库独立开关；默认只清除可重建派生数据，人物名与人工关系必须
  二次确认才能清除。
  - Owner：产品 + 隐私负责人；依赖：INT-001。
- [ ] `INT-004` 建立合法的语义质量数据集 manifest、查询标注和评分脚本。
  - Owner：QA/ML；证据：生成/来源/许可/哈希，不提交受限内容。
  - 2026-08-25：严格 manifest schema/validator、8 张 CC0 程序绘制图、16 条中英文查询和 Recall/MRR
    scorer smoke 已落地；至少 1,000 张代表性合法质量集仍未完成。
  - 2026-08-26：新增 10 张 Wikimedia Commons 公开许可 Cosplay 照片的固定 revision/作者/许可/hash
    manifest、只写临时目录的元数据复核下载器、24 条中英文配对标注和多相关 Top-3 recall。它只完成
    pilot，不满足至少 1,000 张及代表性场景/失败分类要求，保持未勾选。
- [ ] `INT-005` 建立合法的人脸 ground-truth 数据集 manifest 和聚类评分脚本。
  - Owner：QA/ML；依赖：隐私评审。
  - 2026-08-25：pair ROC/cluster/core/cannot-link/manual scorer 与正反合成 fixture 已完成；合法真实 ground truth 未完成。
  - 2026-08-27：schema v2 validator/template 已把允许用途、访问角色、保留、删除、不可再分发、授权与
    隐私评审引用变成自动门槛，并拒绝用公开许可替代生物特征授权；模板无真实数据且引用为占位符，
    所以未关闭真实来源、签署、受控存储、删除演练或质量验收，主任务保持未勾选。
- [ ] `INT-006` 比较至少两个视觉-文本候选的中英文质量、吞吐、RSS、模型大小和权重许可。
  - Owner：inference/semantic；依赖：INT-004。
  - 2026-08-25：SigLIP 2 candidate A 与较小的 SigLIP 1 candidate B 均已固定 revision/hash，完成离线
    macOS arm64 PyTorch/F32 合成比较；B 的 RSS 更低但吞吐无优势。代表性质量、ONNX/Linux 与整进程
    并发仍缺失，保持未勾选。
  - 2026-08-26：公开许可 10 图 pilot 中，SigLIP 2 中文/英文 Recall@1 均 1.0，SigLIP 1 为
    0.917/1.0；宽泛多相关查询两者均有漏召回。直接原图解码 RSS 不代表生产输入链；仍需 1,000 图、
    受控缩放、ONNX/Linux 和 4 GiB 联合容量，不能据此选型，保持未勾选。
  - 2026-08-26：512px WebP/quality 82 Pillow surrogate 三次新进程复测保持全部 Top-1/首个相关项
    排名，模型进程 RSS 中位数相对原图直解码约下降 31%/43%。它未运行生产 libvips，不补齐受控生产
    输入、Linux/ONNX、浏览并发或 1,000 图质量要求，保持未勾选。
  - 2026-08-26：当前生产 govips adapter 已在 Linux/amd64 QEMU 对同一 10 图生成 512px WebP；
    两候选 24 条 Top-1/首个相关项排名相对原图和 Pillow 均稳定。QEMU 不是原生双架构，PyTorch
    macOS 推理也不是 ONNX Linux 整进程证据，保持未勾选。
  - 2026-08-26：原生 Linux/arm64 完整 `libvips-test` target 通过；arm64 native 与 amd64 QEMU 的
    10 个 WebP byte-for-byte 相同。原生 amd64、ONNX embedding 和整进程仍缺，主任务保持未勾选。
  - 2026-08-26：上游 ONNX 来源复核拒绝将 Google 未合并 PR ref 或单一贡献者维护的 ONNX Community
    转换仓库作为受信发行源；后续必须从已固定 Google revision 自行可复现导出并以项目签名发布。
    原生 amd64/双架构完整数值验证尚未完成，主任务保持未勾选。
  - 2026-08-26：固定源权重、PyTorch/Transformers/Optimum ONNX/Accelerate/ONNX、opset 18 和
    224×224/length-64 后三次自导出 byte-identical；macOS arm64 ORT 1.29.0 与原生 Linux/arm64
    no-network ORT 1.28.0 均匹配 PyTorch 四输出，最大绝对误差约 `1.05e-5`。Linux 单 session 峰值
    约 2.19 GiB，动态形状有 tracer warning；100-call smoke 的错误输入/取消/恢复与 RSS 增长通过，
    但 amd64、统一生产 runtime、真实 Recall、C/Go adapter、hostile graph、长循环/重复加载和完整
    4 GiB 进程仍缺，主任务保持未勾选。
  - 2026-08-26：固定 1×3×224×224 image 与 1×64 text 的 split graph 两次 byte-identical，大小
    371.7/1,129.3 MB；原生 Linux/arm64 三次 image P95 90.2～97.5 ms、text P95 28.9～37.0 ms，
    排名不变。单体图相同输入 P95 为 118.2/131.9 ms，正式拒绝；split 顺序卸载/加载仍达到约 2.03 GiB
    HWM，必须继续验证 scheduler/并发/长期卸载和生产 adapter，主任务保持未勾选。
  - 2026-08-26：30 轮 split session 轮换的 RSS 从首轮 537,752 KiB 上升至末轮 750,756 KiB，
    早期跃升后在较高平台波动，峰值 2,305,768 KiB。它未证明无限泄漏，但已证明 close 不会恢复基线并
    失败 128 MiB retained-growth 门槛；必须更换/配置 lifecycle 或缩小调度方案，主任务保持未勾选。
  - 2026-08-27：未修改 128 MiB 门槛，关闭 ORT CPU memory arena 后同一 30 轮 cycle RSS 固定
    509,360 KiB、增长 0 KiB、峰值 2,046,084 KiB；三次 public pilot image/text P95 分别
    92.8～99.6/29.8～32.9 ms，24 条 float16 Top-3 全部不变。该配置解除 arm64 allocator 子阻断，
    但 amd64、production adapter enforcement、完整进程并发和长周期仍缺，主任务保持未勾选。
  - 2026-08-27：延长到 100 轮后 retained process RSS 仍只增 28 KiB，但 cgroup peak 达
    3,719,651,328 bytes，超过 `R-024` 3.2 GiB 门槛。arena-off allocator 子证据保留，反复完整加载两图
    的生命周期方案判失败；必须比较有界常驻 session 或隔离 worker，主任务保持未勾选。
  - 2026-08-27：双 session 常驻 100 轮没有持续增长，但 cgroup current/peak 约 3.56/4.01 GB，
    `memory.stat` 约 1.90 GB anon + 1.65 GB file；在 SQLite/HTTP/人脸尚未加载前已无安全余量，方案
    判失败。当前 SigLIP 2 没有通过 4 GiB 的 lifecycle，必须转向更小模型或删除双 encoder 共存需求。
  - 2026-08-27：SigLIP 1 split 两次导出一致，image/text 为 371.7/441.2 MB；arm64 双 session 100 轮
    peak 2.18 GB，叠加 100k float16 三次 peak 2.364～2.370 GB且恢复完整。它转为资源优先候选，但
    10 图 pilot 仍有一次中文 Top-1 miss，1,000 图质量、amd64、production runtime 和 browse Gate 未过，
    不能选型，主任务保持未勾选。
  - 2026-08-27：dynamic-QInt8 将 image/text 降至 95.8/184.8 MB，Linux peak 约 811 MB且推理更快；
    但 native Linux 中文 Recall@1/3 降至 0.25/0.5，macOS/Linux 仅 8/24 Top-3 相同。该配置在容量前
    即正式拒绝，不能用资源收益抵消质量/跨 runtime 失败，主任务保持未勾选。
  - 2026-08-27：float16-internal 将两图降至 186.0/220.7 MB，24 条 pilot 排名全部保持；双 session
    100 轮 peak 1.614 GB、生产 10k/100k peak 2.906 GB，低于 3.2 GiB。CPU 推理变慢，global search
    单次相对 +41.6%，且只一组配对/不同 ORT minor/小质量集，转为资源优先候选但不能选型。
  - 2026-08-27：扩为三组 fresh production 配对后，AI peak 2.860～2.951 GB、ordinary browse
    +6.0%～14.5% 均稳定通过；global +41.6% 未复现，但 storyboard 一轮 +20.34%，六次 keyset 绝对
    预算均失败。arm64 资源子证据保留，完整 Gate 仍不通过。
- [ ] `INT-007` 比较人脸 detector/embedding 候选的检测、ROC、聚类质量、RSS 和权重许可。
  - Owner：inference/face；依赖：INT-005。
  - 2026-08-25：YuNet/SFace 已固定 revision/hash 并完成随机张量与临时图片 pipeline smoke；ROC、聚类质量和合法 ground truth 未完成。
- [ ] `INT-008` 在原生 linux/amd64 与 arm64 验证候选 runtime 的加载、取消、泄漏、畸形输入和无网络运行。
  - Owner：inference/release；依赖：INT-006、INT-007。
  - 2026-08-27：SigLIP 1 float16 split 已在 macOS/arm64 ORT 1.29 与原生 Linux/arm64 ORT 1.28
    拒绝三类损坏模型和 image/text 各四类错误输入，错误后正常推理恢复。有效恶意图、生产 C/Go adapter
    的 admission/取消、统一 runtime 和 native amd64 仍缺，主任务保持未勾选。
  - 2026-08-27：进一步用 parser-valid ONNX fixture 验证同目录 external-data 控制图可运行，而父目录
    external-data、未知算子和循环图在两套 arm64 runtime 均失败关闭。仍未覆盖 oversized allocation、
    native amd64 和生产 adapter；并且产品包边界不得依赖 runtime，主任务保持未勾选。
  - 2026-08-27：6 GB 固定输出 hostile graph 在 Linux/arm64 三个有效地址空间档均返回
    `RuntimeException`，控制图正常且无 hang/crash；过低的 512 MiB 档因控制失败不计证据。生产单进程
    不能把 child RLIMIT 当隔离，仍须 exact hash/固定 shape/发行校验及 native amd64 adapter 证据。
  - 2026-08-27：官方 ORT 1.28.0 C 包与独立 Go/cgo harness 在 native arm64 通过 100 轮取消/恢复、
    RSS 门槛和 30 轮 race build；同时证明 raw runtime error 会带模型路径，生产只能映射稳定错误码。
    仍缺 amd64、最终镜像 ABI/扫描及生产 context/admission owner，主任务保持未勾选。
  - 2026-08-27：隔离 distroless arm64 final-stage 首次因遗漏 `libonnxruntime.so.1` SONAME 链接以 127
    失败，补齐后在 non-root/read-only/no-network/cap-drop 下通过 100 轮；进一步用 `base-nossl` +
    Debian-tracked C++ 最小闭包移除未使用 OpenSSL，镜像约 25.8 MB。Grype 仍有 glibc 1 Critical/2 High，
    Debian 将其列为 trixie minor/no-dsa 也不构成自动豁免；扫描器未识别 ORT package component 的
    缺口已由 arm64 显式 CycloneDX 组件补足，但尚未并入最终镜像。仍缺安全 VEX/修复基座、native
    amd64、生产 composition 与 signed provenance，主任务保持未勾选。
  - 2026-08-28：隔离 no-SSL distroless arm64 镜像已把 SentencePiece 0.2.1 与 ORT 1.28 合并为同一
    动态闭包；模型仍由只读 bind mount 提供。non-root/read-only/no-network/cap-drop 下 31 条 tokenizer→
    text encoder 链路通过，最大绝对误差 `1.811981201171875e-05`，并记录 SentencePiece archive/library/
    license/tag commit 与 incomplete CycloneDX component。该镜像继承尚未处置的 glibc 1 Critical/2 High，
    后续固定 Grype 0.116.1/关闭 DB 自动更新的 exact-image 扫描确认相同结果且无 fixed version；native
    amd64、exact distribution archive provenance、最终签名 SBOM/provenance、生产 composition 与
    ADR-0014 接受仍缺，主任务保持未勾选。
  - 2026-08-28：固定源码归档已逐文件对照官方 `v0.2.1` commit 的完整 GitHub tree：258/258 文件、
    258 个 raw Git blob 与 7 个 executable mode 全部一致。初次出现的 3 个 CRLF 差异已确认是本机
    `git hash-object` clean filter 造成的误报，离线 verifier 直接构造 Git blob hash 并避免该问题。它关闭
    “归档内容是否对应官方提交”，但不替代 exact download
    URL、上游签名或构建 provenance，`INT-008` 仍保持未勾选。
  - 2026-08-28：组合镜像构建已从 SentencePiece 的 117-step install graph 收窄到 42-step inference
    shared-library target，不再构建 training library、static archive 或五个 CLI。重建后的受限容器仍通过
    31 条 text parity，runtime `.so` hash 不变。Dockerfile 参数化并改用跨架构通用标签后的默认 arm64
    回归构建成功，当前 runnable manifest 为 `sha256:dedc1b601040f45682b46e86469d5d1d042bf7b322cee10e27f8cee4907d9447`。
    这只减少 builder scope，不关闭漏洞、amd64 或 provenance 门槛。
  - 2026-08-28：官方 SentencePiece URL 的 curl 与 BuildKit remote ADD 均在约 300 秒窗口未完成，因此
    不能作为受限网络下的唯一构建输入。组合 Dockerfile 已改为 SentencePiece/ORT 两个本地 reviewed
    archive context，并在解包前分别强制 SHA-256；完全离线重建成功。Dockerfile 内容不变时，先前重复构建
    保持相同 runnable manifest/config，而 BuildKit 重新生成的 provenance 会改变外层 local index；参数化及
    通用镜像标签则按预期改变 config/manifest。发布证据必须签 exact published index，内容复现对照 platform
    manifest。当前默认 arm64 config 为 `5891f7a4…`，local index 为 `7fed8b18…`。
  - 2026-08-28：再次以新临时文件从官方 tag URL 完整下载，GitHub 正常重定向到 codeload，但 300 秒后仅
    收到 10,155,379/13,485,527 bytes；服务端不支持 byte-range resume，已停止自动全量重试。该结果只证明
    当前网络条件不适合作为 release 输入，不能补写 exact distribution archive provenance；源码 258/258
    blob 等价证据仍有效，发布来源阻断保持开放。
  - 2026-08-28：同一参数化 Dockerfile 已用固定 ORT x64 archive、amd64 distroless child manifests 和
    `/usr/lib/x86_64-linux-gnu` 在 QEMU 构建并运行。non-root/read-only/no-network profile 下 31 条 text
    parity 通过，最大绝对误差 `2.09808349609375e-05`；这关闭 amd64 package/SONAME/numeric preflight，
    但 QEMU 不能替代 native linux/amd64 的 timing/RSS/cancel/soak/scan，`INT-008` 继续未勾选。
- [ ] `INT-009` 实现 SQLite blob 精确向量基线并跑 10k/50k/100k 查询、过滤和稳定排序。
  - Owner：semantic/store；仅限 spike。
  - 2026-08-25：已完成 macOS arm64 开发基线、稳定 tie-break、未索引过滤基线和 float16/int8 随机向量比较；Linux 双架构、真实 embedding recall、并发和恢复未完成，保持未勾选。
  - 2026-08-26：SQLite/WAL 新 generation 与 active pointer 在未提交事务内完成后真实强杀；macOS/arm64
    和原生 Linux/arm64 重启只保留完整旧代，`integrity_check=ok`，随后可重新构建并原子切换。真实
    embedding、100k 联合负载/浏览并发、amd64 与最终空间预算仍缺，保持未勾选。
  - 2026-08-26：原生 Linux/arm64 受限容器三次联合随机向量 backfill/search/keyset-browse/cancel/restart；
    search P95 229.843～258.371 ms，恢复正确，但 browse 相对退化 16.25～23.22 倍，未通过 20% Gate。
    DB 410,619,904 bytes 未含视频/人脸/WAL/备份，拒绝 float32 作为 combined final layout。真实 embedding
    float16/降维质量、生产 browse path 和 amd64 仍缺，主任务保持未勾选。
  - 2026-08-26：同一受限矩阵的 float16 SQLite 文件为 136,880,128 bytes，search P95
    138.854～156.912 ms，取消/恢复正确；容量与 I/O 性能明显优于 float32，但 browse 相对 Gate 仍失败，
    且随机向量不能证明真实模型 Recall。float16 只进入下一轮候选，不视为选型完成。
  - 2026-08-26：SigLIP 2 ONNX 在生产 govips 生成的公开许可 10 图/24 查询 pilot 上，macOS 与原生
    Linux arm64 的 float32/float16 Top-3 记录一致，float16 未损失当前指标。该 pilot 不代表 1,000 图
    困难集；且当前单体 ONNX 每次同时执行 image/text encoder，必须先做 split graph 对照，保持未勾选。
- [ ] `INT-010` 至少比较一种可重建 ANN 方案的 build、查询、recall、RSS、损坏恢复和双架构闭包。
  - Owner：semantic/release；依赖：INT-009。
  - 2026-08-25：`coder/hnsw` 已完成 10k 三档 recall/build/query、截断拒绝和重建；因低档 recall 失败、高档构建昂贵、import 分配边界和非确定导出，当前拒绝。native 双架构闭包仍未完成，保持未勾选。
- [ ] `INT-011` 验证视频复用 4/10 格故事板的质量、frame 聚合和无重复 FFmpeg admission。
  - Owner：semantic/media；依赖：INT-006。
  - 2026-08-25：用静态 embedding 组合的 4 帧代理已比较 mean pooling 与 max-frame；前者在极小合成集
    更稳定。真实 4/10 格视频、frame sampling 和复用既有 FFmpeg admission 尚未执行，保持未勾选。
  - 2026-08-26：原生 Linux/arm64 已用项目固定 FFmpeg 7.1.5 经现有生产 adapter 跑通合成 H.264 的
    4/10 帧计划，统一结果验证通过且源 hash 不变。它关闭“是否能复用既有 adapter”的技术子问题；
    100 个合法代表性真实视频、embedding/聚合质量、amd64 和联合负载仍缺，保持未勾选。
- [ ] `INT-012` 验证核心簇/边缘建议分层，以及 manual assignment/cannot-link 重聚类不被覆盖。
  - Owner：face；依赖：INT-007。
  - 2026-08-26：隔离合成状态机已证明 core 批量确认、edge 单项确认、anonymous merge、single-face
    assignment、cannot-link 和 model-generation rollover 可保持人工关系优先；还主动拒绝了 edge 随组
    自动写入人物库的危险语义。真实模型 core precision、合法 ground truth、transaction/concurrency、
    删除/恢复与 API 授权仍缺，主任务保持未勾选。
- [ ] `INT-013` 运行 4 CPU/4 GiB 100k 联合容量：backfill + browse/search + restart + cancel。
  - Owner：performance；依赖：INT-008～012。
  - 2026-08-27：三次 arm64 隔离组合负载未 OOM，真实 split image encoder 输出正常，synthetic float16
    exact search 与取消/恢复通过，容器峰值低于 3.2 GiB；但 keyset proxy 的相对 browse Gate 三次均
    失败，且尚非生产 HTTP/catalog、真实 100k 图片 backfill、人脸 runtime 或 amd64，主任务保持未勾选。
  - 2026-08-27：较小 SigLIP 1 双 session 常驻与 synthetic 100k float16 三次并发，cgroup peak
    2.364～2.370 GB、search P95 162.977～171.009 ms、恢复完整；但 browse 相对退化 4.76～8.70 倍，
    且仍缺生产全进程、真实 backfill、人脸及 amd64，主任务保持未勾选。
  - 2026-08-27：复用生产 scanner/catalog/SQLite/storyboard admission 的 10k 目录/100k 文件容量档，
    ordinary recursive browse P95 从 28.267 到 31.431 ms（+11.2%），但整容器 peak 从 1.604 到
    3.590 GB，超过 3.2 GiB；global/storyboard browse 为 +25.3%/+20.3%。两轮 search-keyset 均未过
    既有 250 ms 预算，且只完成一组配对，仍不能关闭 Gate，主任务保持未勾选。
  - 2026-08-27：float16-internal 同路径 peak 2.906 GB、recursive browse +14.5%、scan/read/search
    +4.7%/+6.8%/+1.8%，但 global search +41.6%，baseline/AI keyset 都失败且缺重复配对、真实 embedding、
    HTTP/browser/face 与 amd64，主任务保持未勾选。
  - 2026-08-27：三组配对的 AI peak 均低于 3.2 GiB，recursive browse 中位退化 +10.4%；但一轮
    storyboard browse 超限，全部 baseline/AI keyset 自身不达预算，且真实 embedding/HTTP/browser/
    face/amd64 仍缺，主任务保持未勾选。
  - 2026-08-27：额外无 AI 基线把 keyset 成本拆为每页一次 count 与一次 list；count P95 66.987 ms，
    list 第一/二页 106.750/130.316 ms，服务两页合计 358.623 ms。它排除了“只是两页合并口径”的解释，
    也表明只消除重复 count 不足以直接证明达标；后续 query-plan 见下一条，不能借 AI Gate修改既有
    预算或静默改生产搜索合同。
  - 2026-08-27：benchmark-only `EXPLAIN QUERY PLAN` 已证明 count 扫描 FTS 候选后按 rowid 回表，
    first/second list 同样以 FTS 驱动并出现 `USE TEMP B-TREE FOR ORDER BY`；第二页 keyset 没改变 plan，
    `assets_browse_folder_name_v2` 也没有产生搜索结果顺序。因此“再加一个普通 browse index”不足以解决，
    后续必须在独立维护 slice 比较 index-ordered membership、bounded/materialized candidate 与 count
    revision/首屏语义；AI Gate 不授权生产查询、schema、API 或预算变化。
  - 2026-08-27：后续 order-first 矩阵扩到 26 个首页面/22 个第二页组合，全部保持生产 ID 顺序；
    arm64 100k 候选单页最坏约 203 ms。三次生产 broad + modified-window 首/次页为
    9.44～19.00/9.55～19.00 s；plan 证明 `assets_modified` 驱动范围扫描、逐行 FTS 探测和临时排序。
    80k image/10k video/10k animated 及 image+video 组合已覆盖；其他组合 kind、真实选择性、重复分布、
    完整 hydration、native amd64 和选择阈值仍缺，
    所以只收窄 maintenance 候选，不授权生产修复或勾选主任务。
  - 2026-08-27：另以两个不重叠库、各 5k 目录/50k 媒体补 global library-ID tie-break；再加入不同
    image/video/animated 比例、image+video、选择性日期和约 2% 稀疏词后，11 个首页/11 个第二页全部
    保持顺序，候选最坏约 83 ms。它关闭合成跨库 mixed-media/date/sparse 正确性子证明，不能替代
    真实分布/选择性、重复 P95、full hydration 或 amd64。
  - 2026-08-27：当前最慢 kind、递归名称排序和 broad modified-window 三个代表场景各采样 20 次，
    首页/第二页 P95 最坏 174.591/135.115 ms，均低于 250 ms；这只关闭代表场景的重复分布缺口，
    不代表其余矩阵、跨库分布或完整 hydration 的 P95。
- [ ] `INT-014` 完成 runtime、模型权重、vector 引擎 SBOM/许可证/漏洞与再分发审查。
  - Owner：合规/安全；依赖：候选收敛。
  - 2026-08-25：完成官方来源初筛；Chinese-CLIP 许可不清、InsightFace 公共权重非商用，均未获批准。
  - 2026-08-27：固定 ORT 1.28.0 arm64 官方 archive、MIT license、6,121 行 third-party notices 与
    各自 hash；SigLIP 1/YuNet/SFace 的精确 revision/hash/目录许可记录已复核。Docker Scout 因本机
    未登录而在分析前停止；固定 Grype 0.116.1 后续虽返回 0 match，却也只识别到 0 component，仍不能
    算漏洞扫描通过。最终镜像 SBOM/双架构扫描/notice 打包仍缺。SFace
    虽由目录 README 声明 Apache-2.0，但精确权重训练数据来源与商业/再分发澄清仍无已采信答复，
    因此 production hold，主任务保持未勾选。
  - 2026-08-27：arm64 distroless `cc` 最终闭包 SBOM 识别 1,284 个 component，扫描仍有 1 Critical/
    9 High；`base-nossl` 对照删去未使用 OpenSSL 后为 1,264 个 component、1 Critical/2 High。剩余三项
    均来自 glibc，Debian tracker 标为 vulnerable 且 minor/no-dsa；必须由安全 owner 出具 VEX/可达性
    结论或更换已修复基座，不能自动忽略。扫描器漏识别的 ORT `.so` 已补通过官方 Schema 结构校验的
    arm64 显式 component，但尚未并入最终 SBOM；amd64/生产镜像/签名 provenance/notice bundle/
    正式合规签署仍缺，主任务保持未勾选。
  - 2026-08-27：从 no-SSL 镜像提取实际 arm64 harness/ORT ELF，52/386 个 undefined symbol 中均无
    受影响 scanf family、`ungetwc`、`ns_printrrf`、`ns_printrr`、`fp_nquery` 直接 import，精确字符串也
    无命中；但无法排除 glibc 内部间接路径、运行时查找或未来生产代码，故仅作为 VEX 评审输入，
    不勾选 `INT-014`。2026-08-28 扩展到完整外部 ELF 闭包后已在两架构 libstdc++ 发现 `ungetwc`
    直接导入，因此旧结果只适用于 harness/ORT 两文件，不能作为 closure-wide VEX 依据。
  - 2026-08-28：Docker Scout 1.24.0 对 exact arm64 text-runtime 两次均识别 17 个 package，却漏掉手工
    复制的 ORT/SentencePiece，并在两次输出间把相同文件随机挂到不同 Debian package。隔离合并器改为
    扁平化该不稳定归属、保留依赖图并加入两个显式 component；两次规范化结果 byte-identical，包含
    1,267 components/18 dependency rows，且无 dangling ref。结果继续标记 incomplete；尚未覆盖模型权重、
    amd64、漏洞/VEX、notice bundle、签名 provenance 或生产镜像，主任务保持未勾选。
  - 2026-08-28：固定 Grype 0.116.1 与有效 2026-08-26 DB、关闭自动更新后扫描 exact arm64 及 QEMU
    amd64 text-runtime；两者均为 15 findings，Critical/High 精确为同三个 `libc6` CVE 且无 fixed version。
    ORT/SentencePiece 仍未被识别为 package，需显式 advisory/VEX correlation；扫描结果直接维持发布 No-Go。
  - 2026-08-28：将直接导入检查扩展到 app/ORT/SentencePiece/libstdc++/libgcc_s，arm64/amd64 分别覆盖
    575/573 个唯一 undefined symbol。scanf 与 DNS debug 三函数无直接导入，但两架构 libstdc++ 的
    `stdio_sync_filebuf<wchar_t>` 均调用 `ungetwc`；`CVE-2026-5928` 不具备“无直接导入”前提，保持 No-Go。
- [ ] `INT-015` 完成人脸数据告知、访问、清除、备份、诊断和共享场景隐私评审。
  - Owner：安全/隐私。
  - 2026-08-27：已形成[隐私评审草案](../gates/POST-MVP-5/int-015-face-privacy-review.md)，明确 face
    embedding/匿名组仍按高敏感/假名化数据处理、默认关闭、本地无网络、不落盘 crop、访问/诊断/
    分享边界、五种清除动作、备份敏感性和真实 ground-truth 治理；同时列出中国现行官方规则作为
    部署者评估输入。跨库人物、禁用/清除、备份和合法依据仍须 privacy/legal owner 签署，保持未勾选。
- [x] `INT-016` 根据证据冻结 revision 1 A+B 的 ORT C API + SigLIP 1 float16-internal + SQLite
  float16 exact + CPU-only；拒绝 SigLIP 2 当前 layout、float32、dynamic-QInt8 和 HNSW；E 移出首版。
  - Owner：技术负责人；依赖：INT-006～015。
  - 2026-08-27：`ADR-0013` 已增加 S0 候选处置表，明确拒绝 SigLIP 2 当前 layout、SigLIP 1
    float32/dynamic-QInt8、SQLite float32、当前 HNSW、内置权重和未运营镜像；只保留 ORT C API +
    SigLIP 1 float16-internal + SQLite float16 exact + `/models:ro`/managed/direct 作为 A+B 下一 Gate
    组合。SFace production hold，Slice E 已移出 revision 1。真实数据/native amd64/security/compliance
    进入 A+B 的 Backend/Release Gate，不阻止本架构决策完成。
- [x] `INT-017` 为所有开放风险指定 owner 角色、最迟决策 Gate、后续复验证据和强制 fallback，更新
  R-024～R-030；风险本身继续开放，Frozen scope 仍须把角色落实到可执行负责人。
- [x] `INT-018` 创建并批准 `POST-MVP-5` scope revision 1；A+B 上限 16 单人工程周，版本总上限
  32 周，C/D/E 不在 revision 1。
  - Owner：产品负责人；依赖：INT-001、INT-016。
  - 2026-08-27：proposal draft 1 已把模型基础、图片语义、受控标签、视频代表帧、人脸人物库拆为
    可独立删减切片；产品用户随后接受只冻结 A+B、16/32 工程周停损和 C/D/E 后续独立 revision，
    [scope revision 1](../releases/POST-MVP-5-scope.md) 已落地。
- [x] `INT-019` 冻结 revision 1 模型分发策略：不内置、不提供项目在线源/国内镜像，仅承诺
  `/models:ro` 离线目录、托管复制和严格直接读取；未来在线源必须新增 scope revision。
  - Owner：产品 + release + compliance；依赖：INT-014。
  - 2026-08-27：决策草案已拒绝应用镜像内置、任意 URL 和未运营“国内镜像”，要求 `/models:ro`
    作为离线基线；项目签名源/部署者镜像只有在真实 owner 与 key/checkpoint/revocation/SLA 到位时纳入。
    当前没有在线 source owner/endpoint，因此 revision 1 正式采用只保留离线安装的 fallback。
- [ ] `INT-020` 隔离验证下载取消/续传/重定向/错哈希/磁盘满，以及 `/models:ro` 扫描、托管复制、
  直接读取、源变化/缺失/可写/symlink/嵌套 mount，以及 external-data graph 的失败关闭
  和恢复。
  - Owner：aimodel + security + release；仅限 spike。
  - 2026-08-25：Linux scanner 已实现单文件及 multi-file schema v2 的 catalog/status/size/hash、整包
    exact-content 与 `openat2` fail-closed 骨架；SFace 下载中断后断点续传和最终 hash 已验证。macOS
    no-replace 原子目录发布通过，Linux 双架构仅交叉编译，完整原生 Linux/下载故障矩阵未执行。
  - 2026-08-26：隔离 `httptest` 矩阵验证 catalog-owned URL、ETag/Range 稳定及中途取消后续传、
    精确同源重定向、loopback 默认拒绝、配额、错 hash 和 no-replace 发布；macOS/arm64 连续复测与
    原生 Linux/arm64 通过；128 KiB tmpfs 验证真实 `ENOSPC` 时 candidate 不可见、active 不变；真实
    子进程 `SIGKILL` 后由新进程以同一 ETag/Range 恢复并精确发布。当时尚缺 DNS、发布边界、签名、
    active generation 与原生 Linux/amd64；其中部分已由下列后续证据补齐，主任务仍保持未勾选。
  - 2026-08-26：catalog transport 增加单次 DNS 解析、特殊/私网/混合 answer 整体拒绝、固定 IP 拨号、
    TLS server name 保留和禁用环境 proxy；macOS/arm64 连续复测及原生 Linux/arm64 通过。真实
    catalog TLS/CDN 轮换、外层 retry/backoff 策略和原生 amd64 仍未完成，主任务保持未勾选。
  - 2026-08-26：Ed25519 签名 catalog primitive 覆盖 domain separation、key ID、序号、时间窗和精确
    payload，严格拒绝未知/错误 key、篡改、回滚、同序号换内容、非 JSON/超限/多余字段；两套 arm64
    环境通过。真实 key 托管/轮换/撤回、durable checkpoint 和原生 amd64 仍未完成，主任务保持未勾选。
  - 2026-08-26：真实子进程在 package no-replace rename 前、rename 后但 parent fsync 前、fsync 后三个
    边界强杀；macOS/arm64 与原生 Linux/arm64 重启均只观察到完整 staging 或完整 final。进程崩溃原子性
    已有证据，但不代表 host power loss durability；amd64 仍缺，主任务保持未勾选。
  - 2026-08-26：原生 Linux/arm64 在完整 staging 与 active sentinel 落盘后，将 128 KiB tmpfs 写至内核
    `ENOSPC`；同文件系统 no-replace rename 与 parent fsync 仍成功，final 内容完整且 active 未变。该结果
    符合 rename 不复制模型字节的语义；host power loss 与 amd64 仍缺，主任务保持未勾选。
- [x] `INT-021` 冻结 `/models` 与 `/app/data/models` 的所有权、备份/恢复、升级、配额、诊断和
  provenance 语义；明确直接模式在模型 unavailable 时的查询行为。
  - 2026-08-27：决策草案已指定 aimodel 为唯一 owner，managed 默认、direct 显式高级选项；direct
    消失/变更/可写即 unavailable，普通浏览继续、依赖 query 返回稳定错误且不自动换代；备份、升级、
    清理和诊断边界已列出。selected package quota、生产合同和 native amd64 进入 S1/Backend/Release
    Gate，不阻止 S0 所有权决定完成。
  - 2026-08-25（历史状态）：spike 已证明 package generation 可由同目录 staging 以 no-replace
    rename 原子发布；当时 active generation、签名 catalog、回滚和数据库事务 owner 尚未冻结，
    后续已由 2026-08-27 的冻结决定收口。
  - 2026-08-26：隔离 SQLite registry 证明 checkpoint 与 active pointer 可在单事务内推进，注入 active
    更新失败会连同 checkpoint 回滚，旧 active 保留且重启后一致。它没有冻结生产 migration/owner、
    文件系统与 DB reconciliation、备份/恢复或 unavailable 语义，主任务保持未勾选。
  - 2026-08-26：原生 Linux/arm64 将完整 `openat2` scan report 接入 registry；精确文件系统 orphan
    只登记不激活，缺失/损坏仅标 unavailable 并保留 active/checkpoint，精确恢复后重新 available，
    未知目录不删除。managed/direct 全生命周期、生产 owner、备份/配额和 amd64 仍缺，保持未勾选。
  - 2026-08-26：原生 Linux/arm64 真实只读 bind mount 验证 direct source kind 不可变、挂载消失只标
    unavailable、同一精确包 remount 后恢复；不复制/删除/自动切 active。测试容器的临时 mount capability
    不进入产品部署要求；生产 direct 配置/异常、API/UI、备份配额和 amd64 仍缺，保持未勾选。
- [x] `INT-022` 用 INT-019～021 证据复审并接受 revision 1 A+B 的 ADR-0013；未来在线源、人脸或
  部署拓扑变化必须重新决策。
  - 2026-08-27：ADR 接受范围只覆盖 A+B 的离线模型路径；online source、native amd64、最终 VEX/SBOM
    仍由后续 Gate 持有，SFace 与 E 不在 revision 1。
- [x] `INT-023` 复审 INT-S0；仅对 Frozen revision 1 的 A+B 为 Go，授权 S1 合同设计，不授权生产实现。
- [x] `INT-023A` 2026-08-27 完成本地探索收口：冻结已有证据，停止扩张同类合成测试，将真实质量、
  native amd64、最终镜像/供应链和生产纵向验证归回对应切片 Gate；五项产品决定均已接受，外部
  条件已登记到后续 Gate。证据：[INT-S0 收口与阻塞清单](../gates/POST-MVP-5/int-s0-closeout-and-blockers.md)。

## S1：A+B 权威合同（完成）

- [x] `INT-101` 更新产品需求、用户流程、UI 设计和 README，写入冻结的 A+B FR/NFR/非目标。
- [x] `INT-102` 接受 aimodel/semantic/capability-owned inference/jobs 接口、依赖方向和错误所有者；
  face 不在 revision 1。
- [x] `INT-103` 更新数据模型，冻结派生/应用状态、FK、revision、retention、backup 和 cascade。
- [x] `INT-104` 冻结只追加 migration 计划以及 fresh/upgrade/idempotent/failure/restore 验收矩阵；实际
  `00022` migration 与自动化验证属于 S2，不在 S1 提前创建。
- [x] `INT-105` 更新安全模型：模型供应链、日志脱敏、清除和无网络边界；明确不创建生物特征合同。
- [x] `INT-106` 更新部署文档：非内置模型、模型位置、磁盘预算、双架构和备份/恢复。
- [x] `INT-107` 更新测试策略与 fixture 治理，冻结准确率、性能和跨架构误差判定。
- [x] `INT-108` 在 `api/openapi.yaml` 只定义 A+B 的 AI model/settings/status、semantic image search、
  rebuild/clear，并统一错误、幂等与 ETag；C/D/E endpoint 不进入 revision 1。
- [x] `INT-109` 生成 TypeScript 客户端并运行 OpenAPI 结构、操作集、摘要锁和确定性生成合同。
- [x] `INT-110` 编写 A+B API/data transaction decision table：partial failure、并发修改、取消、清除和
  generation rollover；stale face 不属于 revision 1。
- [x] `INT-111` 冻结模型列表、operation cancel、fixed-directory scan、managed/direct、activate/status
  合同；所有请求只接受 opaque ID，不接受 URL 或路径；revision 1 明确没有 download endpoint。
- [x] `INT-112` 冻结 `/models:ro` 包格式、扫描上限、临时空间、8 GiB managed quota、安全余量和
  Compose 示例；下载源、镜像、重定向与代理全部不在 revision 1。
- [x] `INT-113` 复审 [INT-S1 Contract Ready](../gates/POST-MVP-5/int-s1-contract-ready.md)：仅 A+B
  为 Go，授权 S2 后端，不授权 C/D/E、在线模型源或生产 UI。

## S1R2：C+D+E 权威合同扩展（当前阶段）

- [x] `INT-114` 冻结受控标签词表、suggestion/review 生命周期、人工标签优先级、失效与删除语义。
- [x] `INT-115` 冻结视频 4/10 帧输入、聚合/命中 DTO、部分失败、generation/source 失效和质量门槛。
- [x] `INT-116` 冻结 face observation、匿名 core/edge、person、assignment、exclusion、cannot-link、
  merge/split/undo 状态机及默认关闭/用户命名/禁止身份推断合同。
- [x] `INT-117` 接受 C/D/E 数据模型、事务 owner、migration、cascade、retention、清除、备份、恢复和
  generation 升级合同；区分可重建敏感派生数据与不可重建人物应用状态。
- [x] `INT-118` 接受 C/D/E OpenAPI：opaque ID、cursor、ETag/idempotency、operation、错误、批量上限、
  auth/CSRF/rate-limit 与不返回向量/crop/路径/模型原始错误的 DTO。
- [x] `INT-119` 接受安全/隐私、部署、测试与质量合同：合法数据 intake、core precision ≥99.5%、tag/video
  门槛、native 双架构、4 CPU/4 GiB/100k 联合负载、SBOM/VEX/notices/provenance 及失败回退。
- [x] `INT-120` 更新 traceability/风险/清单并复审 C+D+E S1R2 Contract Ready；未 Go 前不得创建
  production migration、handler、worker、`internal/face` 或消费者 UI。

## S2A：模型管理与图片语义搜索后端

- [x] `INT-201` 实现 manifest/hash/license/architecture 校验和 model unavailable 状态。
  - 2026-08-27：新增 `internal/aimodel` 的严格 JSON/重复 key/未知字段/文件角色/大小/hash/许可/
    runtime architecture 内建 catalog 比对；新增 availability revision 与 SQLite compare-and-swap，
    wrong hash/unknown field/wrong architecture/stale revision 均失败关闭。
  - 2026-09-01：availability CAS 在 wall-clock 回拨时把更新时间钳制到模型创建时间；语义库 settings 的
    首次插入与后续 revision 更新同样不再早于 library/settings 创建时间。两条旁路回归各连续 20 次通过，
    记录见 [S2 durable job 时钟回拨收敛](../changes/FIX-2026-09-01-s2-job-clock-rollback.md)。
- [x] `INT-202` 实现 inference port 与已选 adapter，固定线程、超时、取消、延迟加载和资源计数。
  - 2026-09-01：模型安装/激活 claim 与通用 AI operation transition/recovery 现在逐行钳制 wall-clock
    回拨，claim 返回持久化后的有效时间；三条明确回拨回归各连续 20 次通过。该可靠性修复不替代最终
    runtime 图、RSS、parity 或 native amd64，`INT-202` 状态不变。记录见
    [S2 durable job 时钟回拨收敛](../changes/FIX-2026-09-01-s2-job-clock-rollback.md)。
  - 2026-08-27：已定义 production-facing `InferenceRuntime.LoadAndValidate`、受控 package opener 和
    validated embedding-dimension 回传合同；activation worker 已执行 availability revision、reviewed
    manifest、source revalidation 和协作取消。
  - 2026-08-27：新增显式 `onnxruntime` build tag 的 ORT 1.28.0 C API adapter；固定 intra-op=2、
    inter-op=1、关闭 CPU arena，并严格校验 split graph 的单输入/单输出名称、float32/int64 类型及
    `[1,3,224,224]→[1,768]`、`[1,64]→[1,768]` 形状。默认/非 Linux 构建以稳定 unavailable
    失败关闭；已用固定 arm64 ORT library/header 完成真实 cgo 编译。ORT `CreateSession` 自身没有
    可中断句柄，当前只在每次 session load 前后响应取消；尚缺执行期 RunOptions 取消、hard timeout、
    resident session/resource accounting 与 native amd64，主任务保持未勾选。
  - 2026-08-28：production adapter 新增 image-only session 与真实 ORT tensor `Run`：严格接受
    `[1,3,224,224]` float32、读取 `[1,768]` 输出，session 持有已安全打开的 model FD，Close 串行等待
    当前执行并释放 value/run options/session/environment/file。每次 Run 使用独立 RunOptions，context
    取消调用 `RunOptionsSetTerminate`，返回只保留稳定错误而不泄露 ORT message；默认/non-Linux build
    的 `OpenImageSession` 继续 unavailable 失败关闭。已在固定官方 ORT 1.28.0 Linux/arm64 header/library
    容器完成 production package native cgo 编译。尚无可提交的审核 ONNX fixture，故本轮没有声称真实
    graph 输出或取消运行通过；generation-bound single-resident manager、hard timeout/idle unload、真实
    SigLIP parity 与 native amd64 仍缺，`INT-202` 保持未勾选。
  - 2026-08-28：新增 application-owned generation session owner：每次加载先从 SQLite 复核 generation
    仍为 active、dimension=768，每次 Run 前经 aimodel owner 复核 model available 与审核 catalog manifest，
    冷加载时再复核模型来源，
    只通过 ActivationPackageSource 打开 image graph。全生命周期最多一个 resident session；切代先关闭旧
    session，单次 load/Run hard timeout 30 秒，非取消运行错误丢弃 faulted session，下次延迟重载，空闲
    5 分钟自动卸载，应用停止时同步 Close。该 owner 已接入全局 background=1 的 production semantic
    worker。仍缺真实审核 graph Run/cancel/parity、RSS 计量与 amd64，故本项仍不勾选。
  - 2026-08-28：同一 lifecycle owner 新增可并发读取的资源计数：当前 resident session、active Run 与
    累计 load/Run/unload。阻塞 Run 的 race test 证明运行中为 `resident=1/active=1`，完成、切代、故障
    丢弃和 Close 后计数精确收敛；计数不包含 generation/model ID、路径、query、vector 或 runtime 原始
    错误。实现侧的 resident-session accounting 已关闭；最终审核 graph 的 RSS/parity 与原生 amd64
    证据仍缺，因此 `INT-202` 保持未勾选。记录：
    [semantic session resource accounting](../changes/FIX-2026-08-28-semantic-session-resource-accounting.md)。
  - 2026-08-28：全仓 `make test-race` 已通过，覆盖 production app composition、aimodel activation/install、
    semantic queue/search/clear、SQLite durable 状态和 integration/performance；未发现 Go data race。
    这补充并发实现证据，但不替代 native ORT/RSS 或原生 amd64 Gate，`INT-202` 状态不变。
  - 2026-09-01：production ORT adapter 已补齐 text session、每次 Run 的 terminate cancellation、固定
    `[1,64] int64 → [1,768] float32` ABI 和有限输出检查；application owner 以单 resident generation、
    30 秒 hard timeout、5 分钟 idle unload、切代/故障关闭及确定性资源边界组合 image/text runtime。
    官方 ORT 1.28.0 Linux/arm64 archive 与 SentencePiece 联合 tagged production 包已原生编译链接通过。
    任务要求的实现边界完成；最终模型 RSS/parity 与原生 amd64 属 `INT-209/210/215` 证据，不再重复阻塞本项。
- [x] `INT-203` 实现 semantic model generation、preprocess 和 deterministic embedding contract。
  - 2026-08-27：新增 `internal/semantic` 权威 embedding codec：有限/非零/固定维度校验，float64
    累加后 L2 normalize、IEEE-754 binary16 little-endian 持久化，以及解码后再次归一化；畸形长度、
    NaN/Inf、零向量均失败关闭。当前候选文本侧实际依赖 SentencePiece，但提议 ADR/包合同尚未接受
    tokenizer runtime，且三文件 manifest 只有单个 `tokenizer.json` role；不得手写不完整 tokenizer 或
    未经评审引入第二 native 闭包。完整 tokenizer、一致性 fixture 和真实 inference 尚缺，
    主任务保持未勾选。
  - 2026-08-27：真实固定 revision `tokenizer.json` 复核确认含 Unigram、byte fallback、Metaspace 与
    precompiled Unicode charmap，拒绝项目内近似重写；已按官方 SigLIP slow tokenizer 落地纯 Go
    Unicode lowercase/ASCII punctuation removal/Unicode whitespace collapse，并用官方 Python reference
    核对中英文、全角、组合字符、emoji、空白与截断 ID。新增提议 ADR-0014，推荐官方 SentencePiece
    C++ + `spiece.model` + package format v2；ADR/供应链/双架构 fixture 未接受前不进入 production adapter。
  - 2026-08-27：隔离 `spikes/int001-sentencepiece-capi` 已在原生 Linux/arm64 以官方 SentencePiece
    0.2.1 和固定 `spiece.model` 跑通真实 Go/cgo 链接；中英文、全角、组合字符、emoji 固定 token ID，
    63-piece 截断与 64 位 EOS/pad 均通过。该证据只关闭 arm64 C API 技术可行性，不批准 ADR-0014，
    也不替代 native amd64、FD 加载、malformed/resource/SBOM、图片 preprocess 和 embedding parity；
    `INT-203` 保持未勾选。
  - 2026-08-28：同一隔离 spike 的 Linux/arm64 生命周期证据已补齐 open-FD 同步加载、32 并发调用、
    幂等 close/use-after-close、empty/truncated/oversized/non-regular model 拒绝、100 次预取消与 100 次
    load/close。显式 Go 内存释放后 RSS 从 26,030,080 增至 33,632,256 bytes，净增 7,602,176 bytes，
    低于 64 MiB spike 门槛。SentencePiece 没有 mid-call interrupt，因此只证明 native 进入前取消和返回后
    复核；native amd64、长时间 soak、完整 Python fixture、text ONNX embedding parity、format v2、最终
    SBOM/provenance 及 ADR 接受仍缺，`INT-203` 继续未勾选。
  - 2026-08-28：相同 tagged suite 在 QEMU 模拟 Linux/amd64 userspace/ABI 编译并通过，100 次
    load/close 的 retained RSS 增量为 7,655,424 bytes。该结果只排除明显的 amd64 编译、链接与行为差异；
    QEMU 不能证明原生 amd64 时序、内存、CPU 行为或长期稳定性，因此 native amd64 gate 不关闭，
    `INT-203` 状态不变。
  - 2026-08-28：新增可确定性重建的 Transformers 4.56.2 + SentencePiece 0.2.1 reference fixture，
    绑定固定 model revision/size/SHA-256，31 组覆盖多语种、全角/组合字符、Unicode whitespace、
    emoji/ZWJ、稀有字符、控制字符与 512-rune 截断，并逐项比对 canonical text 和全部 64 IDs。首次运行
    真实发现 Greek final sigma 漂移：Transformers 的 non-greedy regex 是逐 code point lowercase，而
    原 Go owner 做 whole-string contextual lowercase；现已修复并由 Linux/arm64 全矩阵通过。literal
    registered special token 的拒绝/转义合同、native amd64、text embedding parity、format v2 与供应链
    仍缺，故 `INT-203` 继续未勾选。
  - 2026-08-28：已冻结 literal tokenizer control policy：canonical owner 在 lowercase 后、ASCII 标点
    删除前以大小写不敏感方式拒绝 `</s>` 与 `<unk>`，SearchService 保证错误发生在 snapshot/encoder/vector
    依赖前，HTTP 只返回无 query 回显的 `invalid_request`；OpenAPI、API/security/product spec 与 ADR 已同步。
    该项关闭 special-token 注入歧义，但 native amd64、text embedding parity、format v2、供应链与 ADR
    接受仍阻断 `INT-203`。
  - 2026-08-28：从已保留且 hash 匹配的 SigLIP 1 split text graph 生成 31×768 完整 float32 reference；
    graph 为 441,217,411 bytes / SHA-256 `16eef127...fd664`，fixture 为 133,700 bytes / SHA-256
    `943c0575...0225`。两份独立 byte-identical export 生成的 reference JSON 逐字节一致，且所有输出有限。
    这只关闭 reference 构建；Go/cgo SentencePiece → Linux ORT 1.28 C API 的逐项 `1e-4` parity、取消和
    lifecycle 尚未执行，故 `INT-203` 仍不勾选。
  - 2026-08-28：隔离 `linux+cgo+sentencepiece+onnxruntime` harness 已打通 canonical query →
    SentencePiece 64 IDs → ORT 1.28 text graph；31×768 共 23,808 个 float 均为有限值并通过
    `atol=1e-4, rtol=1e-4`，相对固定 ORT 1.29 reference 的 max abs 为
    `1.811981201171875e-05`。同时覆盖 hash 绑定、pre-cancel、幂等 close/use-after-close。尚缺 production
    FD owner 集成、active-run cancel、text session concurrency/load-close RSS、native amd64、format v2、
    供应链和 ADR 接受，故 `INT-203` 仍不勾选。
  - 2026-08-28：text harness 已观察进入 `OrtRun` 后取消并返回 `context.Canceled`，随后同一 session
    被 8 个 Go caller 并发复用且在 native handle 串行成功。10 次 warm-up 后再做 10 次 load/close，
    measured retained RSS 仅 +28,672 bytes，未见持续 per-cycle leak；但 cold 14,938,112 → stable
    370,835,456 bytes，实际扩张 355,897,344 bytes，必须计入完整进程容量，不能以斜率平坦忽略。
    尚缺 long soak、production FD owner、native amd64、format v2、供应链和 ADR 接受，`INT-203` 不勾选。
  - 2026-08-28：format v2 已从概念补成隔离可执行合同：严格三角色
    `image_encoder|text_encoder|sentencepiece_model`，以及 image preprocess、text canonicalization、
    tokenizer metadata/sequence、embedding storage 四个精确 contract ID。validator 证明 valid v2 通过，
    v1 number/v1 tokenizer role、unknown contract、nested path、duplicate role/key、unknown field 与 trailing
    JSON 均失败关闭。production parser 仍保持 v1-only；未获 ADR 接受前不迁移，`INT-203` 状态不变。
  - 2026-08-27：新增 `internal/semantic` 固定图片 tensor 合同：只接受恰好 224×224 的 interleaved
    uint8 RGB，以 reference 相同的 float64 rescale → float32 normalize 步骤输出 `[3,224,224]` CHW，
    并固定关键像素的 IEEE-754 bits，防止公式化简造成静默 1 ULP 漂移。解码、方向、alpha、色彩空间与
    bicubic resize 由下一项有界 libvips adapter 负责；端到端 parity 仍未完成，故 `INT-203` 继续未勾选。
  - 2026-08-27：现有唯一 `internal/media/imagevips.Processor` 已扩展固定语义输入方法，不新增第二套
    decoder：只接收已打开 `io.ReadSeeker`，沿用 256 MiB/尺寸/格式边界与 JPEG shrink-on-load，完成
    autorotate、sRGB、PIL `convert("RGB")` 等价的 resize 前 alpha removal、双轴 cubic resize，并直接输出
    tensor；不接受路径且所有 native 错误收敛为稳定错误。默认无 libvips 构建失败关闭，真实 libvips
    专项测试覆盖透明像素、非方图和格式错配并通过。尚缺固定公开 fixture 的 Python/libvips 像素容差、
    双架构 hash 与 ONNX embedding parity，`INT-203` 保持未勾选。
  - 2026-08-28：`make test-libvips` 已在 Docker 固定源码闭包中重建 production `libvips` tag，并通过
    `internal/media/imagevips` 专项测试；`make spike-ai` 的隔离 AI scorer/package/vector 状态也通过。
    结果证明默认 stub 没有掩盖当前 native preprocess 回归，但不提供最终模型 embedding parity、合法
    质量集或原生 amd64 证据，`INT-203` 状态不变。
  - 2026-09-01：`internal/aimodel` production parser 已接受严格 semantic format v2，固定三角色和四个
    contract ID；v1 只保留历史可读且 activation 明确拒绝。`internal/inference/sentencepiece` 已实现
    Linux FD-anchored 官方 C++ adapter、固定 64 token 合同和不可用失败关闭；production text session
    与 activation smoke 同时校验固定 token IDs、text graph ABI 和 768 维有限非零输出。图片 preprocess、
    generation、binary16 codec 与 deterministic contract 至此由正式 owner 完整串联，本项完成。
- [x] `INT-204` 实现幂等 `semantic_image` 任务、keyset admission、公平调度、lease/retry/cancel。
  - 2026-08-28：新增 capability-owned backfill service 与 migration 23 durable queue；幂等键只保存
    SHA-256 digest，请求摘要冲突失败，同 library/generation/mode 主动意图合并而不同 mode 冲突。入队时
    由 SQLite catalog port 计算图片/动图 eligible 数，不接受客户端计数；候选以 `asset_id ASC` bounded
    keyset 分页，`missing` 同时识别无 embedding 与 source fingerprint 变化。
  - 2026-08-28：claim 在同一 library 只允许一个 running/cancelling job，并按各库最近终态时间公平选择；
    lease heartbeat、operation/job 双状态、claimed revision、协作 cancel、terminal commit 均在短事务中推进。
    过期任务最多重领 3 次，claim revision 单调变化使旧 worker 无法提交；通用 AI interrupted recovery 已
    排除 restart-safe semantic operation，由 semantic queue 独立重排/失败。尚缺生产 worker 与真实
    inference/preprocess 接线、startup/app composition 及多库压力公平性证据，故本项保持未勾选。
  - 2026-08-28：新增 bounded `semantic.BackfillProcessor`，默认逐页 16 项、逐图顺序处理，不为资产创建
    goroutine；安全 source、libvips preprocess、generation-bound encoder 均为 capability ports。每页将
    embedding/succeeded/failed/stale 与 checkpoint 原子提交，source identity 变化记 stale，encoder/model
    unavailable 立即失败关闭而不生成假向量；候选在运行中消失时用 admission 余量收敛 stale，operation
    只有 completed=total 才能 succeeded。`internal/app` 已补 ContentService→semantic source 边界（复核
    library ID、fingerprint、格式并经 `internal/files` 打开）、durable DB ports 与通用 LeaseQueue adapter。
    该阶段没有在缺少 production `Run`/resident session 时提前启动 adapter；否则只会把任务批量标记
    model unavailable。后续记录在补齐门槛后才完成 lifecycle composition。
  - 2026-08-28：上述门槛中的 production `Run`、单 resident owner、hard timeout、idle unload 与 app
    lifecycle composition 已补齐；semantic queue 随 AI worker component 启动，启动恢复仍由自身 lease
    queue 负责，并共享全局 background admission。HTTP admission/cancel 后续也已接线；两库定向测试证明
    已完成过任务的库再次排队时，从未被服务的库优先，同时维持同库单 active job。至此任务状态机、
    keyset、调度、lease/retry/cancel 和 lifecycle owner 完整，本项完成；真实模型纵向归 `INT-202/203/209`，
    多库压力与 100k 容量归 `INT-210`，不重复阻塞本项。
- [x] `INT-205` 实现 SQLite embedding repository，短事务、source fingerprint 失效和 cascade。
  - 2026-08-27：新增 capability-owned `semantic.EmbeddingRepository` 与 SQLite adapter；单批受 Store
    上限约束且拒绝重复 asset，generation dimension/state 和全部 binary16 blob 在写事务外验证，事务内
    只复核 generation 后执行 bounded upsert。相同 fingerprint/vector 幂等重放不改 `created_at`，source
    fingerprint 改变可条件删除，asset/library/generation 继续由复合 FK cascade。
  - 2026-08-27：新增 embedding + `semantic_library_progress` + `semantic_jobs.checkpoint_id` +
    `ai_model_operations.completed_items/revision` 单一短事务提交；旧 worker 必须同时匹配 claim revision、
    progress revision 和 expected checkpoint，FK/计数/任一更新失败会整体回滚。成功、失败、stale outcome
    都受同一批量上限与 eligible/operation total 约束。job admission/claim/lease/retry/cancel 和 catalog
    keyset 读取已由 `INT-204` 接线；生产 inference worker、并发 backfill 与完成/失败端到端证据仍缺，
    `INT-205` 保持未勾选。
  - 2026-08-28：backfill processor 已用真实 SQLite queue/repository 集成测试覆盖 1-item bounded page、
    terminal success、embedding/progress 对齐和 encoder unavailable 失败关闭；安全媒体 source app adapter
    也已验证跨库拒绝。生产 worker lifecycle 随后已接入。真实 ONNX/libvips 纵向属于 `INT-202/203/209`，
    不再错误地阻塞 repository 自身；短事务、幂等、source fingerprint 失效和 FK cascade 合同均有定向测试，
    本项完成。
- [x] `INT-206` 根据 revision 1 的 SQLite exact 决策关闭独立 vector-index artifact 路径；embedding
  generation 本身就是唯一可重建索引，generation ready/active/retired 与旧 active fallback 由
  semantic/aimodel 的单事务 owner 管理。不得再创建 ANN temp 文件、第二 active pointer 或额外 checksum
  truth；若未来重新引入 ANN，必须新增 scope/ADR/迁移和损坏恢复任务，不能复用本项冒充已实现。
- [x] `INT-207` 实现文本 query、scope/filter、stable cursor、generation conflict 和 coverage 状态。
  - 2026-08-28：新增 capability-owned exact vector search repository 与 SQLite adapter。query 使用与图片
    embedding 相同的 float64 norm 合同；只扫描 active generation、enabled library、online source 且
    stored/current fingerprint 相同的行，逐行解码 binary16 并计算 cosine dot。Top-K 使用固定上限 200
    的 bounded heap，不把全部结果或 vector 集合载入内存；排序和翻页元组严格为
    `score DESC, asset_id ASC`，同分以 asset ID 稳定打破，library scope 与 disabled 排除已覆盖。
  - 2026-08-28：补齐 directory direct/recursive SQL scope；新增 read-transaction search snapshot，验证
    active/available generation、selected library enabled/online 和 directory ownership，并按有序 library
    generation/settings/progress tuple 生成稳定 fingerprint 与聚合 coverage。semantic search service 统一
    query canonicalization、真实 text encoder port、Top-K repository 调用和 authenticated opaque cursor；
    cursor 只保存 query hash，不保存查询正文，并绑定 scope、catalog revision、generation、progress
    fingerprint 及 `score DESC, asset_id ASC` continuation。篡改、query/scope 不匹配和 snapshot stale 已
    定向覆盖。search HTTP 已实现 query 参数白名单/默认上限、opaque resource ID、资产投影、coverage /
    excluded library DTO 和稳定错误映射，但生产 composition 不注入该可选 route。当前仍缺 accepted
    SentencePiece/text ONNX production adapter，因此 endpoint 继续不注册且 `INT-207` 保持未勾选；
    100k exact scan latency/RSS 也仍须以真实 768-d 数据验证。
  - 2026-08-28：search service 在生成结果/cursor 前新增 repository page 失败关闭校验：数量不得超过
    limit，library 必须属于 snapshot 且符合 selected scope，asset 不重复，score 必须有限，排序严格为
    `score DESC, asset_id ASC`，后续页必须位于 cursor continuation tuple 之后。hostile adapter 的越界、
    跨库、NaN、乱序、重复和回退页均收敛为内部损坏错误，不产生不稳定 cursor。记录：
    [语义搜索边界失败关闭](../changes/FIX-2026-08-28-semantic-search-result-validation.md)。该维护不提供
    production text adapter 或 100k×768 证据，`INT-207` 仍不勾选。
  - 2026-08-28：检修 race 复现 Raw URL Base64 末尾未使用 bit 可产生同 bytes 的不同 cursor 字符串；
    共享 cursor codec 现强制 decode→canonical re-encode 完全一致，语义篡改测试也改为确定性修改中间
    字符。snapshot 同时拒绝超长 generation、outcome 总量越过 eligible、member/excluded 重叠等内部
    损坏，并以脱敏 500 而非客户端 400 收敛。记录：
    [Cursor 规范编码与语义 snapshot 失败关闭](../changes/FIX-2026-08-28-canonical-cursor-decoding.md)。
    后续检修还将多库 eligible/completed/failed/stale/revision 汇总改为受检 `int64` 聚合，溢出时不返回
    回绕计数；该项同样属于内部失败关闭维护，不改变 `INT-207` 状态。
  - 2026-09-01：accepted fail-closed production text owner 已注入 `semantic.SearchService`，文本 query、
    scope/filter、stable cursor、generation conflict 与 coverage snapshot 由既有 capability owner 直接服务；
    空 reviewed catalog 时 endpoint 仍稳定 unavailable。100k 性能属于 `INT-210`，不再阻塞本实现任务。
- [x] `INT-208` 集成 API/auth/CSRF/限流/错误脱敏，不返回原始向量。
  - 2026-08-28：已实现 semantic library settings repository/service/HTTP：不存在设置时只读返回
    disabled revision 1；PUT 使用强 ETag CAS，启用前要求 active generation + available model，禁用不删除
    embedding。GET 仅返回 ID、状态、revision 与 coverage，不查询/返回 vector、路径或模型文件名。
  - 2026-08-28：已接 `POST /libraries/{id}/ai/semantic/jobs` 到 hashed-idempotency admission，服务端从
    settings 解析 active generation，不接受客户端 generation/eligible；返回公共 AI operation、Location、
    ETag 和 replay header。通用 operation cancel 已按 kind 分派到 semantic job+operation 同步取消，避免只
    改公开 operation 而留下 worker 继续运行。最后一批 progress 提交在同一事务将 enabled library 转为
    ready 或 degraded 并推进 coverage revision。路由沿用现有全局 auth/CSRF transport；search production
    composition、完整 auth transport 集成测试与错误矩阵仍缺，`INT-208` 保持未勾选。
  - 2026-08-28：search HTTP adapter 已按 OpenAPI 返回完整 `Asset`、nullable cursor、truthful coverage 和
    disabled/offline excluded libraries；拒绝 unknown/repeated 参数、无 library 的 directory scope 和越界
    limit，并覆盖 invalid/stale cursor、disabled/offline/model unavailable 与资产并发消失的脱敏映射。
    该 route 仅在显式提供 production search service 时注册；当前 app 未提供 text encoder，所以保持关闭。
    尚缺 production 注册和完整 auth transport 集成，`INT-208` 仍未勾选。
  - 2026-08-28：新增 migration 24 独立 semantic clear tombstone/request，避免用必须绑定 active
    generation 的 backfill job 伪装清除。clear 使用 settings ETag CAS 与 hashed idempotency，接纳事务
    立即禁用搜索并进入 `clearing`；与同库 backfill 双向互斥，即使模型 unavailable 仍可执行。独立
    worker 按最多 500 行短事务删除全部 generation embedding，最后事务删除 progress、切回 disabled
    并完成 operation；失败/取消为 degraded，绝不触碰原媒体、模型、收藏或人工标签。clear HTTP 已按
    frozen confirmation/If-Match/Location/ETag 合同接入，通用 operation cancel 已分派 clear queue。
  - 2026-08-28：search HTTP 的结果资产投影改为 catalog-owned bounded batch（最多 200 ID、单次 SQLite
    查询），由 catalog service 校验唯一/完整结果并恢复 semantic score 顺序，消除逐结果 `GetAsset` 的
    N+1。搜索与投影之间发生资产删除时返回 `semantic_cursor_stale`，不返回缺项或错序页面。
  - 2026-08-28：新增 semantic search 独立 transport policy，每个验证后 client address 每分钟 30 次，
    operation key 与普通 browse/filename search 隔离；全局 interactive semantic slot 固定为 1，竞争请求
    fail-fast `429 semantic_busy` + `Retry-After: 1`。bucket 继续复用唯一 transport limiter 的 4,096
    硬上限，未新增第二套无限 key map。
  - 2026-08-28：真实 `NewRoutes` middleware 测试证明未认证 semantic GET 在调用 search service 前返回
    `authentication_required`，认证后才进入可选 route；查询正文不出现在成功/错误响应。至此 settings、
    backfill、clear 与 search adapter 的 auth/CSRF owner 均复用全局 transport；`INT-208` 仅因 production
    text encoder/search composition 尚未成立而保持未勾选。
  - 2026-08-28：补充底层 encoder 返回含查询正文错误的 hostile stub；经真实 `NewHandler` 后，统一 500
    响应与结构化 request-completed 日志均不含 query、URL query string 或 runtime error。语义查询隐私不
    依赖具体 SentencePiece/ORT adapter 自觉脱敏。
  - 2026-08-28：search HTTP 将 catalog 批量 hydration 逐项绑定到 semantic match 的 asset/library ID，
    并要求来源仍 online；搜索与投影间的缺失、乱序、跨库 ID 复用按 `semantic_cursor_stale` 失败，offline
    转换按 `semantic_not_ready` 失败，不返回竞态后的错误资产。与 repository page 校验共同记录于
    [语义搜索边界失败关闭](../changes/FIX-2026-08-28-semantic-search-result-validation.md)。production
    search composition 仍受 text adapter Gate 阻塞，`INT-208` 不勾选。
  - 2026-08-28：落实 OpenAPI 已冻结的 2,048-byte cursor 上限；semantic service 在 snapshot/decode/
    encoder/vector 前拒绝过短或超长 token，HTTP 在 `url.ParseQuery` 前以 16 KiB raw-query 硬上限限制
    百分号编码放大。边界测试证明异常输入未进入服务依赖；这只关闭 transport 资源缺口，不解除 Gate。
  - 2026-08-28：共享 transport limiter 修复 wall-clock 回拨：未来 bucket 起点重基准但保留已用额度，
    semantic 30/min 不会被回拨重置，`Retry-After` 不越过一分钟，4,096 bucket 满载也会在一个窗口后恢复。
    证据：[限流时钟回拨恢复](../changes/FIX-2026-08-28-rate-limit-clock-rollback.md)。production search
    route 仍未注入，`INT-208` 不勾选。
  - 2026-08-28：共享 AI JSON decoder 修复 4 KiB 上限绕过；合法对象后的超长空白不再被 LimitReader
    伪装为 EOF，第 4,097 个字节会在 install/settings/backfill/clear 服务调用前统一拒绝；同时与既有
    JSON 写接口统一按媒体类型解析，接受 `application/json; charset=utf-8` 等合法参数，仍拒绝非 JSON
    或畸形媒体类型。证据：
    [AI JSON 请求体硬上限](../changes/FIX-2026-08-28-ai-json-body-limit.md)。这只关闭 transport 资源缺口，
    production composition 与 Gate 不变。
  - 2026-08-28：模型激活、operation 取消和 semantic settings/clear 共用的强 ETag parser 拒绝
    `r01`、`r+1` 等非规范 revision 别名；只有服务端实际签发的 canonical validator 能进入 service。
    证据：[AI 强 ETag 规范匹配](../changes/FIX-2026-08-28-canonical-ai-etag.md)。这只关闭并发控制
    adapter 缺口，production composition 与 Gate 不变。
  - 2026-09-01：production app 已注册 semantic image/text search 与 video semantic search service，
    继续复用全局 auth/CSRF、独立 30/min limiter、单 interactive slot、catalog hydration 和统一脱敏错误；
    API 从不返回 token、向量、模型路径或 runtime 原始错误。空 catalog/缺 build tag 仍失败关闭，本项完成。
- [x] `INT-209` 覆盖模型缺失/损坏、索引损坏、offline、取消、重启、源变化和原媒体不变。
  - 2026-09-01：语义 backfill durable job 的 claim/refresh、running cancel 与 terminal finish 已在 wall
    clock 持续回拨下保证完整 lease，并逐行钳制 job/operation updated/finished time；明确生命周期回归连续
    20 次通过。标签、视频、人脸分析/清除的同构状态机也同步修复，见
    [S2 durable job 时钟回拨收敛](../changes/FIX-2026-09-01-s2-job-clock-rollback.md)。最终审核模型的真实
    corruption/kill 纵向与原生双架构仍缺，故本项状态不变。
  - 2026-09-01：embedding 原子批次提交进一步同步钳制 progress、job、operation 与 library settings 时间；
    rollback 回归连续 20 次验证 checkpoint、coverage、完成计数及事务边界。最终外部证据缺口不变。
  - 2026-08-28：公共 AI operation owner 补齐持久化状态失败关闭：queued/active/terminal phase 与
    error code 必须一致，且 Get/create/transition 的 repository 返回值全部走同一校验；拒绝
    `queued+completed`、`running+queued`、成功带错误、失败/取消无错误及超长错误码；同时校验 Get
    返回请求 ID、create 返回初始 identity、transition 保持 identity/owner 并精确单步推进 revision，
    另一条合法 operation 也不能串单。异常只映射脱敏 internal error。证据见
    [AI operation 持久化状态校验](../changes/FIX-2026-08-28-ai-operation-state-validation.md)。这关闭
    状态读取子路径，不替代最终模型、native inference 或完整 app 强杀纵向，主项仍不勾选。
    安装/激活 admission 虽不经过 OperationService，也已在 ManagementService 出口校验 operation kind、
    激活模型绑定、Created/Replayed 互斥及新建 queued/revision-1 状态，不能由替换 adapter 绕过。
  - 2026-08-29：校验进一步前移到 install/activation admission 的 `Wake()` 之前；queue 返回必须绑定
    idempotency/request、candidate 或 model、availability revision 和初始 operation。故障注入篡改安装
    source 或激活 model 时稳定返回 repository error 且 wake 次数为 0。证据见
    [AI admission 唤醒前校验](../changes/FIX-2026-08-29-ai-admission-prewake-validation.md)。这关闭
    queue adapter 子路径，不改变主项或 Gate。
  - 2026-08-29：worker 端进一步复核 install/activation claim 的 request hash、candidate/model 绑定与
    `running` phase/revision；损坏 claim 在 installer、模型文件源或 native runtime 前失败关闭。故障注入
    证明 installer/runtime/open 调用均为 0。证据见
    [AI worker claim 返回值失败关闭](../changes/FIX-2026-08-29-ai-worker-claim-validation.md)。这维护已批准
    worker 边界，不提供最终模型或完整 app 强杀链，主项和 Gate 不变。
  - 2026-08-29：安装/激活 worker 的失败出口统一复用 operation owner 的终态 CAS 收敛；取消请求若在
    worker 读状态与 transition 之间推进 revision，会有限重读并优先完成为 cancelled，否则才保留原失败码。
    故障注入证明不再遗留 `cancelling` 等待重启。证据见
    [AI worker 终态 CAS 竞态收敛](../changes/FIX-2026-08-29-ai-worker-terminal-cas-convergence.md)。这不替代
    完整 app/native 强杀纵向，主项和 Gate 不变。
  - 2026-08-28：clear 已覆盖模型 unavailable 仍可接纳、queued cancel 后 degraded、首批删除后 lease
    过期重排、第二次 claim 从 durable operation progress 继续、最终 completed/total 精确且 settings 只在
    终态退出 clearing。backfill 已有 source fingerprint stale、取消与 lease recovery 定向证据；真实
    SQLite vector search 也已验证 source fingerprint 变化会排除旧 embedding，损坏/错误长度 binary16
    blob 会失败关闭为 `ErrInvalidEmbeddingRecord`，不会静默当零分。运行中 clear cancel 也已覆盖：保留已提交
    批次，lease 收敛 job/operation 为 cancelled，settings 为 degraded，剩余 embedding 留待显式重试。
    尚缺 install worker/DB/native inference 的完整强杀纵向和完整生产模型纵向，故保持未勾选。
  - 2026-08-28：app-level 临时 managed 包经生产 catalog/validator/registry/activation transaction 建立
    active generation 与一条 embedding 后，实际篡改 text graph 会把模型从 available r1 收敛到
    unavailable r2，同时 active pointer、generation 和 embedding 均保留；该状态跨数据库组件停启持久，
    重启后恢复精确字节会回到 available r3，generation/embedding 仍在。同一纵向的只读媒体哨兵 SHA-256、
    size 与 mtime 前后完全一致。证据见[managed model corruption recovery](../evidence/int-001/managed-model-corruption-recovery-2026-08-28.md)。
    这关闭 managed 文件损坏/重启恢复及该路径原媒体不变语义，但包仍是合成审核 fixture；install worker+
    DB registration/native inference 强杀和最终 native 模型纵向仍缺，主项保持未勾选。
  - 2026-08-28：semantic backfill 新增真实子进程强杀纵向：helper 经生产 SQLite store claim 100 ms
    lease 后被 OS kill，新 store 在 lease 到期后精确重排一次；下一次 claim 的 attempt 1→2、claim revision
    2→3，而 checkpoint 101 与 completed/total 1/2 不变。证据见
    [semantic backfill process-kill recovery](../evidence/int-001/semantic-backfill-sigkill-recovery-2026-08-28.md)。
    这关闭 backfill queue 强杀恢复，但不代替 install worker/DB、真实 native inference 或最终审核模型纵向，
    `INT-209` 仍不勾选。
  - 2026-08-28：managed install 新增真实子进程强杀纵向：production worker 从 SQLite durable queue
    claim 后完成 managed store staging/fsync/atomic rename，并在 `RegisterInstalled` 前被 OS kill。新进程将
    running operation 收敛为 failed/`operation_interrupted`；完整 final 经 reconcile、catalog/架构和全文件
    校验后只登记为 available/inactive，不自动激活。证据见
    [managed install process-kill recovery](../evidence/int-001/managed-install-worker-sigkill-recovery-2026-08-28.md)。
    仍缺真实审核 native 模型 inference 强杀纵向，主项保持未勾选。
  - 2026-09-01：按 CR-2026-022 的 S2/S4 Gate 分离，本任务以 production-boundary synthetic/candidate
    fault matrix 验收；最终审核模型的完整容器 kill/recovery 由 `INT-401/405` 持有。模型缺失/损坏、
    embedding 损坏、offline、取消、lease 重启、source fingerprint 变化和原媒体 sentinel 已全部覆盖，本项完成。
- [x] `INT-210` 运行本地 100k 查询、并发浏览、backfill、Go memory 与 DB/WAL 空间测试；最终模型 RSS、
  final image 和 native amd64/arm64 pairing 由 `INT-402/403` 验收。
  - 2026-08-31：再次在 darwin/arm64、`GOMAXPROCS=4` 强制执行 10k 目录/100k 资产基线；普通并发浏览
    P95 `433 µs`、并发搜索 P95 `23,529 µs`，但 production keyset P95 `319,616 µs` 仍超过冻结的
    `250,000 µs`，因此 `make spike-capacity` 按预期失败关闭。order-first 候选同轮 P95 为首页
    `9,822 µs`、续页 `12,243 µs`、完整 hydration `9,913 µs`，26 个矩阵/22 个 cursor 场景继续与
    production 结果等价；候选尚未取得 owner 接受且缺 native 双架构，不能据此勾选本项。
  - 2026-09-01：将有序外层扫描 + 精确 FTS membership 提升为 repository 唯一实现；两库 100k 全矩阵
    继续逐 ID 等价，`make spike-capacity` 的 production keyset P95 降至 `133,637 µs` 并得到
    `budgetViolations=[]`。本地 production 查询计划与 100k 预算子项关闭；最终模型联合负载、原生 Linux
    amd64/arm64 和 release owner 验收仍缺，因此 `INT-210` 保持未勾选。
  - 2026-09-01：强制基线复跑通过：`searchKeysetP95Us=130238`、并发浏览 P95 `369 µs`、并发搜索 P95
    `66276 µs`、peak Go heap `51,979,024` bytes、DB+WAL `157,274,112` bytes，`budgetViolations=[]`；
    100k×512 face 聚类同轮为 7.174 秒、Go memory sys `409,381,208` bytes。S2 本地容量完成，最终模型
    inference RSS 与 native paired final image 明确保留给 `INT-402/403`。证据见
    [S2 local capacity refresh](../evidence/int-001/s2-local-capacity-refresh-darwin-arm64-2026-09-01.md)。
- [x] `INT-211` 实现 aimodel service：审核 manifest、安装状态、generation、托管/直接来源唯一 owner。
  - 2026-08-27：已建立 service/repository owner、opaque ID、幂等 package 登记、model snapshot 与
    availability revision；generation/activation、operation 和真实 managed/direct source 尚未完成。
  - 2026-08-27：已补 durable operation service/SQLite CAS，覆盖 queued/running/cancelling/terminal、
    phase/progress、stale revision、协作取消和启动 `operation_interrupted` 收敛；generation/activation、
    worker claim 与 app/API composition 仍未完成，主任务保持未勾选。
  - 2026-08-27：新增单一 `aimodel.ManagementService`，统一 model list、candidate scan/current-only
    resolve、install admission、operation query/cancel；模型管理 HTTP 边界已覆盖 ETag/If-Match、幂等键
    校验和脱敏 wire test。durable worker、生产 app composition、generation/activation 仍未完成。
  - 2026-08-27：安装请求现以 migration 22 的 `ai_model_install_requests` 和 operation 同事务持久化，
    幂等键冲突不复用任务；queued 可在重启后继续 claim，只有 running/cancelling 被 interrupted 收敛。
    单 worker 已能执行 installer、轮询协作取消并写入稳定终态；生产 app composition 尚未完成。
  - 2026-08-28：scanner、installer、activation admission/worker、generation commit、availability refresh
    与 managed/direct source router 均已由单一 `ManagementService` 对 API 暴露，并在 `internal/app` 完成
    lifecycle composition；HTTP 不直接协调 repository/files/runtime。本项实现完成，真实包纵向与故障矩阵
    分别留给 `INT-209/215`，不重复阻塞 service owner。
- [x] `INT-212` 实现管理员触发的有界 managed copy、取消、临时文件、空间预检、校验和原子发布；
  revision 1 不实现网络下载或续传。
  - 2026-08-27：已实现 8 GiB quota、`max(1 GiB, 10%)` 安全余量、同文件系统 `.partial-*`、
    context cancel、逐文件 size/SHA-256 复核、file/dir fsync、Linux `RENAME_NOREPLACE` 原子发布及
    已存在包重新校验；durable operation、HTTP admission/取消和重启 reconciliation 待续。
  - 2026-08-27：durable operation 及启动 interrupted 收敛已完成；managed store reconciliation 只删除
    确认的 `.partial-*` 目录，已发布/未知 entry 只计数不删除。worker admission、HTTP 取消和操作与
    installer 的异步组合仍待续，主任务保持未勾选。
  - 2026-08-27：HTTP 已实现 install `202`、operation GET、强 ETag 协作取消和稳定错误映射；尚未接
    durable idempotency/worker 与生产 lifecycle，不能视为可用安装流程。
  - 2026-08-27：durable idempotency/admission/claim 和 installer worker 已实现，覆盖 queued 重启保留、
    原子 claim、重复请求回放、冲突拒绝及执行期取消轮询；仍缺生产 lifecycle 装配和强杀集成证据。
  - 2026-08-28：installer worker、operation cancel 和 managed-store reconciliation 已进入生产 lifecycle；
    复制仍只消费服务端 current candidate，不接受路径/URL。本项实现完成。真实进程强杀、磁盘满和恢复
    纵向属于 `INT-215`，不会被错误计作复制实现缺失。
  - 2026-08-28：补齐安装成功后的 operation 终态收敛：`finalizing`/`succeeded` CAS 与并发取消冲突时
    重新读取唯一 operation owner，cancelling 收敛为 cancelled；仍为 running 的事务失败收敛为
    failed/`internal_error`，不再无限遗留 running/cancelling。定向测试分别注入两种最终事务冲突。
- [x] `INT-213` 实现固定 `/models` 安全枚举、opaque candidate、托管复制和只读直接来源校验。
  - 2026-08-27：已复用 `internal/files.Root` 完成固定根的有界 direct-child `.foliomodel` 枚举、
    no-symlink/no-cross-device regular-file 打开、流式 SHA-256、64 package/16 file/4 GiB 上限，以及
    内存 scan revision + opaque candidate 失效；后续补充 Linux anchored `Fstatfs(ST_RDONLY)`、扫描
    source identity 绑定、managed copy installer 和 direct 安装前复核。真实启动/session-load 持续复核、
    app composition 与 HTTP 尚待完成，因此主任务保持未勾选。
  - 2026-08-27：candidate ID 由服务端 current-scan owner 解析，新扫描后旧 ID 稳定返回 stale；HTTP
    只输出 reviewed metadata 或固定 `unknown`，测试证明不泄露 `/models`、文件名或 source identity。
  - 2026-08-27：固定 `/models` source、`/app/data/models` managed store、scanner/installer/admission/worker、
    database 窄转发和 authenticated HTTP 已接入生产 `internal/app` 生命周期；缺失 `/models` 不阻断浏览，
    扫描明确返回 unavailable。内建审核 catalog 仍为空，等待合规签署后才能接受真实包。
  - 2026-08-28：启动、管理员扫描、激活和 semantic cold load 均复用 source router/availability owner
    重新验证 installed direct/managed 包；opaque candidate、新 scan 失效、只读 direct 与 managed copy 边界
    已完整接线。本项实现完成；空审核 catalog 是发布/合规 fail-closed 状态，不是否定安全枚举实现。
- [x] `INT-214` 实现模型选择/激活、旧 generation fallback、直接来源失效和显式 unavailable 错误。
  - 2026-08-27：migration 22 已增加 durable activation request；SQLite owner 已实现 available/revision
    admission、幂等请求、原子 claim，以及“退役旧 generation + 插入新 active generation + 切换双指针
    + operation succeeded”单事务提交。唯一约束注入失败证明事务完整回滚并保留旧 active。真实 runtime
    load/validate worker、direct session-load 复核和 activate HTTP 尚未完成，主任务保持未勾选。
  - 2026-08-27：activation worker 已接 runtime/source ports，只有 runtime 成功返回合法维度后才构造
    generation commit；stale availability、未审核 manifest、来源失效或取消均不切 active。真实 ORT adapter、
    managed/direct activation source adapter、worker app composition 和 HTTP 仍待续。
  - 2026-08-27：新增 storage-mode router；direct 激活前除只读 mount/source identity 外，会重新流式
    校验 manifest 中每个文件的 size/SHA-256，managed 激活重新验证完整已发布包、精确 content-hash
    identity、文件名和 regular/no-exec 约束。来源 adapter 已完成，ORT/worker composition/HTTP 仍待续。
  - 2026-08-27：runtime opener 改为 Linux 已打开句柄 + `/proc/self/fd/<fd>`，避免 200～400 MB ONNX
    进入 Go heap 或复制到二次临时目录；不向 runtime/API 传真实模型路径，非 Linux 失败关闭。
  - 2026-08-27：activation validation 已有 ORT adapter 可严格加载并检查两张图，但生产 app worker、
    activate HTTP、tokenizer/deterministic inference 验证、hard-timeout 决策和双架构证据仍缺，因此
    不关闭主任务。
  - 2026-08-27：新增 activation admission owner，以 model ID + availability revision 绑定幂等 hash，
    queued operation 与 request 同事务写入并只在首次创建时唤醒 worker；生产 composition 在一次启动
    recovery 后并行运行 install/activation 两条 durable queue，避免重复 recovery 竞态。activate HTTP
    已按 `"{modelId}-r{availabilityRevision}"` 强 If-Match、幂等回放、Location/operation ETag 接入；真实
    catalog 仍为空，且 `INT-203` 前不会产生可用语义 generation。
  - 2026-08-28：新增唯一 availability refresh owner；应用启动在 worker 前复核所有 installed
    managed/direct 包，管理员触发固定目录扫描时再次复核。来源缺失、被替换、不安全或已移出审核 catalog
    只以 availability CAS 标 unavailable，保留 active generation/embedding；精确来源恢复后推进 revision
    并回到 available。取消立即停止，CAS 冲突只按当前 revision 有界重试一次。该项关闭状态主动刷新缺口，
    但真实审核 catalog、模型损坏纵向和 production text adapter 仍阻断 `INT-214`。
  - 2026-08-28：运行期激活验证与 semantic cold session load 也接入同一 availability owner；来源验证、
    runtime 打开期间出现 `model_source_unavailable`/`model_incompatible` 时以当前 availability revision
    标 unavailable，再让 operation/encoder 失败关闭。用户取消、deadline 或进程 shutdown 不会误标模型；
    已有 resident session 不被异步强拆，旧 active generation 和 embedding 不变。定向测试覆盖激活与冷加载
    两条路径。真实审核 catalog、完整 tokenizer activation fixture 与源文件损坏纵向仍未完成，主项不勾选。
  - 2026-08-28：activation 最终 SQLite 事务除 operation revision 外，再次 compare-and-swap admission
    绑定的 availability revision；模型在长时间 runtime load 中即使 unavailable 后又恢复为 available，旧请求
    也不能提交 stale generation、退休旧 generation 或切换 active 指针。最终事务拒绝后，worker 会重新读取
    operation：cancelling 收敛为 cancelled，其余 running 以稳定 `model_unavailable` 失败，不再遗留 running。
    定向证据见 [activation availability CAS](../evidence/int-001/activation-availability-cas-2026-08-28.md)。
    真实审核 catalog、完整 tokenizer activation fixture 与源文件损坏纵向仍阻断主项。
  - 2026-09-01：semantic activation 现在只允许 format v2，并在 generation commit 前以安全 FD 同时验证
    SentencePiece metadata、固定 token IDs、image/text graph ABI 与有限非零输出；legacy v1 在打开任何
    runtime 文件前失败。运行期 text/image session 都复核 active generation、available model 与 source，
    切代/来源失效保留旧可靠 generation/embedding 并返回稳定 unavailable。实现要求已完成；最终审核
    catalog 和故障纵向分别归 `INT-209/215`。
- [x] `INT-215` 覆盖恶意包、错架构/hash、symlink/mount、磁盘满、强杀、恢复和诊断脱敏；revision 1
  无网络 source，因此 SSRF/重定向不是本切片测试面。
  - 2026-08-28：managed final graph 真实字节篡改已通过生产 validator → availability CAS → 数据库组件
    停启 → 精确恢复纵向，并证明不退休 active generation、不删除既有 embedding；该路径的只读媒体哨兵
    SHA-256/size/mtime 不变。既有 unit/integration 还覆盖错架构/hash、candidate stale、managed staging
    reconciliation、direct symlink/read-only 边界和错误脱敏；install worker/DB/native inference 的完整强杀、
    磁盘满生产纵向、nested mount 及最终模型包仍缺，因此本项不勾选。
  - 2026-08-28：semantic backfill 的生产 SQLite queue 已通过真实子进程强杀/lease recovery，保留
    checkpoint/progress 并用更高 claim revision 排除旧 worker；该证据只关闭 backfill 子路径。install
    worker/DB、磁盘满生产纵向、nested mount 与最终模型包仍是 `INT-215` 缺口。
  - 2026-08-28：production `ManagedModelStore` 已通过原子 rename 两侧的真实子进程强杀：copy 中断仅留
    `.partial-*` 且新 store 精确清除、不出现 final；publish+parent fsync 后中断则新 store 只报告并保留
    完整 known final，随后幂等 publish 精确校验复用。因开发盘已低于冻结的 10% reserve，测试仅通过未导出
    seam 注入 2 GiB 容量探针，其他 staging/copy/hash/fsync/rename/reconcile 全走生产实现。证据见
    [managed publication process-kill recovery](../evidence/int-001/managed-publication-sigkill-recovery-2026-08-28.md)。
    仍缺 install worker+durable operation+DB registration 同一强杀纵向、真实 ENOSPC、nested mount、
    Linux/amd64 与最终模型包，因此本项不勾选。
  - 2026-08-28：启动 recovery 已补齐 publish→DB crash window：managed store 只返回最多 256 个排序
    content hash，不返回路径；只有完整未截断报告中同时匹配内建 catalog、当前架构并通过 manifest/
    regular/no-exec/size/SHA-256 全校验的 final，才经唯一 aimodel service 幂等登记为 available 且 inactive。
    unknown、corrupt、wrong-architecture 或截断报告均不登记、不激活、不删除。app corruption 纵向现从该
    recovery owner 开始，而非测试直接注册。证据见
    [managed orphan reconciliation](../evidence/int-001/managed-orphan-reconciliation-2026-08-28.md)。
    当前 production catalog 仍为空，因此机制已实现但没有获批真实包可恢复；最终 catalog/模型、组合强杀、
    ENOSPC、nested mount 与 Linux/amd64 仍阻断本项。
  - 2026-08-28：publish→DB crash window 已进一步通过真实 install worker + SQLite 子进程强杀，而非仅
    独立 publisher 测试：重启恢复保持 operation 失败、model inactive 且无 staging 残留。证据见
    [managed install process-kill recovery](../evidence/int-001/managed-install-worker-sigkill-recovery-2026-08-28.md)。
    这关闭 install worker/DB registration 强杀子项；真实 ENOSPC、nested mount、Linux/amd64、最终模型及
    native inference 强杀仍阻断 `INT-215`。
  - 2026-08-28：托管发布的预检后写路径现区分 source read 与 managed target write；目标
    mkdir/write/fsync/rename 的 `ENOSPC/EDQUOT` 保留 errno 并稳定映射为 `insufficient_space`，不再被
    `model_incompatible` 吞掉或降为 `internal_error`。定向测试证明同名 source reader errno 不会被误分类。
    随后的原生 Linux/arm64 128 KiB tmpfs 验证见下条。
  - 2026-08-28：固定 `/models` 的 direct package nested-mount 子项已在原生 Linux/arm64 隔离容器
    通过：扫描时 bind-mounted `.foliomodel` 由 kernel `RESOLVE_NO_XDEV` 拒绝；正常扫描后再以 bind mount
    替换同名 package，旧 source identity 的 reopen 同样返回 cross-device，未读取替换内容。证据见
    [direct model nested-mount boundary](../evidence/int-001/direct-model-nested-mount-linux-arm64-2026-08-28.md)。
    Linux/amd64、最终模型和 native inference 强杀仍阻断 `INT-215`。
  - 2026-08-28：managed store 的真实 kernel ENOSPC 子项已在原生 Linux/arm64 隔离容器通过：只注入
    预检容量探针以进入 post-preflight 写路径，production publisher 向 128 KiB tmpfs 写入 512 KiB 模型，
    实际返回 `ENOSPC` + stable `insufficient_space`，无可见 final 且 `.partial-*` 已清除。证据见
    [managed model ENOSPC](../evidence/int-001/managed-model-enospc-linux-arm64-2026-08-28.md)。
    native inference 强杀结果见后续条目；Linux/amd64 与最终审核模型仍阻断 `INT-215`。
  - 2026-08-28：production Go/cgo ORT image session 的 native process-kill 子项已在 Linux/arm64 通过：
    177 MiB SigLIP 1 FP16 候选 image encoder 进入 C API `Run`，独立 goroutine 在 20 ms 后确认仍 active，
    父进程随即 SIGKILL；全新进程重新加载同一只读模型并完成 768 维有限值推理。证据见
    [native inference process-kill recovery](../evidence/int-001/native-image-inference-sigkill-linux-arm64-2026-08-28.md)。
    这关闭 runtime 子项但不把候选提升为最终审核模型；Linux/amd64、最终 catalog/model 与完整 app
    backfill composition 仍阻断 `INT-215`。
  - 2026-09-01：format v2 hostile manifest、tokenizer exactly-one-before-open、default/native runtime
    unavailable、direct nested mount、managed kernel ENOSPC、install/backfill/native Run 强杀恢复、hash/
    architecture/symlink 与 API/log 脱敏已有 production-boundary 证据。按 CR-2026-022，最终包同 digest 的
    native 双架构/供应链纵向归 `INT-403/404/405/407`，不再重复阻塞 S2，本项完成。
- [x] `INT-216` 复审 [S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)：
  2026-08-28 判断 **No-Go**。实现已到当前合同/输入允许边界；ADR-0014、最终审核 catalog/model、合法
  质量集、原生 Linux/amd64、最终 100k 全进程容量和完整 app native-kill 纵向未解除。复审完成不等于
  当时的复审完成不等于 Release Gate Go，也不授权发行 UI；不再用重复 arm64 合成测试代替这些输入。
  - 2026-08-28：新增可执行 No-Go fitness check；只要 ADR-0014 仍为提议且本 Gate 为 No-Go，
    `make arch-check` 就强制 production catalog 为空、保留已批准 Semantic 管理边界，并禁止生产组合注入
    `SemanticSearch`。证据见
    [INT-S2A No-Go 生产组合 fitness check](../changes/FIX-2026-08-28-int-s2a-no-go-fitness.md)。
  - 2026-09-01：fitness check 已随 ADR-0014 的 fail-closed backend 接受而转换：要求空 reviewed catalog，
    要求 semantic search route 已组合，同时继续禁止 face runtime/route。Gate 仍因 `INT-209/210/215` 的
    最终模型、质量、native 容量和供应链证据为 **No-Go**。

## S2B：标签建议与视频代表帧搜索后端（revision 2，当前阶段）

- [x] `INT-221` 冻结受控标签词表输入、Top-K、阈值、suggestion 生命周期和 ignore 语义。
- [x] `INT-222` 实现 tag text embedding/cache 和 suggestion repository；不直接写人工标签表。
  - 2026-08-29：migration 25、controlled tag binary16 cache、Top-5 finite confidence 与稳定 tag-ID tie、
    generation/vocabulary/source replacement、人工 tag/review suppression 已通过 capability/SQLite 测试。
  - 2026-08-30：migration 32 补齐独立 missing/all durable job、hashed idempotency、lease/retry/cancel、
    每资产显式 outcome 与每库 coverage；零 Top-5 suggestion 也记录 ready，不再用 suggestion 行数推算完成数。
    tag embedding builder 按 active generation/vocabulary 缓存缺失文本向量，job 在资产评分前失败关闭。
  - 2026-09-01：stale generation/source suggestion invalidation 在 wall-clock 回拨时保持 revision 与时间约束；
    明确回归连续 20 次通过。
- [x] `INT-223` 接受建议时调用 curation service，并处理 tag revision/precondition/cascade。
  - 2026-08-29：新增 curation-owned append-only `AddAssetTag`，suggestion service 逐项复核 revision；accept
    成功后才记 review，dismiss/accepted 跨重建抑制重复建议，批量硬上限 100。
  - 2026-08-30：migration 29～31 保留跨重建 lineage、hashed request/item outcome ledger 与每库 review
    revision；进程在 curation 成功后中断可安全重放而不重复写 tag。review clear 使用强 ETag、独立 lease
    queue 与二次确认，回归证明只删 review audit，已确认 `asset_tags` 保留。
  - 2026-09-01：reviewed suggestion 的 retirement 与幂等 request outcome/completed 迁移在回拨时仍可原子重放；
    两条明确回归各连续 20 次通过。
- [x] `INT-224` 实现视频 storyboard generation 依赖和 frame embedding，禁止第二套抽帧。
  - 2026-08-30：semantic source 只打开 thumbnail owner 已发布的完整 WebP sprite，由 libvips 有界切分
    4/10 个 cell；semantic worker 不读取原视频、不调用 FFmpeg。migration 26/27、完整 plan 校验、独立 durable
    queue、lease/attempt/recovery，以及 frames/progress/checkpoint/operation 的单事务 CAS 已通过测试。
- [x] `INT-225` 实现 max/best-frame 排名、video hit DTO 与 source/storyboard version 失效。
  - 2026-08-30：查询只接纳与当前 ready storyboard source/transform/plan 一致的完整 4/10 帧组；逐视频按
    `score DESC, ordinal ASC` 选最佳帧，最终按 `video_score DESC, asset_id ASC` 走独立加密 cursor，并返回
    matched-frame 证据。图片/视频交互检索共用单一全局 admission slot。
- [x] `INT-226` 覆盖无故事板、部分帧失败、10→4 降级、取消、重复任务和缓存淘汰重建。
  - 2026-08-30：capability/SQLite/app 测试覆盖 no-storyboard degraded、单帧失败不提交、既有
    storyboard 10→4 fallback、幂等 replay/coalesce、queued/running cancel、三次 lease 恢复后终止、checkpoint
    重复 CAS，以及 ready cache 丢失后的原子重排与 worker wake；`make test-libvips` 通过 production tag 构建。
  - 2026-08-30：video durable job、operation cancel 与完整 storyboard cache source 已进入 production
    composition，共享 background=1；video query route 仍因未接受的 text runtime Gate 有意不注册。
    完整本地证据见 [S2B implementation evidence](../gates/POST-MVP-5/int-s2b-local-implementation-evidence-2026-08-30.md)。
- [x] `INT-227` 实现并验证 governed quality 输入/评分/失败关闭合同、tag precision 与 video Top-20
  重算；最终真实审核结果与批准签署由 `INT-403/411` 验收。
  - 2026-09-01：tag/tag-review-clear/video durable job 的 claim、lease、进度/批次、running cancel 与 finish
    已补齐 wall-clock 回拨回归，各连续 20 次通过；这只关闭任务可靠性缺口，不产生审核集质量结果，
    `INT-227` 继续未勾选。
  - 2026-09-01：tag review 幂等请求的逐项 outcome 与 completed 迁移也在回拨下保持可重放，明确回归连续
    20 次通过；正式审核集 Gate 状态不变。
  - 2026-08-31：新增 `make verify-intelligent-media-quality`，把真实质量结果绑定到非 synthetic schema v2
    ordinary-media manifest、manifest/model hash 和 product/ML/QA 批准引用；重新计算逐 tag precision/
    recall、宏平均、人工接受率及 100-video 中英文 Top-20 success，低于批准 tag 门槛或视频 80% 时退出
    失败。格式、时长、4/10 帧、motion/static、indoor/outdoor 与结果 ID 完整性均由 validator 强制。
    当前没有真实审核数据、最终模型结果或三方批准，故评分入口完成不等于 `INT-227` 完成。
  - 2026-09-01：按 CR-2026-022，S2 验收 scorer、schema v2 输入治理、逐项结果重算、阈值失败和 C/D
    fallback 合同；最终 governed dataset/model output 与 product/ML/QA 批准保留给 S4。后端质量验证入口及
    失败语义已完成，本项以 **Backend Ready / Release No-Go** 收口。
- [x] `INT-228` 复审 [S2B Backend Evidence Ready](../gates/POST-MVP-5/int-s2b-backend-evidence-ready.md)：
  2026-08-30/31 判断 **No-Go / Implementation Authorized**。`INT-221～226` 的实现与失败矩阵已完成，
  但 `INT-227` 合法 tag/video 质量集、最终模型/runtime 供应链、原生双架构和最终联合容量仍未解除。
  复审完成不等于 Gate Go，不授权 S3 UI；与 `INT-216` 使用相同的“复审任务可完成、Gate 继续 No-Go”口径。

## S2C：人脸聚类与人物库后端（revision 2，Backend Ready）

2026-08-31：产品用户明确授权以本机、非公开、非训练/非模型发布方式使用一组网络图片做功能测试。
隔离的 YuNet → align → SFace 链路均衡抽样 135 图，135 图解码成功、79 个候选均产生有效 128 维
embedding，且没有复制原图或持久化 crop/embedding；见[本地功能冒烟证据](../evidence/int-001/face-functional-local-arm64-2026-08-31.md)。
该结果只证明候选链路能运行；没有 face-level ground truth，不能计算 recall、ROC、聚类 precision 或偏差，
也不关闭模型许可、隐私发布准入、原生 Linux 双架构或 production composition，因此本节计数保持不变。

2026-08-31：产品用户要求继续到全部 S2 完成后统一汇报，授权 S2C 进入 backend-first 实现。Gate 调整为
**Implementation Authorized / Release No-Go**：允许 fail-closed capability、persistence、worker 和 HTTP
adapter，但在质量、隐私发布、模型供应链和原生双架构证据通过前不得注册可用 runtime 或消费者 UI。

- [x] `INT-241` 实现 face detect/quality/embedding，归一化 box、fingerprint 和输入上限。
  - 2026-09-01：worker 将 analyzer 的 `ErrRuntimeUnavailable` 作为代次级故障立即以
    `model_unavailable` 终止；不推进 checkpoint/覆盖计数、不继续扫描、不创建部分 cluster build，设置转为
    `awaiting_model`。worker context 取消仍保留 durable claim 交给 lease recovery，不伪装成资产失败。记录见
    [人脸 runtime 不可用终止语义](../changes/FIX-2026-09-01-face-runtime-unavailable-terminal.md)。当时最终审核
    runtime/权重仍缺；后续 fail-closed 组合包/activation 实现见本项末尾收口记录。
  - 2026-09-02：补齐 settings、missing/all admission、derived/manual clear 的 HTTP adapter 与 application
    control owner；active generation 只从服务端设置派生。通用 operation 取消会按 kind 回到 face job/clear
    唯一状态机，缺少 owner 时失败关闭；production composition 仍由架构测试禁止注入 face dependencies。
    记录见[人脸控制面与取消状态机闭环](../changes/FIX-2026-09-02-face-control-and-cancellation.md)。当时
    detector/quality/embedding runtime、审核权重与 Linux 双架构仍缺；后续后端 runtime/activation 已收口，
    审核权重与最终双架构继续由 `INT-250/251` 失败关闭。
  - 2026-09-02：关闭设置现在与任务停止原子绑定：queued job 直接 cancelled，running job 进入 cancelling，
    worker heartbeat 后收敛为 cancelled；禁用不删除 observation、人物或人工约束。claim/disable 写事务竞争、
    operation/job 同步和禁用状态保持均有 SQLite 回归。
  - 2026-09-01：整仓回归暴露 wall clock 早于 library/job 创建时间时设置插入与停用取消会违反时间约束。
    settings、analysis job 与 operation 现在逐行把 updated/finished time 钳制到各自 created time，明确的
    rollback 回归各连续 20 次通过；后续又把同一规则覆盖分析 admission、claim/refresh 完整 lease、进度、
    failed finish 与 queued cancel，并以持续回拨生命周期回归连续 20 次验证。状态/revision/原子取消语义
    不变。记录见
    [人脸设置时钟回拨收敛](../changes/FIX-2026-09-01-face-settings-clock-rollback.md)。
  - 2026-09-02：processor 进一步在 detector 前拒绝视频和未知格式，只允许冻结的 JPEG/PNG/WebP/GIF，
    并继续把 source fingerprint 复核放在 runtime 调用前；最终审核 runtime/权重仍是本项剩余阻塞。
  - 2026-08-31：审计 Open Model Zoo 的 Apache-2.0 ArcFace ResNet100 ONNX 替代候选；固定 261,036,388
    bytes、SHA-256 和上游 SHA-384 后，图虽可加载并暴露 `data [1,3,112,112] → fc1 [1,512]`，首个真实
    对齐人脸在冻结 ONNX Runtime 1.28 的 BatchNormalization 执行中失败。候选在质量/容量前明确标为
    rejected，不为兼容它静默改图或引入 OpenVINO。隔离的确定性规范化实验另生成新 digest，并以单一
    tensor 相对 OpenCV 原图执行达到 max-abs `6.736e-6`；135 张授权本地图片产生 79 个有限 embedding，
    Open Model Zoo 将该精确来源标为 `LResNet100E-IR,ArcFace@ms1m-refine-v2` 并声明 Apache-2.0 分发，
    但 InsightFace 当前官方条款将公共预训练权重限制为非商业研究、商业使用需另行许可；前者不能自动
    覆盖后者的精确权重限制。该 261 MB 派生图仍缺商业许可或权威书面澄清、新合同、正式质量、容量、
    双架构和签署，因此继续 compliance hold，未进入 production catalog。证据见
    [ArcFace replacement rejection](../evidence/int-001/arcface-resnet100-replacement-rejection-darwin-arm64-2026-08-31.md)
    与[normalized candidate](../evidence/int-001/arcface-resnet100-normalized-candidate-darwin-arm64-2026-09-01.md)。
  - 2026-09-01：新增 fal 官方 AuraFace v1 候选筛选。精确 `glintr100.onnx` 为 260,694,151 bytes、
    SHA-256 `a7933e...25c60`；官方仓库标 Apache-2.0、明确面向商业场景并声明使用 commercial dataset。
    原图无需改图即可由 ORT 1.28 执行；授权本机 135 图中 79 个 YuNet candidate 全部产生有限 512 维
    embedding。官方卡未披露数据集身份、许可方、同意/删除依据，且尚无 production adapter、正式质量、
    native 双架构、容量与合规签署，因此只保留为 candidate，不进入 production catalog。证据见
    [AuraFace v1 replacement candidate](../evidence/int-001/auraface-v1-candidate-darwin-arm64-2026-09-01.md)。
  - 2026-09-02：按操作者澄清确认授权功能根包含多个人，目录不是身份 ground truth。功能报告 schema v2
    已将 `same/different` recall/FPR 纠正为组内/跨组 accept rate，并固定
    `directory-group-only-not-identity-ground-truth`；相同 135 张只读有界样本重跑仍得到 79/79 个有限
    embedding。该记录只补功能证据，不关闭真实 50×20 ground truth、bias、模型合规或发布 Gate。记录见
    [目录分组口径纠偏](../changes/FIX-2026-09-02-face-functional-group-metrics.md)。
  - 2026-09-01：复核 Intel 官方 `face-reidentification-retail-0095` 替代源。Open Model Zoo 对精确 IR 文件
    固定 size/SHA-384 并应用仓库 Apache-2.0，Intel 官方用例另以 MIT 发布下载脚本和生产用途说明；但权重
    只以 OpenVINO `.xml + .bin` 分发，采用它会新增第二套 production inference runtime，而官方资料只披露
    LFW evaluation、没有训练数据/同意/删除链。未用反向转换绕过 ADR，候选保持 architecture/provenance
    hold。证据见
    [Intel alternative hold](../evidence/int-001/intel-face-reidentification-retail-0095-hold-2026-09-01.md)。
  - 2026-09-01：production ONNX adapter 新增严格 `face_detector` / `face_embedder` session，保持
    kernel-handle 打开、模型大小绑定、ORT 1.28 版本锁定和 context termination；该边界在 Linux/arm64 对
    精确 YuNet/AuraFace 候选完成原生编译、12-output detector 与 embedding 图合同校验、detector 有界
    decode/NMS 及 512 维有限非零输出。随后以冻结 libvips 和公开许可 JPEG 运行完整 decode、方向/尺寸约束、
    BGR detector tensor、五点对齐、AuraFace tensor 与 embedding，产生至少一个有限非零 candidate；没有持久化
    crop/向量/路径。最终模型包激活 owner、native amd64、正式质量/合规与 production composition 仍缺，故本项
    仍未勾选。记录见
    [人脸检测与嵌入原生 session 边界](../changes/FIX-2026-09-01-face-embedding-session-boundary.md)。
  - 2026-09-01：同一 production boundary 又在 Docker Desktop 的 foreign-architecture Linux/amd64 目标完成
    固定源码 libvips 构建、x64 ORT 图/ABI 和完整 pipeline 预检；模型、输入和 rootfs 只读且运行期断网。
    同一公开输入在 arm64 与模拟 amd64 均产生 3 个相同 box；逐值比较的 embedding 最大绝对差
    `1.2406e-4`、无分量超过 `5e-4`、三组 cosine 均至少 `0.999999999449`。0.001 量化结果哈希并不相同，
    因此不声称 bitwise/quantized-bin identity。该运行宿主仍是 arm64，只证明 amd64 compile/functional
    compatibility 与数值漂移预检，不满足 native amd64 Gate，也不改变本项状态。证据见
    [emulated Linux/amd64 preflight](../evidence/int-001/auraface-production-boundary-emulated-amd64-2026-09-01.md)。
  - 2026-09-02：把上述候选 pipeline 固化进原生双架构 workflow：按 runner 架构锁定 ORT archive SHA，
    固定 YuNet/AuraFace/公开 fixture SHA，拒绝 machine/Go/Docker arch 不一致，并在断网只读有界容器执行。
    paired verifier 还严格校验每侧 candidate record、相同输入和显式非批准 flags。后续 run
    `33616238888` 已在同一 source SHA 原生运行并通过该 candidate preflight，但候选仍明确缺质量/合规，
    因此 `INT-241` 不勾选。记录见
    [人脸候选原生工作流预检](../changes/FIX-2026-09-02-face-native-workflow-preflight.md)。
  - 2026-09-01：确认现有 model catalog/activation 只拥有 semantic singleton，不能用测试 seed 或复用
    semantic commit 创建 face generation。新增
    [ADR-0015](../adr/0015-face-model-package-and-generation-activation.md)，冻结“审核组合包 + 独立 face
    generation commit + 各库完整重建后切 cluster”的方向。ADR 已接受 fail-closed 后端实现；具体模型、
    内建 catalog 与 application composition 仍等待模型/隐私/质量/双架构和 owner Gate。
  - 2026-09-01：为 ADR-0015 增加隔离的 face format v3 可执行合同：组合包必须同时绑定唯一 detector、
    embedder 与 governed threshold-profile 文件、七项 transform contract 和逐组件 license ID；semantic
    version/purpose 混淆、缺失/重复 role、未知/重复/trailing JSON 和路径/容量攻击均拒绝。`make spike-ai`
    已通过；架构 fitness 同时阻止 `cmd/`、`internal/` 导入该 spike parser，并锁定 Release No-Go。
  - 2026-09-01：production `internal/aimodel` 已独立实现 face format v3、严格 threshold-profile parser、
    purpose-aware activation worker 和 SQLite migration 35。face commit 只退休旧 face generation，不修改
    semantic active pointer，也不提前切换各库旧 cluster；三项文件/ABI smoke 任一失败均保留旧可靠代次。
    API 暴露独立 nullable `activeFaceModelId`，供应链 evidence schema v2 要求 detector/embedder 两个唯一角色。
    授权 `coser` 最大有界运行对 2,347 张图片全部解码并产生 1,515 个有限 512 维候选，且实际暴露并验证了
    bridge-face 防护。后端实现和功能矩阵已完成，故本项勾选；正式 50×20/偏差质量与发布签署仍归
    `INT-250/251`，内建 catalog/runtime composition 继续为空。记录见
    [组合模型包与独立激活](../changes/FIX-2026-09-01-face-composite-model-activation.md)。
- [x] `INT-242` 实现 observations/embeddings 的幂等 repository、失效、删除和库级隔离。
  - 2026-08-31：SQLite 原子替换、零脸完成 marker、source 变化失效、generation/library/asset 复合约束及
    manual lineage 保留均有 repository 与 migration 回归。
  - 2026-09-01：observation 首次写入、内容更新及 source-change anchor 失效在 wall-clock 回拨时均不早于
    各自创建时间；明确生命周期回归连续 20 次通过。
- [x] `INT-243` 实现匿名 cluster generation、core/edge 角色、增量更新和全量重聚类。
  - 2026-08-31：确定性 exact/LSH 有界候选、cannot-link 优先、staged build 原子激活及旧 build 有界删除已实现；
    后续 512 维 100k 回归修复 LSH 桶边界漏比较，并在受限 Linux/arm64 容器完成；仍不替代最终原生
    双架构联合容量。
  - 2026-09-01：staged build 激活发布在 wall-clock 回拨时保持 build 与 library settings 时间约束，明确
    回归连续 20 次通过。
- [x] `INT-244` 实现 people、重名、revision 和空人物规则。
  - 2026-08-31：NFC 1～100 code point 名称、允许重名、revision CAS、空人物和稳定 keyset 列表已验证。
  - 2026-09-01：rename/delete revision 路径在 wall-clock 回拨时保持 updated/tombstoned 不早于创建时间；
    完整人物生命周期回归连续 20 次通过。
- [x] `INT-245` 实现从 cluster 建人物、cluster→person、single face→person、move/remove/exclude。
  - 2026-08-31：person mutation service 以 idempotency key 派生 opaque person/anchor ID；空人物与 core cluster
    创建、cluster/face assignment、move/split/exclude、rename/delete 均使用 revision/短事务并有 HTTP adapter
    和 replay/原子失败集成回归。超过 100 个 core face 的整组动作失败关闭。
  - 2026-09-01：assignment/split/undo 在持续 wall-clock 回拨下保持人物与 anchor revision/时间约束，明确
    端到端回归连续 20 次通过。
- [x] `INT-246` 实现 person→person 合并事务、audit/alias、冲突、取消和幂等。
  - 2026-08-31：短事务 merge、cannot-link 全量冲突失败、alias/audit、request hash replay、guarded undo 与
    fail-closed HTTP mutation adapter 已验证；不存在部分迁移。
- [x] `INT-247` 实现 manual assignment/cannot-link 优先级并验证模型升级后不覆盖。
  - 2026-08-31：assignment/exclusion/cannot-link、generation reconciliation、needs_review、模型重聚类过滤和
    revision-guarded undo 已由 capability/SQLite 集成测试固定。
  - 2026-09-01：换代 reconciliation 与 guarded undo 的既有 anchor/exclusion 更新统一钳制，持续回拨回归
    保持人工关系与审计事务原子。
- [x] `INT-248` 实现 people/assets、cluster/detail cursor 和多人图片 face DTO；不暴露 embedding/crop path。
  - 2026-08-31：补全权威 OpenAPI、生成客户端、active-build/person-revision 绑定 keyset cursor、asset 多 face
    粗略整数百分比区域及 HTTP wire 泄露回归；详见
    [安全读取投影合同补全](../changes/FIX-2026-09-01-face-safe-read-projections.md)。
  - 2026-09-02：人物跨库资产 cursor 升级为同时绑定 person revision 与所有 bound library 状态 revision；
    任一来源 offline/not-ready 统一失败关闭为 `face_not_ready`，不会把静默过滤后的不完整页解释为空人物，
    且列表后复核覆盖查询期间的状态竞争。catalog hydration 还绑定 library/asset/顺序/availability/kind，
    删除、短页、乱序或跨库替换按 stale 失败，不再落成 500 或错误资产。
- [x] `INT-249` 实现库级 AI 清除、库删除、备份恢复和孤立人物处理；证明原媒体 hash/mtime 不变。
  - 2026-08-31：derived/manual clear 分离、lease/cancel/recovery、真实 library removal 级联、SQLite 一致备份
    恢复、空人物保留及合成 sentinel SHA-256/mtime 不变均有集成回归。
  - 2026-09-01：语义与人脸清除在设置行尚不存在且 wall clock 早于 library 创建时间时，现在均把设置
    `updated_at_ms` 钳制到 `created_at_ms`；claim/refresh 还保证完整 lease，批次、取消与终态逐行钳制
    updated/finished time。四条持续回拨的 admission/lifecycle 回归各连续 20 次通过，清除请求和运行任务
    不再因时间约束误失败。记录见
    [AI 清除设置时钟回拨收敛](../changes/FIX-2026-09-01-ai-clear-settings-clock-rollback.md)。
- [x] `INT-250` 跑授权功能质量、错误合并、源变化、offline、崩溃和 100k×512 人脸容量矩阵；最终
  50×20 identity、五维偏差与 99.5% core precision 由 `INT-403/406/411` 验收。
  - 2026-09-01：face analysis/clear 的 admission、claim/refresh、进度/批次、失败或成功终态及 queued/running
    cancel 已在持续 wall-clock 回拨下验证，lease 不缩水且 job/operation/settings 时间约束保持；三条回归各
    连续 20 次通过。该结果不替代真实质量/偏差、原生 amd64 或最终联合容量，主项不勾选。
  - 2026-09-01：把原 32 维 100k 聚类基线升级到候选所需 512 维时，首次运行在 10 分钟超时并定位到
    LSH 桶边界漏比较：桶首 core 成员被降为 edge 后触发逐成员 × 逐簇退化。修复为在有界窗口内跳过
    前一 signature 后扫描当前桶，并增加 4,098 成员双桶回归。进一步把大集合 edge 限定到同一 LSH
    邻域且只允许附着 core，关闭全簇平方扫描和 edge→edge 误挂；100k × 512 现在连续覆盖 50,000 个
    paired core 与 100,000 个全单例极端。Darwin/arm64 合计 7.194 秒，Linux/arm64 4 CPU/4 GiB 只读
    断网容器合计 7.157 秒、Go memory sys 约 406 MiB，结果指纹一致。
    原生 workflow 和 paired verifier 已强制每侧运行/校验 `face-capacity.json`，但尚无远端 native amd64
    配对，且该合成容量不含真实质量、数据库/浏览并发或最终模型联合负载，故本项仍不勾选。记录见
    [LSH 容量修复](../changes/FIX-2026-09-01-face-lsh-bucket-boundary-capacity.md)与
    [Linux/arm64 100k × 512 证据](../evidence/int-001/face-clustering-100k-512-linux-arm64-2026-09-01.md)。
  - 2026-09-01：补齐 runtime-unavailable 失败矩阵：首次 analyzer 调用返回模型不可用后，任务立即失败，
    progress 与 checkpoint 均不前移、cluster build 为零、settings 转 `awaiting_model`；普通媒体失败语义不变。
    最终模型联合容量、真实质量/偏差及 Linux 双架构仍缺，因此主项保持未勾选。
  - 2026-08-31：按用户授权对本机九个顶层目录组做只读扩大验证：725 图全部解码，457 个有限 embedding；
    13,632 个组内 pair 与 90,564 个跨组 pair 的阈值扫描在 0.7 仍出现 39 个跨组接受，而 0.8 为零但
    组内接受率仅 7.31%。目录包含多个人且不是 identity ground truth。结果只保留聚合值，不持久化路径、
    姓名、crop 或 embedding；它关闭本机
    “错误合并可被测量且高阈值明显牺牲 recall”的功能子项，不是 50 identity × 20 图合法 ground truth，
    也没有五维偏差标注。证据见
    [local face group-pair functional evidence](../evidence/int-001/face-group-pair-functional-local-arm64-2026-08-31.md)。
  - 2026-08-31：补齐真实进程强杀纵向：子进程通过 production SQLite store claim 后由父进程杀死，
    lease 到期后新进程精确重排一次；checkpoint `101`、completed `1/2` 保持不变，attempt `1→2`、
    claimed revision `2→3` 排除旧 worker 提交。证据见
    [face analysis process-kill recovery](../evidence/int-001/face-analysis-process-kill-recovery-darwin-arm64-2026-08-31.md)。
    source-change、offline、retry exhaustion、100k bounded clustering 已有定向证据；真实质量/偏差与
    Linux 双架构联合容量仍缺，因此主项保持未勾选。
  - 2026-09-02：修复领取后 library 转 offline 仍逐资产 admission 的缺口。worker 现在每个候选前先续租并
    让 cancellation 优先收敛，再复核库状态；offline-before-process 零次调用 runtime，处理中 offline 只保留
    已通过 source fingerprint CAS 的首个安全批次，随后以稳定 `library_offline` 终止。旧 observation/result、
    人物与人工约束不删除，不把未处理资产计作普通失败；offline 新任务也在持久化前以独立 HTTP 错误拒绝，
    不再误报模型不可用。记录见
    [人脸任务 offline admission 收敛](../changes/FIX-2026-09-02-face-offline-admission.md)。
  - 2026-09-02：staged cluster build 在激活短事务内再次绑定 ready/enabled/active generation 及当前
    running job claim；offline、禁用、取消、换代或旧 worker 竞争删除未激活 build 并保留旧 active build。
    长聚类结束后 worker 再复核取消，禁用/取消竞争不会误报 succeeded。任务接纳与 cluster 激活还只允许
    `building|ready|degraded` 设置状态，拒绝
    `enabled=1 + awaiting_model` 等非可运行组合且不落 job/operation。job 终态与 recovery 只在 settings
    仍为该任务拥有的 `building` 时写 ready/degraded，不覆盖并发产生的 awaiting-model 等较新状态；错误
    claim revision 在创建 cluster build 前即被拒绝。每候选复核还绑定 requested generation 和 settings；
    运行中转 awaiting-model 或 generation 失效时以 `model_unavailable` 停止，不继续 analyzer。
  - 2026-08-31：授权媒体的二级目录扩大抽样处理 180 个目录、3,070 张文件，全部解码并产生 1,996 个
    有限 embedding；修正 prefix 截断偏差后，12,848 个目录内 pair 与覆盖全部有效目录组合的 100,000 个
    跨目录 pair 在阈值 0.7 有 541 个跨目录候选，阈值 0.8 仍有 6 个且目录内 recall 降至 19.54%。目录只作
    内存中的功能分组，成员未经过 face-level 审核且缺少五维偏差标注，因此仍不是正式 ground truth。证据见
    [nested-directory functional evidence](../evidence/int-001/face-nested-group-functional-local-arm64-2026-08-31.md)。
  - 2026-09-01：AuraFace 候选在相同有界 9 组/135 图功能输入上得到 79/79 有限 embedding；目录 pair
    阈值 0.6 时跨组接受为零但组内接受率为 60.93%。目录并非逐脸身份标注，故该结果仅关闭候选运行与
    错误合并可测性，不替代 governed 质量/偏差或 99.5% core precision Gate。
  - 2026-09-01：新增只读私有复核准备工具。首轮 496 张/309 候选的单链结果曾表面形成八个大簇，但扩大到
    1,547 张/986 候选后，0.6 单链阈值经 bridge face 把八个来源串成 834-member 巨簇，故撤回首轮的稳定
    身份解释。production 与工具现统一采用 smallest-ID anchor coherence；重算后扩大样本为 316 簇、9 个
    `≥20` pending candidate，巨簇消失且 100k×512 容量保持。目录仍不是 identity ground truth，不能伪造
    正式 `50×20` 或 bias slice；私有缩略图、向量、路径和 CSV 仅在 `/tmp`，production 继续固定
    `groupAssignmentAllowed=false`。见
    [coser 私有复核材料准备](../changes/FIX-2026-09-01-coser-private-face-review-preparation.md)与
    [core bridge 防护](../changes/FIX-2026-09-01-face-core-bridge-prevention.md)。
  - 2026-09-01：继续把授权 `coser` 每来源上限提高到 500，2,347 张全部解码，1,496 张产生 1,515 个
    唯一、有限的 512 维候选。0.60～0.80 阈值扫描的 `≥20` 候选簇依次从 17 降到 0；0.70 仅剩两个，
    且人工拼图复核仍发现其中一个含可疑异人。因此 `coser` 已纳入功能、阈值退化和错误合并证据，但
    不能靠调参生成正式 50×20 身份集或五维偏差标签；`INT-250` 与 Release Gate 保持未勾选。
  - 2026-09-01：production package 的完整候选 pipeline 已在 native Linux/arm64 加载精确 detector/embedder，
    并对一张固定公开 JPEG 运行 decode/detect/align/embed；该单图 smoke 不替代 native amd64、真实质量/偏差
    或最终联合容量，因此主项保持未勾选。
  - 2026-09-01：授权 `coser` 2,347 图/1,515 face 的 pipeline、阈值退化与人工错误合并抽查，结合
    source/offline/cancel/kill/retry exhaustion、100k×512 deterministic capacity 和 production candidate
    Linux/arm64 pipeline，完成 S2 功能矩阵。formal biometric quality/bias 和 paired final image 按
    CR-2026-022 归 S4，production `groupAssignmentAllowed=false` 且 face route/runtime 继续缺席。
- [x] `INT-251` 安全/隐私复审并签署 S2C **Backend Ready / Release No-Go**。
  - 2026-09-02：原生 identity/outcomes/final-model evidence、供应链 manifest 与最终四份 summary 已统一按
    不可信输入处理：16 MiB 上限、non-symlink regular-file identity 复核，并拒绝 duplicate key、unknown
    field 与 trailing JSON；聚合器继续绑定原始字节 hash、同 commit/model package 和双架构最终镜像。
    quality/face-quality governed input 的 strict decoder 也递归拒绝 duplicate key，并以 256 MiB 上限、
    non-symlink regular-file 和 open 前后 identity 复核约束逐项结果输入。该收口防止证据歧义，但
    privacy/compliance/security/release owner 的真实签署仍未到位，因此本项保持未勾选。
  - 2026-09-01：S2C 后端复审确认默认关闭、无身份推断、原媒体只读、派生/人物状态分离、清除/备份、
    offline/取消/崩溃保留最后可靠状态、API/日志/诊断不暴露 crop/vector/path/query；授权 `coser` 私有输出
    不进入 Git/CI。签署范围仅为 Backend Ready，最终 privacy/compliance/security/release owner 批准继续由
    `INT-406/411` 持有，故发布仍 No-Go。

## S3：消费者与 UI（Consumer/UI Ready / Release No-Go）

- [x] `INT-301` 管理设置：按库开关、模型/空间、覆盖率、失败、重建和清除确认；模型获取覆盖固定
  `/models` 扫描、兼容/拒绝状态、托管/直接选择、真实 operation 进度和 unavailable 恢复。
  - 2026-09-02：新增真实后端驱动的“智能功能 → 媒体库 / 模型 / 任务”管理界面：按库 ETag 开关与
    offline fail-closed、覆盖率/失败/generation、补齐缺失、全量重建和独立清除确认；模型页支持固定
    `/models` 扫描、兼容/拒绝、managed/direct 风险确认、installed/active/unavailable 和激活；任务页
    只凭服务端 opaque operation 查询、条件轮询及取消，不复制任务状态机。简中/英文、390px 响应式、
    reduced-motion、真实 CSRF/ETag/idempotency adapter 与 3 条交互回归已落地；全量前端 170 tests、类型和
    architecture check 通过。在线审核源与下载不在 accepted revision 1；
    [CR-2026-023](../changes/CR-2026-023-s3-offline-model-ui-contract-alignment.md) 已把清单验收与 ADR-0013、
    `api/openapi.yaml`、offline package contract 对齐，禁止以 mock 或假按钮补齐。
- [x] `INT-302` 搜索模式：文件名/画面语义明确切换，URL/query key/cursor 唯一 owner。
  - 2026-09-02：搜索 URL 的 canonical owner 新增 `mode=semantic`，统一归一化语义不兼容的 kind/date/sort，
    TanStack query key 显式绑定 mode/scope/directory/recursive/query，切换模式或筛选自然丢弃旧 cursor。
    画面语义只调用生成客户端边界的 `/api/v1/semantic/assets`，不发送文件名排序或日期参数；页面复用
    同一搜索入口并把图片类型固定为只读，中文/英文错误明确只影响语义搜索。URL、adapter 与页面交互
    25 条聚焦回归通过，类型及 frontend architecture check 通过。
- [x] `INT-303` 语义结果复用虚拟媒体集合、预览/查看器、视频命中帧和索引不完整提示。
  - 2026-09-02：图片/视频语义结果共用 `MediaCollection`/preview/viewer；视频 adapter 保留命中时间与
    4/10 帧计划，覆盖不完整和 stale cursor 有独立呈现，聚焦搜索回归通过。
- [x] `INT-304` AI 标签审核：建议/置信度/接受/忽略，人工标签和 AI 状态视觉/语义分离。
  - 2026-09-02：待审/接受/忽略进入 URL；卡片显式标记 AI 建议和置信度；最多 100 项批量提交绑定
    suggestion/curation 双 revision、CSRF/幂等键，冲突刷新。
- [x] `INT-305` 人物列表：已命名人物、匿名组、搜索、空/加载/失败/offline 状态。
  - 2026-09-02：人物搜索和匿名 core/edge 双入口消费真实 keyset read model，复用 Loading/Error/Empty/
    InlineStatus；人脸未就绪或 offline 不解释为空。
- [x] `INT-306` 匿名组详情：core/edge、批量选择、排除、建人物、并入人物。
  - 2026-09-02：组详情支持有界多选、逐脸批量排除/归类及 core 建人物；`groupAssignmentAllowed=false`
    显式禁用整组自动归类，保持质量 Gate 失败关闭。
- [x] `INT-307` 单图多人 face 选择和归类；键盘/触摸可准确选中框并有可访问替代列表。
  - 2026-09-02：粗略百分比区域只作选择，不宣称精确框；同 DOM 顺序的语义按钮列表提供完整键盘替代，
    两条路径共享选中状态和 revision 写入。
- [x] `INT-308` 人物详情：资产、错误成员移动/移除、合并、重名消歧和 revision conflict。
  - 2026-09-02：详情消费 person/assets 与 asset/faces，提供重命名、移动、split 移除和 merge；重名附
    opaque ID 后缀，412 冲突刷新后要求重新确认。
- [x] `INT-309` 清除/隐私说明禁止暗示真实身份识别；完成简中/英文文案评审。
  - 2026-09-02：简中/英文均明确“不识别现实身份、不联网查人、不训练/发布模型”；derived/manual
    clear 分离，后者需精确计数，两个动作都说明原始媒体不变。
- [x] `INT-310` Storybook 状态、单元/交互、axe、URL 恢复、四断点、主题/locale 和 reduced motion。
  - 2026-09-02：新增智能审核 Storybook 状态；50 个测试文件 181 tests、Storybook build、生成/架构/
    visual-reference check 通过。真实 Chromium 在 390×844、768×1024、1265×800、1440×900 交叉
    zh-CN/en、light/dark、reduced-motion 运行 WCAG A/AA axe，严重/致命问题为 0 且无水平溢出。
- [x] `INT-311` 复审 INT-S3 Consumer/UI Ready。
  - 2026-09-02：[INT-S3 Gate](../gates/POST-MVP-5/int-s3-consumer-ui-ready.md) 签署
    **Consumer/UI Ready / Release No-Go**；S4 的最终模型、质量、native 双架构、联合容量、供应链和
    owner 批准不因 UI 完成而降低。

## S4：纵向、容量与发布

- [ ] `INT-401` 真实容器纵向：启用→backfill→搜索/建人物→重启→升级→回滚配对→清除。
  - 根据 CR-2026-022，本项同时持有最终审核 semantic/face package 的完整 app inference 强杀、来源损坏、
    lease recovery 与旧 generation/原媒体不变纵向；S2 synthetic/candidate 证据不能替代。
- [ ] `INT-402` 100k/10k 最终镜像容量与并发浏览；验证禁用时零模型常驻和零后台 admission。
  - 2026-08-31：darwin/arm64 强制基线实际失败，production `searchKeysetP95Us=296212` 超过
    250,000 µs；不得登记为通过。benchmark-only order-first 候选在两库 10k/100k 的 11×2 scope/filter/
    sort 页面及 0/1/10/100/>100 cardinality 页面与 production 逐 ID 等价，最慢候选页 49,073 µs。
    后续同档再以 22 个完整 hydrated 页面验证 asset/library/grid/storyboard/favorite 行，并在 FTS rebuild、
    integrity-check 和数据库连接重开后复核代表性全局页；均保持等价，最慢 candidate ID 页 47,924 µs。
    再加入中文两字、组合字符、sharp-s、标点、多词 AND 和带引号 fallback 后，17 个首页、11 个第二页、
    28 个完整 hydrated 页面仍通过，最慢 candidate ID 页 67,374 µs。该证据推进查询计划补充 Gate，
    随后完整重扫并发期间 10 次 hydrated 对照和扫描发布后复核也通过，最慢 candidate ID 页
    67,471 µs。2026-09-01 已把该策略纳入 production repository，旧 cursor/完整矩阵继续通过；强制
    `make spike-capacity` 得到 production keyset P95 `133,637 µs`、零预算违规。本地查询容量子项已关闭，
    但当时最终模型联合负载、最终镜像和 native 双架构仍缺，所以 `INT-402` 未完成。
  - 2026-09-02：同一 source `5af4da0…` 的真实原生 linux/amd64 与 linux/arm64 workflow 均通过 4 CPU、
    10k 目录/100k 资产 baseline；扫描为 `51,738 / 45,441 ms`，production keyset P95 为
    `182,299 / 247,994 µs`，RSS 为 `60,002,304 / 48,594,944 bytes`，两端均零预算违规，paired verifier
    通过。该结果关闭基础 catalog/native 容量子证据；最终模型联合负载和 final image digest 仍缺，主项
    保持未勾选。见 [paired native baseline evidence](../evidence/int-001/int-s4-native-baseline-linux-amd64-arm64-2026-09-02.md)。
- [ ] `INT-403` 原生 amd64/arm64 最终 digest 的模型质量、RSS、索引重建和数值容差。
  - 根据 CR-2026-022，本项持有 governed semantic/tag/video/face 最终结果、50×20 face 与五维偏差、
    tag precision/video Top-20、99.5% core precision，以及同一 final package/image digest 的双架构结果。
  - 2026-08-31：新增手动
    [Intelligent media native evidence](../../.github/workflows/intelligent-media-native.yml) 入口，分别锁定
    GitHub-hosted `ubuntu-24.04` x64 与 `ubuntu-24.04-arm` ARM64 runner，拒绝 QEMU/平台覆盖，并在两端
    执行仓库检查、production libvips、两库查询矩阵和强制 10k/100k 容量基线。该入口建立时尚未在目标
    source SHA 上实际运行；后续成功执行见下方记录。只有同一 source SHA 的两份成功 artifact 经 Gate
    复核后才可计入。
  - 同日补齐 `make verify-intelligent-media-native-evidence` 与 workflow paired job：严格拒绝单架构、跨
    commit/run/attempt、QEMU/错误 runner identity 以及 repository/libvips/search/capacity 任一步骤失败。
  - 后续新增 `make verify-intelligent-media-native-model-evidence` 严格模式，要求两架构额外提供最终 model
    package、质量集/ranking/tie fixture hash、Top-20 集合、`1e-3` 数值容差、3.2 GiB RSS、查询/浏览/
    派生空间预算、index rebuild/restart 与 runtime failure matrix。当前没有获准最终模型生成的真实文件，
    baseline workflow 也不会伪造它们，所以入口完成仍不改变 `INT-403` 状态。
  - 严格模式随后改为实际读取并复算 quality summary、Top-20、numeric、runtime、index 五份报告的
    SHA-256，拒绝路径逃逸、symlink、非普通文件和篡改；审核数据本体仍不上传，只绑定 governed hash。
  - 2026-09-02：run `33616238888` 首次在同一 source `5af4da0…` 获得原生 x86_64/aarch64 两份 complete
    baseline artifact，且 paired verifier 通过 native identity、同 commit/run/attempt、QEMU 拒绝、仓库、
    libvips、候选 pipeline、100k×512 synthetic face、搜索矩阵和容量检查。summary 明确
    `finalModelEvidence=false`，候选也明确 `productionApproved/qualityGate/complianceGate=false`，所以只关闭
    baseline 缺口，不满足 final model/image/quality/numeric 条件，`INT-403` 保持未勾选。证据同上。
  - 2026-09-01 只读复核确认本地 HEAD 与 `origin/aifeature` 均为 `fdede8c…`，远端基线提交已含智能媒体
    baseline workflow；但当前 S2 增量仍有 96 个 modified/untracked path，不属于该 SHA，最近运行也没有
    paired artifact。baseline workflow 不生成严格模式需要的最终 `model-evidence.json`。证据见
    [远端就绪审计](../evidence/int-001/native-remote-readiness-audit-2026-08-31.md)；未经明确授权不提交、不推送、
    不触发任务，`INT-403` 保持未完成。
- [ ] `INT-404` 最终 SBOM/provenance/license/notices/vulnerability 及模型权重清单签署。
  - `make verify-intelligent-media-supply-chain` 已建立失败关闭的真实文件/哈希/批准绑定入口；只有最终
    source SHA 的实际 manifest 成功通过并经 Gate 人工复审后才能勾选，本地 verifier 测试不能替代证据。
  - `make verify-intelligent-media-s2-evidence` 进一步要求 quality、strict native、supply-chain 三份已验证
    summary 使用同一 source commit/model package，并逐架构匹配 final image digest；baseline native
    summary 会失败。当前三份真实 summary 均不存在，交叉绑定工具完成不等于 `INT-404` 完成。
- [x] `INT-405` 备份/恢复人物应用状态；索引缺失自动重建；损坏模型/DB/磁盘满失败关闭。
  - 2026-09-02：SQLite 一致性备份/重开保留 person、anchor 与人工关系并通过 integrity check；catalog
    FTS 缺失/不一致由数据库打开自动修复，missing/stale semantic 派生行由有界 `missing` backfill 重算，
    且进程强杀后从 durable checkpoint 恢复。真实 linux/arm64
    候选容器又通过离线备份/恢复、SIGKILL/WAL 恢复、数据卷写满与损坏数据库启动失败；managed model
    的实际 ENOSPC、损坏/变化来源和 unavailable 路径均失败关闭。该任务不包含最终模型质量或双架构
    发行签署，后者仍由 `INT-401～404/410/411` 持有。
- [ ] `INT-406` 人脸/向量/查询/姓名不进入日志和诊断，API/缓存/删除后残留检查通过。
  - 本项还持有最终 privacy/compliance/security 发布复审；S2C Backend Ready 签署不替代 release approval。
  - 2026-09-02：新增 `make test-intelligent-media-privacy`，对 face/person/name/vector/score/bbox/crop 等日志
    属性执行 canary 脱敏，复核 face/semantic/diagnostic 安全 DTO，并把 `secure_delete=ON` 固化到每个 SQLite
    连接；文件级 canary 在删除和 WAL truncate 后不再出现在活动 DB/WAL/SHM，derived/manual clear 继续保持
    分类数据边界和原媒体 hash/mtime。该结果关闭工程子项，但不替代 privacy/compliance/security owner
    发布批准，故本项仍不勾选。证据见
    [S4 隐私工程证据](../evidence/int-001/int-s4-privacy-engineering-darwin-arm64-2026-09-02.md)。
- [x] `INT-407` 原媒体只读 hash/mtime/mount 证据和路径攻击矩阵通过。
  - 2026-09-02：真实 linux/arm64 候选以只读 rootfs、`/library:ro`、capabilities 全丢弃运行，媒体矩阵、
    Compose、可信代理与恢复演练前后 sentinel SHA-256 一致；face clear 回归同时比较 source
    fingerprint/size/mtime。`internal/files`、`internal/pathpolicy` 和 integration 全包测试覆盖 traversal、
    duplicate encoding、NUL、symlink/hardlink、跨设备/nested mount、目录替换竞态和 poisoned catalog path。
- [ ] `INT-408` 三浏览器、物理输入、移动断点、键盘和大集合虚拟化证据通过。
  - 2026-09-02：锁定 Playwright 1.61.1 的 Chromium、Firefox 151、WebKit 26.5 已通过桌面键盘/焦点/
    降级、200% 等效重排和 storyboard 行为；100k 虚拟集合三引擎均为 60 个 mounted item、超过 59 FPS、
    P95 不高于 21 ms 且低于 1.5 GiB RSS。原始 JSON 与边界见
    [darwin/arm64 browser automation evidence](../evidence/int-001/int-s4-browser-automation-darwin-arm64-2026-09-02.md)。
    Playwright WebKit 不冒充 retail Safari，模拟 viewport/touch 不冒充物理触控，故不勾选。
  - 同日又在 macOS 26.6.2 的 retail Safari 26.6.2 上，以临时只读 fixture 库验证完整键盘顺序、预览、
    完整查看器、`I` 信息、`Escape` 焦点恢复和真实 200% 页面缩放；媒体 hash 前后相同。证据见
    [retail Safari evidence](../evidence/int-001/int-s4-retail-safari-darwin-arm64-2026-09-02.md)。
  - 同日产品用户明确 `INT-S4` 不需要 VoiceOver；
    [CR-2026-024](../changes/CR-2026-024-int-s4-screen-reader-waiver.md)只删除真实屏幕阅读器人工验收，不把
    未执行测试登记为通过，也不删除语义/键盘/axe 门槛。现仅剩物理触控与目标移动设备证据，仍不勾选。
  - 实体设备只读预检发现已配对的 iPhone 17 Pro（iOS 26.5）当前为 `unavailable`，developer tunnel
    不可用、DDI service 不可用且 USB 未连接；未安装应用或读取设备数据。见
    [physical-device readiness evidence](../evidence/int-001/int-s4-physical-device-readiness-darwin-arm64-2026-09-02.md)。
    模拟器和 touch emulation 不替代真机，故本项继续未勾选。
- [x] `INT-409` 更新用户 README、部署、升级、隐私、限制、故障排除和模型来源文档。
  - 2026-09-02：中英文 README 已从历史 revision 1 “planned”更新为 revision 2 “已实现、发行 No-Go”，
    明确匿名分组不等于现实身份识别；部署文档集中说明 reviewed catalog 精确来源、离线 `/models:ro`、
    升级/配对回滚、人物状态恢复、容量限制、稳定错误排查、隐私诊断字段和两类清除。当前没有最终获准
    模型包或在线/国内镜像，文档不向用户展示虚构下载路径。
- [ ] `INT-410` 在无外网、发行源、真实部署镜像和 `/models:ro` 四种拓扑验证安装/升级/恢复；没有
  真实镜像证据时从发布范围和 UI 删除镜像选项。
  - 2026-09-02：新增 `make test-intelligent-media-offline`，本机原生 linux/arm64 候选以
    `--network none`、只读 rootfs、`/library:ro`、`/models:ro` 完成管理员初始化、空 reviewed catalog/
    candidate scan、重启和双 sentinel hash 不变，证明无模型时核心服务可用且不写来源。在线/镜像入口
    已删除；最终发行源、最终模型和同 digest native amd64/arm64 的安装/升级/恢复仍缺，故本项不勾选。
- [ ] `INT-411` 复审 INT-S4 Integrated Slice Done；未签署不得宣称功能发布完成。
  - 必须同时复核五个 intelligent-media final verifier 及 product/ML/QA/privacy/compliance/security/release
    批准引用；任一缺失时 reviewed catalog/face composition/UI release 继续失败关闭。

当前 S4 可审计结论见 [INT-S4 current](../gates/POST-MVP-5/int-s4-current.md)：`3 / 11` 完成，
**Release No-Go**。本机恢复和只读边界通过不替代最终审核模型、governed 质量、native 双架构或角色签署。

## 强制验证入口

实现阶段至少运行并准确记录：

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

此外需增加独立的模型 fixture、AI quality、100k vector、native amd64/arm64、license/SBOM 和最终容器
目标；命令名在 S0/S1 冻结。任何未执行项必须明确报告，不能写“应该通过”。
