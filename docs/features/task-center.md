# FTR-OPS-001：后台任务中心

## 状态与版本

- 状态：Confirmed Direction / Scope Proposed
- 目标版本：`POST-MVP-3` / `Post-MVP/3`
- 当前阶段：S0 Architecture Ready 准备；不授权生产实现或修改权威 OpenAPI
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Change Record：[CR-2026-008](../changes/CR-2026-008-task-center.md)
- 开发清单：[FTR-OPS-001 开发任务清单](task-center-task-list.md)
- 新风险：[R-020](../risk-register.md)

`POST-MVP-3` 当前只是目标版本标签，没有冻结 scope manifest。该 feature 不进入正在加固的
MVP，也不插入 `POST-MVP-1` 或已冻结的 `POST-MVP-2`。

## 用户问题

FolioPath 已有可靠完整扫描、媒体派生任务、租约恢复、缓存水位和失败代码，但这些能力分散在
不同 capability 与页面。管理员无法从一个入口回答：

- 当前有哪些高层任务正在等待或运行；
- 某次任务处于什么阶段，处理了多少项目；
- 失败、离线或取消后是否保留可靠状态；
- 哪些任务可以取消、重试、补齐缺失派生数据或完整重建；
- 操作是否创建了重复或无界后台工作。

任务中心提供统一的产品投影和安全操作入口，但不建立第二套 scanner、thumbnail 或 worker
状态机，也不把逐资产内部队列表直接暴露给浏览器。

## 需求

| ID | 需求 |
| --- | --- |
| `FR-OPS-001` | 管理员必须能在独立任务中心查看高层任务，并按进行中、全部、需处理筛选。列表使用稳定 keyset cursor，不读取或渲染无界历史。 |
| `FR-OPS-002` | 每个高层任务必须有可直达详情，显示类型、状态、阶段、可靠进度或未知总量、计数、时间、安全错误摘要和当前允许操作。 |
| `FR-OPS-003` | 取消和重试必须幂等、可恢复并委托给任务真实 owner；取消、失败、离线和中断不得删除最后可靠索引或已安全发布的派生缓存。 |
| `FR-OPS-004` | 管理员可以按媒体库或全部媒体库创建“补齐缺失缓存”批次；批次只登记缺失、过期或不兼容的派生项，并有界分页 admission。 |
| `FR-OPS-005` | 管理员可以在明确确认后创建“重建全部缓存”批次；新派生文件成功发布后才替换旧缓存，不清空可用缓存作为开始条件。 |
| `FR-OPS-006` | 终态任务和稳定失败摘要必须保留用于诊断；首版不提供删除任务历史、清空失败记录或任意修改内部并发数。 |
| `NFR-OPS-002` | 任务中心聚合、轮询、批量 admission 和详情查询必须有界，不能使浏览、扫描、poster 或现有缩略图任务无限饥饿。 |

## 首版范围

### 包含

- 既有完整扫描 run 的统一列表投影、详情、取消和重新请求；
- 缺失缩略图/视频封面的有界补齐批次；
- 全部缩略图/视频封面的有界重建批次；
- 派生批次的 durable parent run、进度、取消、失败摘要和重试历史；
- 活动、全部、需处理筛选，以及可选的媒体库和任务类型筛选；
- 任务页运行概览：active、attention、在线媒体库和缓存水位；
- 原型已有的独立任务详情路由和安全说明。

### 不包含

- 系统健康、完整性检查、数据库备份、诊断包或版本更新；
- AI 搜索、OCR、人脸识别、重复检测或其他未来模型任务；
- 逐资产任务浏览器、原始错误日志或底层 SQL/lease 管理；
- 暂停全部 worker、清空队列、删除失败历史或绕过重试策略；
- 每种任务的任意并发输入框；
- 新部署单元、独立 worker 服务、Redis、外部数据库或 WebSocket。

## 产品任务模型

### 高层任务，不是内部工作项

任务中心只展示可理解、可操作的高层 run：

1. `full_scan`：现有 `scan_runs`；
2. `derived_backfill`：补齐缺失或过期派生数据的批次；
3. `derived_rebuild`：经确认后完整重建派生数据的批次。

`media_jobs` 中的逐资产 grid/storyboard 工作项仍是内部队列事实，不成为公共任务历史。页面
可以在高层 run 详情中显示聚合计数，但不能向客户端发送 10 万个子任务。

### 状态

统一公共状态：

- `queued`
- `running`
- `succeeded`
- `failed`
- `cancelled`
- `offline`
- `interrupted`

公共投影不能改变底层语义：

- scan 继续使用现有 `queued/running/succeeded/failed/cancelled/offline/interrupted`；
- derived run 的 `offline` 表示源媒体库当前不可用，不表示媒体不存在；
- 只有 capability owner 可以决定状态转换和稳定错误码；
- 聚合层只映射状态，不复制 retry、lease、cleanup 或 generation 规则。

### 阶段

