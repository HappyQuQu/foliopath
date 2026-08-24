# FTR-INT-001 本地智能媒体发现开发任务清单

## 当前状态

目标版本是尚未冻结的 `POST-MVP-5` 提案。当前 [INT-S0](../gates/POST-MVP-5/int-s0-architecture-ready.md)
为 **No-Go**；只有 `INT-001～023` 的文档、fixture、隔离 spike 和评审工作获准。下面所有 `[ ]`
都表示没有完成证据，不能因已有方案文档改成 `[x]`。

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

## S0：范围、可行性与架构（当前唯一获准阶段）

- [ ] `INT-001` 产品负责人复核 FTR-INT-001 的四项范围和全部非目标。
  - Owner：产品负责人；证据：签署后的 Change Record/Gate。
- [ ] `INT-002` 决定人物库是实例级还是严格库级，冻结跨库合并、库移除和空人物语义。
  - Owner：产品 + face owner；依赖：INT-001。
- [ ] `INT-003` 决定按库启用时 semantic 与 face 能否独立开关，以及默认清除/保留策略。
  - Owner：产品 + 隐私负责人；依赖：INT-001。
- [ ] `INT-004` 建立合法的语义质量数据集 manifest、查询标注和评分脚本。
  - Owner：QA/ML；证据：生成/来源/许可/哈希，不提交受限内容。
- [ ] `INT-005` 建立合法的人脸 ground-truth 数据集 manifest 和聚类评分脚本。
  - Owner：QA/ML；依赖：隐私评审。
- [ ] `INT-006` 比较至少两个视觉-文本候选的中英文质量、吞吐、RSS、模型大小和权重许可。
  - Owner：inference/semantic；依赖：INT-004。
- [ ] `INT-007` 比较人脸 detector/embedding 候选的检测、ROC、聚类质量、RSS 和权重许可。
  - Owner：inference/face；依赖：INT-005。
- [ ] `INT-008` 在原生 linux/amd64 与 arm64 验证候选 runtime 的加载、取消、泄漏、畸形输入和无网络运行。
  - Owner：inference/release；依赖：INT-006、INT-007。
- [ ] `INT-009` 实现 SQLite blob 精确向量基线并跑 10k/50k/100k 查询、过滤和稳定排序。
  - Owner：semantic/store；仅限 spike。
- [ ] `INT-010` 至少比较一种可重建 ANN 方案的 build、查询、recall、RSS、损坏恢复和双架构闭包。
  - Owner：semantic/release；依赖：INT-009。
- [ ] `INT-011` 验证视频复用 4/10 格故事板的质量、frame 聚合和无重复 FFmpeg admission。
  - Owner：semantic/media；依赖：INT-006。
- [ ] `INT-012` 验证核心簇/边缘建议分层，以及 manual assignment/cannot-link 重聚类不被覆盖。
  - Owner：face；依赖：INT-007。
- [ ] `INT-013` 运行 4 CPU/4 GiB 100k 联合容量：backfill + browse/search + restart + cancel。
  - Owner：performance；依赖：INT-008～012。
- [ ] `INT-014` 完成 runtime、模型权重、vector 引擎 SBOM/许可证/漏洞与再分发审查。
  - Owner：合规/安全；依赖：候选收敛。
- [ ] `INT-015` 完成人脸数据告知、访问、清除、备份、诊断和共享场景隐私评审。
  - Owner：安全/隐私。
- [ ] `INT-016` 根据证据冻结或缩减模型、索引、容量和人脸交互；更新 ADR-0013。
  - Owner：技术负责人；依赖：INT-006～015。
- [ ] `INT-017` 为所有开放风险指定 owner、最迟 Gate 和 fallback，更新 R-024～R-030。
- [ ] `INT-018` 创建并批准 `POST-MVP-5` scope manifest 与 scope budget；未批准则保持提案。
  - Owner：产品负责人；依赖：INT-001、INT-016。
- [ ] `INT-019` 冻结模型分发策略：应用镜像内置、签名发行源、项目镜像、部署者镜像和离线目录中
  哪些进入首版；没有真实 owner/基础设施的来源必须删除。
  - Owner：产品 + release + compliance；依赖：INT-014。
- [ ] `INT-020` 隔离验证下载取消/续传/重定向/错哈希/磁盘满，以及 `/models:ro` 扫描、托管复制、
  直接读取、源变化/缺失/可写/symlink/嵌套 mount 的失败关闭和恢复。
  - Owner：aimodel + security + release；仅限 spike。
- [ ] `INT-021` 冻结 `/models` 与 `/app/data/models` 的所有权、备份/恢复、升级、配额、诊断和
  provenance 语义；明确直接模式在模型 unavailable 时的查询行为。
