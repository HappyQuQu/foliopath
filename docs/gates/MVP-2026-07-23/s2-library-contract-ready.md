# Stage 2 媒体库管理 Contract Ready

## 结论

**Go — S2-001 Contract Ready。**

安全目录选择、媒体库创建/列表/详情/改名、离线展示/重试交接和异步移除的 HTTP、数据、
并发、幂等与失败语义已经冻结。可以进入 `S2-002` 安全目录枚举和 `S2-003` 媒体库规则
实现。

本记录不表示媒体库后端已经可用。`S2-004` 中创建后唤醒/执行首次扫描、离线重试和活动扫描
取消必须继续服从尚待完成的 `S2-101` 扫描 Contract Ready；不得用临时队列或第二套扫描
状态机绕过该依赖。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap stage：Stage 2 / `S2-LIB`
- 需求：`FR-LIB-001～008`、`FR-SCN-001`、`NFR-SAFE-001`、`NFR-SEC-001～002`、
  `NFR-PRIV-001`、`NFR-REL-001`
- Architecture Ready：
  [Stage 2 媒体库与可靠扫描](stage-2-architecture-ready.md)
- Capability owner：`internal/library`
- Filesystem adapter：`internal/files`
- Persistence adapter：`internal/store/sqlite`
- Transport owner：`internal/api`
- Composition owner：`internal/app`
- 权威 HTTP 契约：`api/openapi.yaml`
- 权威数据契约：`migrations/00001_initial.sql`、
  `migrations/00003_library_contract.sql` 和
  `internal/store/sqlite/queries/libraries.sql`
- 架构决策：ADR-0001、ADR-0002、ADR-0003、ADR-0004、ADR-0006、ADR-0008、ADR-0009
- 风险：R-002、R-003、R-004、R-005、R-011、R-012、R-013、R-016

## 已冻结的 HTTP 行为

### 目录选择

- `GET /api/v1/library-paths` 只列出 `/library` 或指定 approved parent 的直接普通子目录，
  使用自然排序、keyset cursor、默认 50/最大 200 的有界页面。
- `parent` 是 allowed-root-relative 值；空或省略表示 `/library`。绝对路径、dot segment、
  NUL、separator/编码绕过在 I/O 前拒绝。
- picker 不跟随目录 symlink，不跨后代 mount，不返回文件、宿主机路径、mount source、
  device/inode 或可供媒体读取复用的路径能力。
- 不可选直接子目录可用稳定 blocked reason 表示；当前 parent 自身不安全或不可用时整个请求
  以稳定 409 失败，不能伪装为空页。创建时必须重新执行全部安全/重叠检查，不能信任 picker
  之前的结果。

### 创建、列表与详情

- `POST /api/v1/libraries` 要求 session、CSRF、`Idempotency-Key` 和严格 JSON
  `{name, rootPath}`；`rootPath=""` 唯一表示 `/library`。
- 名称在服务端规范化后非空、实例唯一；根必须存在、可读、安全、无 symlink/mount crossing
  且与任何已有根没有相同/祖先/后代重叠。
- 库记录、唯一 `library_created` queued full scan 和幂等记录在一个短 SQLite 事务提交；
  真实根检查在事务外，worker 只在提交后唤醒。任一写入失败时三者一起回滚。
- 成功返回 `201`、Library/ScanRun、Location、强 ETag 和
  `Idempotency-Replayed`。同 key/同规范请求返回同一逻辑结果且绝不创建重复库；同 key/不同
  请求返回 `idempotency_conflict`。结果已被移除时，在保留期内返回
  `idempotency_conflict`，不会重新执行创建。
- `GET /api/v1/libraries` 使用名称 + 稳定 ID 的 keyset 顺序；详情返回
  allowed-root-relative `rootPath` 和仅供管理员显示的 `/library/...` 标签，不返回宿主机路径。
- Library 的 asset/directory counts 来自最后可靠索引。offline/error 不是空库，保留原计数。

### 改名与并发

- `PATCH /api/v1/libraries/{libraryId}` 只接受 `{name}`；schema 不接受 `rootPath`，
  capability 也不提供修改根的方法。
- GET/创建返回的强 ETag 在任一 Library 表示字段变化时更新。PATCH 和 DELETE 必须携带
  `If-Match`；缺失为 428，过期为 412。
- 改为相同规范名称是 no-op，不推进 validator；改名不改变 ID、根、索引、扫描历史或缓存。
- 处于活动 removal 的库不能改名，返回 `idempotency_conflict`。

### 离线与重试

- 已存在的库根不可用、不可读、身份被替换或边界失效时显示 offline 并保留最后可靠索引；
  不能把它解释为空或在读取路径中清理记录。
- “重试”复用 `POST /api/v1/libraries/{libraryId}/scans` 的 durable full-scan admission，
  不另建 library-local queue。具体 coalesce/create、取消和 schedule 行为由 `S2-101` 再次
  审计，但不得改变 offline 不清理和同库唯一 active scan。

