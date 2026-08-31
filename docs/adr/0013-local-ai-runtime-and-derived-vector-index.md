# ADR-0013：本地 AI 运行时与可重建向量索引

## 状态

**Accepted for POST-MVP-5 revision 1 A+B（2026-08-27）**。本 ADR 授权 S1 合同设计，不授权跳过
模型质量、双架构、安全供应链、Backend Evidence 或 Release Gate。

## 决策角色

- 决策者：产品负责人、技术负责人、安全/隐私负责人、发布负责人
- 执行者：推理运行时、语义检索、人脸、数据和运维 capability owners
- 被咨询者：前端、QA、许可证合规负责人

## 背景与驱动因素

FTR-INT-001 需要对原图和视频代表帧生成视觉向量，并对图片人脸生成独立 embedding。系统当前是
单进程 Go 模块化单体、SQLite、单容器和 4 CPU/4 GiB 目标档；原媒体只读，应用状态与派生数据位于
`/app/data`。部分部署环境无法稳定访问模型上游，若模型只随超大镜像分发会增加升级成本，若只提供
境外在线下载又会让功能无法启用。选择云 API 会传输敏感媒体/人脸，独立服务会改变部署拓扑，把全部
向量只存外部 ANN 索引又会扩大恢复和备份风险。

## 备选方案

1. 云端模型 API：开发较快，但改变隐私、网络、成本和可用性边界。
2. 独立 AI 容器/worker：隔离更强，但新增部署单元、协议、队列和运维面。
3. Go 进程内本地运行时 + SQLite 权威派生记录 + 可重建索引：保持拓扑，需承担 native 运行时和内存风险。
4. 纯精确扫描、无 ANN：恢复简单，是否满足 10 万媒体和人脸聚类延迟未知。
5. 直接把 ANN 文件作为权威数据：查询方便，但损坏、迁移和备份会成为不可恢复边界。
6. 模型全部打入应用镜像：离线可用，但镜像体积、每次升级流量和多模型许可负担最大。
7. 审核清单下载 + 可选 `/models:ro` 离线来源：兼顾小镜像和受限网络，但新增下载供应链、临时空间、
   可选挂载与直接读取失效语义。

## 决策

revision 1 的 A+B 运行与索引采用方案 3，模型获取采用方案 7 的离线子集：

- 在同一 Go HTTP 进程内，通过 capability-owned 窄接口调用审核后的本地推理运行时；首版 CPU-only。
- 默认不联网、不上传媒体、人脸或查询；不新增容器、worker、数据库或 GPU 运行时。
- 模型由固定 manifest、版本、SHA-256、来源和许可证标识，不接受任意 URL 或模型代码。
- 首个发布切片的 ONNX graph 内嵌全部 initializer，不支持 external-data；模型包仍可包含多个独立、
  逐项签名列单的 graph/tokenizer/声明文件。发行校验解析并拒绝 external-data，运行时只接受固定
  size/hash 的审核图；未来放宽须重新复审安全与格式决策。
- 采用方案 7 的固定只读 `/models` 离线来源；revision 1 不提供项目在线下载源或国内镜像。未来在线
  catalog 只有在真实运营 owner 与签名/轮换/撤回方案成立并新增 scope revision 后才能启用。
- `/app/data/models` 是默认托管位置；离线模型默认校验后复制并原子发布。允许管理员从
  `/models:ro` 直接使用完全匹配的包以节省空间，但必须固定内容哈希，缺失/变化时失败关闭。
- 模型下载和导入由单一 model-management capability 拥有，使用有界并发、临时文件、磁盘安全余量、
  取消/续传和校验后发布；不把模型来源规则复制到 HTTP handler、jobs 或 inference adapter。
- SQLite 保存生成过的 embedding、generation 和人工关系；向量加速索引放在 `/app/data/ai-indexes`，
  始终可从 SQLite 重建，使用临时文件和原子发布。
- 模型代次/索引代次并行构建，验证后切换；不原地破坏最后可靠结果。
- 具体运行时、模型、向量引擎和量化格式延后到 spike 证据完成后在本 ADR 的替代/接受记录中冻结，
  不能把候选名称当成决定。

若进程内运行时不能在 4 GiB 内安全运行，不能静默改成微服务或云端；应缩减 feature 或提出新 ADR。

## S0 候选处置记录（2026-08-27，revision 1 决策）

下表把已有证据转成明确的继续/拒绝边界，避免“都先保留”拖入生产。`Retain for next Gate` 是
revision 1 的唯一候选路径，不等于最终质量、双架构或发布验收。