- [ ] `INT-022` 用 INT-019～021 证据复审 ADR-0013；未通过则只保留托管内置模型或整个
  feature 继续 No-Go。
- [ ] `INT-023` 复审 INT-S0；只有全部适用条件有证据才能改为 Go。

## S1：权威合同（等待 S0 Go）

- [ ] `INT-101` 更新产品需求、用户流程、UI 设计和 README，写入冻结的 FR/NFR/非目标。
- [ ] `INT-102` 接受 semantic/face/inference/jobs capability 接口、依赖方向和错误所有者。
- [ ] `INT-103` 更新数据模型，冻结派生/应用状态、FK、revision、retention、backup 和 cascade。
- [ ] `INT-104` 设计只追加 migration；验证新库、旧库升级、重复迁移、回滚配对和失败关闭。
- [ ] `INT-105` 更新安全模型：生物特征、模型供应链、日志脱敏、清除和无网络边界。
- [ ] `INT-106` 更新部署文档：镜像体积、模型位置、磁盘预算、双架构和备份/恢复。
- [ ] `INT-107` 更新测试策略与 fixture 治理，冻结准确率、性能和跨架构误差判定。
- [ ] `INT-108` 在 `api/openapi.yaml` 定义 AI settings/status、semantic search、tag suggestion、clusters、people、
  face assignment/merge/exclude、rebuild/delete，并统一错误与 ETag。
- [ ] `INT-109` 生成客户端并运行语义兼容、确定性生成和合同 fixture。
- [ ] `INT-110` 编写 API/data transaction decision table：partial failure、并发修改、stale face、generation rollover。
- [ ] `INT-111` 冻结模型列表、download/cancel、fixed-directory scan、import/direct、activate/status 合同；
  所有请求只接受 opaque ID，不接受 URL 或路径。
- [ ] `INT-112` 冻结下载源/镜像部署配置、重定向/代理策略、临时空间、模型配额和 `/models:ro` 示例。
- [ ] `INT-113` 复审 INT-S1 Contract Ready。

## S2A：模型管理与图片语义搜索后端

- [ ] `INT-201` 实现 manifest/hash/license/architecture 校验和 model unavailable 状态。
- [ ] `INT-202` 实现 inference port 与已选 adapter，固定线程、超时、取消、延迟加载和资源计数。
- [ ] `INT-203` 实现 semantic model generation、preprocess 和 deterministic embedding contract。
- [ ] `INT-204` 实现幂等 `semantic_image` 任务、keyset admission、公平调度、lease/retry/cancel。
- [ ] `INT-205` 实现 SQLite embedding repository，短事务、source fingerprint 失效和 cascade。
- [ ] `INT-206` 实现 vector index build/temp/atomic activate/checksum/rebuild/old-generation fallback。
- [ ] `INT-207` 实现文本 query、scope/filter、stable cursor、generation conflict 和 coverage 状态。
- [ ] `INT-208` 集成 API/auth/CSRF/限流/错误脱敏，不返回原始向量。
- [ ] `INT-209` 覆盖模型缺失/损坏、索引损坏、offline、取消、重启、源变化和原媒体不变。
- [ ] `INT-210` 运行 100k 查询、并发浏览、backfill、RSS、DB/index 空间与双架构测试。
- [ ] `INT-211` 实现 aimodel service：审核 manifest、安装状态、generation、托管/直接来源唯一 owner。
- [ ] `INT-212` 实现管理员触发的有界下载、取消/续传、临时文件、空间预检、校验和原子发布。
- [ ] `INT-213` 实现固定 `/models` 安全枚举、opaque candidate、托管复制和只读直接来源校验。
- [ ] `INT-214` 实现模型选择/激活、旧 generation fallback、直接来源失效和显式 unavailable 错误。
- [ ] `INT-215` 覆盖 SSRF/重定向、恶意包、错架构/hash、symlink/mount、磁盘满、强杀、恢复和诊断脱敏。
- [ ] `INT-216` 复审 S2A Backend Evidence Ready。

## S2B：标签建议与视频代表帧搜索后端

- [ ] `INT-221` 冻结受控标签词表输入、Top-K、阈值、suggestion 生命周期和 ignore 语义。
- [ ] `INT-222` 实现 tag text embedding/cache 和 suggestion repository；不直接写人工标签表。
- [ ] `INT-223` 接受建议时调用 curation service，并处理 tag revision/precondition/cascade。
- [ ] `INT-224` 实现视频 storyboard generation 依赖和 frame embedding，禁止第二套抽帧。
- [ ] `INT-225` 实现 max/best-frame 排名、video hit DTO 与 source/storyboard version 失效。
- [ ] `INT-226` 覆盖无故事板、部分帧失败、10→4 降级、取消、重复任务和缓存淘汰重建。
- [ ] `INT-227` 用审核集验证 tag precision 和 video Top-20；未达标则缩减/删除对应范围。
- [ ] `INT-228` 复审 S2B Backend Evidence Ready。