derived run 的候选阶段：

1. `queued`
2. `enumerating`
3. `admitting`
4. `processing`
5. `finalizing`
6. `completed`

扫描阶段继续使用权威 `ScanPhase`。统一 DTO 允许稳定的 task-specific phase 字符串，但客户端
只能用于展示，不能根据 phase 推导可操作性；`allowedActions` 是唯一操作依据。

### 进度

- `progressRatio` 只有在拥有可靠分母时才返回 0～1，否则为 null；
- derived run 在枚举完成前总量可为空，不能使用历史资产数量伪造百分比；
- 计数至少区分 `discovered`、`queued`、`running`、`succeeded`、`failed`、`skipped`；
- 计数和状态变化推进强 ETag，详情页使用条件轮询与退避；
- 终态停止轮询。

## 操作语义

### 补齐缺失缓存

请求参数限定为：

- scope：全部媒体库或一个媒体库；
- variant：首版为 `grid`；视频封面属于 grid 的媒体类型处理，不建立第二套 poster 队列；
- mode：`missing`。

枚举只登记以下项目：

- 没有 ready 派生状态；
- ready 记录的缓存文件缺失；
- source fingerprint 或 transform version 不匹配；
- transient failed 且仍在允许重试范围，或管理员显式重试的新 run。

unsupported/invalid 等永久失败不得在每次补齐时无限重试；必须由 transform/model/version 变化
或显式的受限重试规则重新获得资格。

### 重建全部缓存

- mode 为 `all`，必须经过前端确认；
- 不先删除现有 ready 缓存；
- admission 按稳定 asset keyset 小批登记当前 transform version 的替换工作；
- 新文件写临时路径并原子发布，SQLite ready 提交成功后旧缓存进入可恢复删除队列；
- 取消停止后续 admission，已经安全完成的替换保留；
- 重试创建新的 run 并通过 `retryOfTaskId` 链接旧历史，不覆盖旧终态。

### 取消

- scan 取消调用现有 scanner cancellation operation；
- queued derived run 可直接到 cancelled；
- running derived run 首次记录 cancel request，并停止后续 admission；
- 已被 worker 领取的单项是否完成由 thumbnail owner 的 checkpoint/发布规则决定，不能在任意
  native 调用中强杀并伪装回滚；
- 重复取消幂等返回当前投影；
- terminal run 返回稳定 `task_already_finished`。

### 重试

- scan 重试继续调用现有按库 request/coalesce operation；
- derived 重试创建新 run，继承受控 scope/variant/mode，但重新执行资格枚举；
- 同 scope、variant、mode 已有 active run 时原子合并并返回现有 run；
- 不允许客户端提交内部 cursor、attempt、priority、lease 或 transform version。

## API 提案

本节是 S0 输入，不是权威合同。只有 `OPS-S1 Contract Ready` 通过后才能修改
`api/openapi.yaml`。

### 读

- `GET /api/v1/operations/tasks`
  - filters：`status=active|all|attention`、`kind`、`libraryId`；
  - keyset：`createdAt DESC, taskId DESC`；
  - 默认 50，最大 200；
  - 返回 `OperationTaskPage`。
- `GET /api/v1/operations/tasks/{taskId}`
  - 返回 `OperationTaskDetail`；
  - 支持强 ETag、`If-None-Match` 和 `304`。

`taskId` 是 operations owner 生成和解析的稳定不透明 ID，客户端不得拼接底层表名或数字 ID。
如果 S0 证明无持久映射即可安全、稳定地编码 source kind + source ID，可以采用版本化编码；
否则由只追加 migration 保存公共 ID 映射。该决定必须在 Contract Ready 前冻结。

### 写

- `POST /api/v1/operations/derived-runs`
  - body：`scope`、可选 `libraryId`、`variant`、`mode=missing|all`；
  - 使用 `Idempotency-Key`；
  - 新 run 返回 `202`，active 等价 run 返回 `200`。
- `POST /api/v1/operations/tasks/{taskId}/cancel`
  - 需要 CSRF；
  - 返回 `202` 和最新 task 表示。
- `POST /api/v1/operations/tasks/{taskId}/retry`
  - 需要 CSRF 与 `Idempotency-Key`；
  - 返回新 task 的 `202` 和 Location；等价 active run 返回 `200`。

### 稳定错误

- `task_not_found`
- `task_already_finished`
- `task_action_not_supported`
- `library_not_found`
- `library_offline`
- `idempotency_conflict`
- `operation_already_active`
- `operation_capacity_exceeded`
- `invalid_request`
- 既有认证、CSRF、限流和内部错误

错误响应不得泄露 SQL、底层表名、租约、宿主机路径、原始媒体相对路径、stderr 或 stack。

## 模块与所有权