| 决策面 | 处置 | 依据与剩余阻断 |
| --- | --- | --- |
| 云 API、独立 worker/容器、GPU | Reject revision 1 | 改变隐私/部署/资源边界，用户需求不需要；失败不能静默转云 |
| Go 单进程内 ONNX Runtime C API | Retain for next Gate | arm64 hostile/error/cancel/recovery/race/distroless ABI 通过；native amd64、production adapter/admission、最终 VEX/SBOM/provenance 未过 |
| SigLIP 2 split | Reject current layout | reload peak 3.72 GB、resident 4.01 GB，未留完整进程余量 |
| SigLIP 1 float32 split | Reject current layout | 生产 100k peak 3.590 GB 超 3.2 GiB |
| SigLIP 1 dynamic-QInt8 | Reject | Linux 中文 Recall@1/3 质量崩溃且跨 runtime 仅 8/24 Top-3 一致 |
| SigLIP 1 float16-internal/float32-I/O | Retain as sole semantic resource candidate | 24/24 pilot ranking 保持，三次生产 100k peak 2.860～2.951 GB、ordinary browse +6.0%～14.5%；仍缺 1,000 图/100 视频、native amd64、真实 backfill/full process，storyboard 有一轮 +20.34% |
| SQLite float32 512-d exact | Reject combined layout | 100k DB 410.6 MB，未含视频/人脸/WAL/backup，接近 500 MiB 总预算 |
| SQLite float16 exact | Retain as sole vector storage candidate | synthetic/10 图 pilot ranking、100k search/restart/136.9 MB 支持继续；真实 embedding Recall、双架构和最终全闭包未过 |
| `coder/hnsw` 当前配置 | Reject | 默认 recall 失败，高配置 build 昂贵且导出不确定；exact retained 候选尚在预算内 |
| 其他 ANN/SQLite native extension | Closed unless exact fails | 不提前引入第二 native 供应链；只有 exact 在真实 Gate 失败才以新候选重开 |
| 视频 4/10 帧 mean aggregation | Retain for quality Gate only | 合成代理优于 max-frame 且现有 FFmpeg adapter 可复用；100 个真实视频未评测，不是选择完成 |
| YuNet detector | Retain for face quality Gate | 精确来源/hash/MIT 和 pipeline smoke 已有；真实 detector recall/失败分类未过 |
| SFace embedding | Production hold | 目录声明不足以关闭精确权重训练数据/商业再分发；无合规签署不能分发或让用户下载绕过 |
| 人脸 Slice E | Conditional delete | 无合法真实 ground truth、隐私签署或合格 embedding 时从 Frozen revision 1 删除；不阻断 A+B 单独复审 |
| 模型打入应用镜像 | Reject revision 1 | 大模型升级/撤回与双架构 image 耦合，无必要收益 |
| `/models:ro` + managed/default + strict direct | Retain | arm64 失败矩阵/reconciliation/direct remount 支持；native amd64、S1 contract、quota/backup/source owner 未过 |
| 项目/国内在线镜像 | Closed until operated | 无真实 owner/endpoint/key/SLA 不得承诺；部署者镜像仍受同一签名 catalog 约束 |
| distroless `base-nossl` 最小 C++ 闭包 | Retain for packaging Gate | arm64 restricted run 通过并去掉 7 个 OpenSSL High；ORT arm64 显式 CycloneDX component 已补但未并入最终镜像；仍有 glibc 1 Critical/2 High、amd64 与 signed provenance 缺口 |

现有 catalog 字面搜索 keyset 预算失败已由 query plan 定位为 FTS candidate scan/rowid 回表和临时排序。
benchmark-only order-first 候选在 arm64 100k 下保持两页 ID 顺序，广匹配完整装配 P95 33.668 ms、
稀疏词 110.061 ms 且均无临时排序，但未覆盖完整查询矩阵、取消和 amd64。该债务属于独立 MVP
maintenance Gate；不能通过 AI ADR
修改生产 SQL、schema、API 或放宽阈值，也不能把其成本错误归因给 retained 模型。

revision 1 接受的组合是：Slice A 模型基础 + Slice B 图片语义、ORT C API、SigLIP 1
float16-internal、SQLite float16 exact、CPU-only、`/models:ro` 离线基线。若真实质量、native amd64
或最终安全容量失败，fallback 是删除 Slice B，不改成云或新服务。

## 后果

正面：保留单容器部署和本地隐私；索引损坏可恢复；AI 故障不影响媒体事实；模型升级可回滚。

代价：模型下载和可选映射目录扩大了网络、文件和部署信任边界；项目若承诺国内镜像，就必须真实运营
镜像、签名清单、可用性和撤回机制。直接使用外部模型节省磁盘但降低自包含恢复能力。native runtime
可能使进程 OOM/崩溃；SQLite embedding 会增加备份体积；双架构和模型数值一致性测试成本高；人脸
数据成为高敏感应用数据。

## 验证与复审

本 ADR 已基于 [INT-001](../spikes/int-001-ai-feasibility.md) 的本地探索接受架构方向。以下事项仍必须在
对应 Backend/Release Gate 完成，不能因 ADR Accepted 跳过：

- 模型与权重许可证/再分发结论、哈希、最终镜像 SBOM/notice/provenance，以及裸 native runtime 的
  显式组件记录；
- 原生 amd64/arm64 的加载、推理、取消、畸形输入和内存结果；
- 原生 amd64/arm64 final-stage 的 SONAME/ABI、non-root/read-only/no-network 运行与 Critical/High
  处置；发行版 `no-dsa` 不能代替安全 VEX 审批；
- 10 万媒体 exact/候选 ANN 的构建、查询、损坏恢复、磁盘和升级比较；
- 若未来新增 Slice E，另行完成合法人物套图评测集的聚类 precision、错误合并、隐私和人工约束；
- 模型下载、续传、镜像切换、离线 `/models:ro` 扫描、托管复制、直接读取、错哈希、源消失、磁盘满、
  恶意文件和恢复矩阵；
- [INT-S0](../gates/POST-MVP-5/int-s0-architecture-ready.md) 已对 A+B 为 Go；上述剩余项由后续 Gate 持有。