## S2C：人脸聚类与人物库后端

- [ ] `INT-241` 实现 face detect/quality/embedding，归一化 box、fingerprint 和输入上限。
- [ ] `INT-242` 实现 observations/embeddings 的幂等 repository、失效、删除和库级隔离。
- [ ] `INT-243` 实现匿名 cluster generation、core/edge 角色、增量更新和全量重聚类。
- [ ] `INT-244` 实现 people、重名、revision 和空人物规则。
- [ ] `INT-245` 实现从 cluster 建人物、cluster→person、single face→person、move/remove/exclude。
- [ ] `INT-246` 实现 person→person 合并事务、audit/alias、冲突、取消和幂等。
- [ ] `INT-247` 实现 manual assignment/cannot-link 优先级并验证模型升级后不覆盖。
- [ ] `INT-248` 实现 people/assets、cluster/detail cursor 和多人图片 face DTO；不暴露 embedding/crop path。
- [ ] `INT-249` 实现库级 AI 清除、库删除、备份恢复和孤立人物处理；证明原媒体 hash/mtime 不变。
- [ ] `INT-250` 跑质量偏差、错误合并、源变化、offline、崩溃和 100k/人脸容量矩阵。
- [ ] `INT-251` 安全/隐私复审并签署 S2C Backend Evidence Ready。

## S3：消费者与 UI（每个页面等待对应 S2）

- [ ] `INT-301` 管理设置：按库开关、模型/空间、覆盖率、失败、重建和清除确认；模型获取覆盖审核源
  下载进度、`/models` 扫描、兼容/拒绝状态、托管/直接选择和 unavailable 恢复。
- [ ] `INT-302` 搜索模式：文件名/画面语义明确切换，URL/query key/cursor 唯一 owner。
- [ ] `INT-303` 语义结果复用虚拟媒体集合、预览/查看器、视频命中帧和索引不完整提示。
- [ ] `INT-304` AI 标签审核：建议/置信度/接受/忽略，人工标签和 AI 状态视觉/语义分离。
- [ ] `INT-305` 人物列表：已命名人物、匿名组、搜索、空/加载/失败/offline 状态。
- [ ] `INT-306` 匿名组详情：core/edge、批量选择、排除、建人物、并入人物。
- [ ] `INT-307` 单图多人 face 选择和归类；键盘/触摸可准确选中框并有可访问替代列表。
- [ ] `INT-308` 人物详情：资产、错误成员移动/移除、合并、重名消歧和 revision conflict。
- [ ] `INT-309` 清除/隐私说明禁止暗示真实身份识别；完成简中/英文文案评审。
- [ ] `INT-310` Storybook 状态、单元/交互、axe、URL 恢复、四断点、主题/locale 和 reduced motion。
- [ ] `INT-311` 复审 INT-S3 Consumer/UI Ready。

## S4：纵向、容量与发布

- [ ] `INT-401` 真实容器纵向：启用→backfill→搜索/建人物→重启→升级→回滚配对→清除。
- [ ] `INT-402` 100k/10k 最终镜像容量与并发浏览；验证禁用时零模型常驻和零后台 admission。
- [ ] `INT-403` 原生 amd64/arm64 最终 digest 的模型质量、RSS、索引重建和数值容差。
- [ ] `INT-404` 最终 SBOM/provenance/license/notices/vulnerability 及模型权重清单签署。
- [ ] `INT-405` 备份/恢复人物应用状态；索引缺失自动重建；损坏模型/DB/磁盘满失败关闭。
- [ ] `INT-406` 人脸/向量/查询/姓名不进入日志和诊断，API/缓存/删除后残留检查通过。
- [ ] `INT-407` 原媒体只读 hash/mtime/mount 证据和路径攻击矩阵通过。
- [ ] `INT-408` 三浏览器、物理输入、移动断点、键盘、读屏和大集合虚拟化证据通过。
- [ ] `INT-409` 更新用户 README、部署、升级、隐私、限制、故障排除和模型来源文档。
- [ ] `INT-410` 在无外网、发行源、真实部署镜像和 `/models:ro` 四种拓扑验证安装/升级/恢复；没有
  真实镜像证据时从发布范围和 UI 删除镜像选项。
- [ ] `INT-411` 复审 INT-S4 Integrated Slice Done；未签署不得宣称功能发布完成。

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