| 行为 | 唯一 Owner | 说明 |
| --- | --- | --- |
| 跨 capability 列表、筛选、task ID、允许操作投影 | 新 `internal/operations` | 只组合公开 capability port，不复制底层状态机 |
| 完整扫描 admission、状态、阶段、取消、重试 | `internal/scanner` | 继续使用 `scan_runs` 和既有 API/service |
| derived run 资格、模式、阶段、聚合计数、取消 | `internal/thumbnail` | 拥有派生业务语义和批次 admission |
| lease、heartbeat、claim、公平、重试退避 | `internal/jobs` | 不成为 HTTP DTO owner |
| 安全媒体读取 | `internal/files` | operations 不解析或打开路径 |
| task read model 与 durable run repository | `internal/store/sqlite` | 实现 capability-owned interfaces |
| HTTP、认证、CSRF、DTO、错误映射 | `internal/api` | handler 只调用 operations service |
| lifecycle 与 worker wiring | `internal/app` | 不在 handler 创建 worker |
| URL codec、query、poll/invalidation | `web/src/features/operations` | 只通过生成 client 与 domain adapter |

新增 `internal/operations` 不改变模块化单体、部署单元或依赖方向，预期不需要 ADR。若 S0 选择
独立服务、外部队列、跨 capability 事务 owner 或新的持久化边界，必须先新增 ADR。

## 数据提案

预计需要只追加的 `operation_runs`，精确 migration 号在 Contract Ready 时按仓库当时最新
版本确定，不能提前占用或修改已发布 migration。

候选字段：

- public task ID 或其稳定 source mapping；
- kind、scope library、variant、mode；
- status、phase、revision；
- stable eligibility cursor；
- discovered/admitted/succeeded/failed/skipped counters；
- cancel requested、attempt/retry-of；
- created/started/finished/heartbeat/lease timestamps；
- stable error code。

不得存储宿主机绝对路径、原始 stderr、无界错误文本或逐资产错误数组。逐资产任务继续留在
`media_jobs`，聚合计数通过有界查询或事务更新获得。

## 资源与容量

- 列表和详情查询必须走覆盖索引，不能扫描完整 `media_jobs`；
- derived admission 每批有硬上限并使用 asset keyset，不使用 OFFSET；
- parent run 数、active 等价 run、全局 pending children 和单次批次均有限制；
- grid/poster 日常需求、浏览请求和手动扫描优先级不得被 rebuild 淹没；
- `mode=all` 默认最低后台优先级，允许在磁盘安全余量不足时暂停 admission 或失败关闭；
- 目标档使用约 100k 媒体、10k 目录、四核/4 GiB，测量 admission 时间、SQLite 写放大、
  queue depth、浏览 P95、RSS、取消延迟和重启恢复；
- 精确数值由 `OPS-S0` spike 冻结，本文件不虚构预算。

## 安全、隐私与可访问性

- 所有端点仅限认证管理员；写操作要求 CSRF；
- API 接受 library ID 和受控枚举，不接受路径；
- 清理和重建只作用于 `/app/data/cache` 的可重建文件；
- 任务状态必须由文字、图标/形状和语义属性共同表达，不能只依赖颜色；
- 筛选、操作、确认和详情轮询支持键盘与焦点恢复；
- terminal failure 保留在“需处理”，不只使用会消失的 toast；
- 中英文复制保持稳定 code 与本地化 message 分离。

## 验收

| ID | 验收 |
| --- | --- |
| `OPS-AC-001` | 100k 媒体历史和 active 状态下，任务列表使用稳定 cursor，页面与服务都不加载无界子任务。 |
| `OPS-AC-002` | scan 统一投影与既有 scan API 的状态、阶段、计数、取消能力和 ETag 一致。 |
| `OPS-AC-003` | missing backfill 只登记缺失/过期项目，重复请求合并，不重新处理可靠且当前版本的缓存。 |
| `OPS-AC-004` | all rebuild 不先删除旧缓存；取消、崩溃、盘满和重启后旧 ready 仍可用，新 run 可恢复或重试。 |
| `OPS-AC-005` | offline/部分不可读不产生批量删除，不把媒体库视为空，最后可靠索引与缓存保持可浏览。 |
| `OPS-AC-006` | retry 创建可追踪的新历史；失败记录不被清空，永久失败不会形成无限重试循环。 |
| `OPS-AC-007` | 浏览、搜索、poster/grid 请求和手动扫描在全量 rebuild 期间仍满足冻结或新测得的资源预算。 |
| `OPS-AC-008` | 任务中心桌面/移动、中英、深浅主题、键盘、焦点、reduced-motion 和三引擎无阻断回归。 |

## Gate

```text
OPS-S0 Architecture Ready
  → OPS-S1 Contract Ready
  → OPS-S2 Backend Evidence Ready
  → OPS-S3 Consumer/UI Ready
  → OPS-S4 Integrated Slice Done
```

当前只允许完成 S0 规格、风险、spike 计划和评审。S1 前不得改权威 OpenAPI、migration 或生产
代码；S2 前不得建立业务前端；S4 前不得宣称任务中心已可发布。
