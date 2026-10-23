# POST-MVP-5 Scope Manifest — Revision 1（已被 revision 2 替代）

- 状态：**Superseded by [revision 2](POST-MVP-5-scope-r2.md)**
- 日期：2026-08-27
- 产品显示标识：`Post-MVP/5`
- Change Record：[CR-2026-021](../changes/CR-2026-021-intelligent-media-discovery.md)
- Feature：[FTR-INT-001](../features/intelligent-media-discovery.md)
- Gate：[INT-S0 Architecture Ready](../gates/POST-MVP-5/int-s0-architecture-ready.md)
- 需求：`FR-INT-001～003`、`FR-INT-005`、`FR-INT-015～020`、`NFR-INT-001～010`；
  `FR-INT-004` 是未来视频切片需求，不在 revision 1
- 产品 Owner：产品用户
- Capability owner：`internal/aimodel`、`internal/semantic`、capability-owned inference port、
  `internal/app` composition、`internal/api` transport、SQLite adapter（均在 S1 接受合同后建立）
- 预算：A+B 最多 16 单人工程周；超过预算或适用 Gate 失败时停止，不扩大部署拓扑或降低安全门槛

## 已冻结范围

### A：模型与派生任务基础

1. AI 按媒体库启用，默认关闭；禁用时不启动 backfill、不保留模型 session、不影响目录、字面搜索、
   预览或查看器。
2. 首版模型来源只承诺固定 `/models:ro` 离线目录。管理员可以扫描兼容包、查看校验状态、选择版本，
   并选择校验后复制到 `/app/data/models` 或对完全匹配的只读包直接读取。
3. 模型包必须由固定 manifest/version/architecture/size/SHA-256/文件清单约束；不接受任意 URL、宿主
   路径、模型附带代码、脚本或 external-data graph。
4. 模型 generation、安装/激活/不可用状态、覆盖率、失败、取消、重建和清除具有稳定合同；新代验证
   完成前保留最后可靠代。
5. 可重建模型、embedding 和索引位于 `/app/data`；原媒体保持只读，不写 sidecar 或 metadata。
6. 当前不提供项目在线下载源或国内镜像。未来只有真实运营 owner、项目签名 catalog、key/checkpoint
   轮换撤回和可用性方案全部成立后，才能通过新 scope revision 加入。

### B：图片语义搜索

1. 对已索引 JPEG、PNG、WebP 和 GIF/animated 图片生成同代可重建视觉 embedding。
2. 支持中文和英文自然语言查询；语义搜索与现有文件名搜索明确分开，不替代目录层级或字面搜索。
3. 查询支持当前媒体库、当前目录（可递归）和全部媒体库，以及现有媒体类型范围；结果使用稳定游标，
   模型代次或索引代次变化时返回明确冲突/覆盖率状态。
4. 语义搜索失败、模型缺失、媒体库离线或索引不完整时，普通浏览、字面搜索和查看器继续可用。
5. 不向客户端、日志或诊断包返回原始向量、查询原文、绝对路径或模型原始错误。

## 已冻结产品默认

- semantic 与未来 face 能力按媒体库独立开关。
- 默认清除只删除可重建派生数据；未来人物名、人工 assignment、exclusion 和 cannot-link 必须单独
  二次确认，不能被关闭 AI 或模型升级静默删除。
- 未来人物库采用实例级人物实体；face observation、匿名组与资产关系按媒体库隔离，只允许用户显式
  跨库合并到同一人物。
- 模型获取基线是 `/models:ro`；没有运营 owner 时不显示没有真实后端的下载入口。

## 不在 revision 1

- Slice C：AI 标签建议。保留为后续独立 scope revision；现有人工标签不变。
- Slice D：视频代表帧语义搜索。保留为后续独立 scope revision；现有故事板不等于视频语义能力。
- Slice E：匿名人脸聚类与人物库。保留为后续独立 scope revision；没有合法 ground truth、隐私评审和
  可商业分发 embedding 前不得开工。

## 非目标

- 自动识别、猜测或建议 Coser、模特、公众人物、真人身份或动漫角色姓名。
- 年龄、性别、种族、情绪等敏感属性推断，人脸登录、监控或外部人物库。
- OCR、自由 caption、内容审核、NSFW 分类、生成或编辑图片。
- 视频语义、声音、字幕、动作时序、精确时间段检索和视频人脸追踪。
- AI 标签自动写入、自动接受建议、自动合并已命名人物或覆盖人工关系。
- 云推理、任意模型市场、任意下载 URL、GPU、独立 AI worker、第二容器、Redis、外部数据库或新
  部署单元。
- 修改、移动、重命名、删除原媒体，或把 AI 数据写回媒体/sidecar。

## 交付 Gate

1. S1 只冻结 A+B 的 PRD/user flow、架构、数据、OpenAPI、错误、任务、删除、备份和部署合同。
2. S2 Backend Evidence 前必须获得合法代表性图片质量集并达到冻结的中英文检索门槛；失败则删除 B，
   不改成云推理或放宽门槛。
3. S2/Release Gate 必须在原生 Linux/amd64 与 arm64 验证 runtime、数值容差、取消、恢复和资源边界。
4. Release Gate 必须完成最终模型/runtime SBOM、许可证/再分发、notices、漏洞/VEX 和 provenance 签署；
   未通过则不分发对应模型。
5. 4 CPU/4 GiB、10k 目录/100k 媒体完整进程容量失败时先降低后台并发或改为手动 backfill；仍失败则
   删除 B，不增加服务或放宽现有浏览预算。
6. 既有 catalog 搜索 query-plan 债务使用独立 maintenance Gate，不属于本 scope，也不能归因给 AI。

## 验收 ID

- `POST-MVP-5-AC-001`：禁用 AI 时没有模型常驻或后台 admission，现有核心流程不退化。
- `POST-MVP-5-AC-002`：`/models:ro` 扫描、托管复制、严格直接读取、激活、失效和恢复均失败关闭。
- `POST-MVP-5-AC-003`：图片语义搜索在合法评测集达到冻结质量门槛，并保持 scope/filter/cursor 稳定。
- `POST-MVP-5-AC-004`：模型缺失、损坏、索引不完整、媒体库离线和取消重启均保留最后可靠状态。
- `POST-MVP-5-AC-005`：原生 amd64/arm64、4 CPU/4 GiB、100k 完整进程和最终供应链 Gate 通过。
- `POST-MVP-5-AC-006`：原媒体 hash/mtime 不变，API/日志/诊断不暴露路径、查询原文或向量。

本 manifest 只冻结范围，不宣称功能已实现。生产代码仍必须按 S1→S2→S3→S4 后端优先 Gate 交付。