### 异步移除

- `DELETE /api/v1/libraries/{libraryId}` 要求 session、CSRF、If-Match 和 Idempotency-Key，
  返回 `202`、Location 与可轮询的 `LibraryRemoval`。
- 接受操作在短事务中建立 durable removal、阻止新扫描并请求协作取消 queued/running scan；
  worker 等扫描到达安全终态后，分批、幂等清理 SQLite 配置/索引/jobs 与 `/app/data` 缓存。
- removal workflow 没有打开、移动、改名、写入或删除 `/library` 的 capability。缓存 I/O
  在数据库事务外；删除大索引不使用一个无界事务。
- 同 key 重试返回同一 removal；不同 key 遇到 active removal 返回
  `idempotency_conflict`。terminal removal 在库配置删除后仍保留可轮询的 ID、库 ID/名称
  快照、状态、时间和安全错误码。

## 已冻结的错误矩阵

`api/openapi.yaml` 的 `x-error-codes` 是每个 operation 的稳定错误映射：

| 类别 | 稳定行为 |
| --- | --- |
| 请求/游标 | `invalid_request`、`invalid_cursor`、`validation_failed` |
| 会话/状态修改 | `authentication_required`、`session_expired`、`csrf_invalid` |
| 路径 | `library_root_unavailable`、`library_root_outside_allowed`、`library_root_symlink`、`library_root_mount_boundary` |
| 业务冲突 | `library_name_conflict`、`library_path_overlap` |
| 幂等/活动操作 | `idempotency_conflict` |
| 并发 | `precondition_required`、`precondition_failed` |
| 查找 | `library_not_found`、`removal_not_found` |
| 资源/内部 | `rate_limited`、`internal_error` |

公开错误仍只有 `code/message/requestId`，不得暴露 SQL、stack、errno、绝对路径、mount
细节、Cookie/CSRF、幂等明文或原始文件名的无界内容。前端只按 code 分支。

## 已冻结的数据与事务

- `libraries.revision` 为正整数，支持强 validator；根不可变 trigger 保持生效。
- `scan_runs_one_creation_per_library` 保证每个 library 最多一个
  `library_created` run；既有 active-scan partial unique index继续保证 queued/running 唯一。
- `library_removals` 是 restart-safe durable 状态，具备 queued/running/succeeded/failed
  状态/时间一致性检查和每库唯一 active removal。
- terminal removal 故意不外键级联到 `libraries`，以便配置删除后仍可轮询结果；它只保存
  opaque library ID 和安全名称快照，不保存根或宿主机路径。
- `idempotency_records` 以 `(operation, SHA-256(key))` 唯一，request hash 也是固定 32
  字节；至少保留 24 小时，不保存明文 key、请求体或宿主机路径。
- migration 只追加为 `00003_library_contract.sql`；`00001`/`00002` 没有被改写。

## 自动证据

- OpenAPI 离线解析、引用、ECMAScript pattern、生成 TypeScript、摘要锁与外部 lint。
- operation 级测试固定七个媒体库/移除端点的错误码、ETag、幂等、首次扫描和安全移除描述。
- migration 升级测试从 version 2 真实升级到 version 3。
- SQLite 测试证明创建事务 commit/rollback 同时影响 library/creation scan/idempotency，
  第二个 creation scan 被拒绝。
- SQLite 测试证明每库只有一个 active removal、幂等摘要长度/唯一性/24 小时下限，且库配置
  删除后 removal/idempotency 记录仍存在。
- sqlc 和 OpenAPI 生成均由源重新生成，`generate-check` 无漂移。

## 明确未完成

- `S2-002`：生产 `/library` 目录枚举、分页、自然排序和 `internal/files` 接线。
- `S2-003`：生产服务的名称、根、重叠、ETag/If-Match 和并发规则。
- `S2-004`：创建/首次扫描原子 service、列表/详情/改名、offline/retry 和 removal worker。
- `S2-005`：traversal、symlink、nested mount、TOCTOU、重叠、离线、权限和公开错误测试。
- `S2-006`：移除前后原文件逐字节不变证明。
- `S2-007`：媒体库 Backend Ready Gate。
- `S2-101`：完整扫描 API、状态、issues、取消、scheduler 和资源上限 Contract Ready。

在 `S2-007` 前不得把媒体库 API 描述为可用；在 `S2-101` 前不得实现临时扫描 admission、
worker、取消或 schedule 语义。

## 交接

- 后端状态：`Contract Ready`
- 允许的下一步：`S2-002`，只通过 `internal/files` 实现安全、有界的 approved directory
  enumeration；随后 `S2-003` 实现 capability 规则。
- 禁止的声明：媒体库 Backend Ready、扫描 Contract/Backend Ready、前端可连接媒体库 API、
  LAN/公网可用、容量目标已证明或 MVP 可发布。
- 评审日期：2026-07-27
