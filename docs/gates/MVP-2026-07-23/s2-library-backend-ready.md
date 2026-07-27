# Stage 2 媒体库管理 Backend Ready

## 结论

**Go — S2-007 Backend Ready。**

安全目录选择与媒体库创建、列表、详情、改名、异步移除后端已经满足 Gate S2。前端可以使用
生成客户端对这些已冻结端点建立领域 adapter，不得再为媒体库管理创建 mock 行为、平行 wire
类型或第二套错误语义。

本结论不表示扫描 worker 已完成。创建媒体库会可靠地产生一个 durable queued full scan，
手动重试也只完成 durable admission；扫描领取、遍历、索引、取消收敛、计划任务和历史/详情
仍由 `S2-102～S2-107` 阻断。依赖“创建后实际完成扫描”的产品流程、媒体库/扫描 Integrated
Done、共享网络预览和稳定版本发布均未获授权。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- Roadmap stage：Stage 2 / `S2-LIB`
- 需求：`FR-LIB-001～008`、`FR-SCN-001`、`NFR-SAFE-001`、
  `NFR-SEC-001～002`、`NFR-PRIV-001`、`NFR-REL-001`
- Capability owner：`internal/library`
- Filesystem adapter：`internal/files`
- Persistence adapter：`internal/store/sqlite`
- Transport owner：`internal/api`
- Composition owner：`internal/app`
- 权威 HTTP 契约：`api/openapi.yaml`
- 权威数据契约：`migrations/00001_initial.sql`、
  `migrations/00003_library_contract.sql`、`migrations/00004_scan_contract.sql` 与
  `internal/store/sqlite/queries/`
- 架构决策：ADR-0001、ADR-0002、ADR-0003、ADR-0004、ADR-0006、ADR-0008、ADR-0009
- 风险：R-002、R-003、R-004、R-005、R-011、R-012、R-013、R-016

实现与验证由以下已合并切片组成：

- [PR #23](https://github.com/HappyQuQu/foliopath/pull/23)：S2-001 媒体库 Contract Ready。
- [PR #24](https://github.com/HappyQuQu/foliopath/pull/24)：S2-002 安全目录枚举。
- [PR #25](https://github.com/HappyQuQu/foliopath/pull/25)：S2-003 领域规则与 SQLite 约束。
- [PR #26](https://github.com/HappyQuQu/foliopath/pull/26)：S2-101 扫描 Contract Ready。
- [PR #27](https://github.com/HappyQuQu/foliopath/pull/27)：S2-004 媒体库生命周期。
- [PR #28](https://github.com/HappyQuQu/foliopath/pull/28)：S2-005 文件系统安全矩阵。
- [PR #29](https://github.com/HappyQuQu/foliopath/pull/29)：S2-006 原媒体不变证明。

## Gate S2 逐项判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 契约与所有权 | OpenAPI 固定 7 个媒体库管理 operation；`internal/library` 拥有名称、根、重叠、游标、幂等和移除规则；API/SQLite/files 仅作 adapter | 通过 |
| 真实组合与认证 | production composition root 接入 path、lifecycle、scan admission 与 removal worker；真实 setup/session/CSRF HTTP 测试经过同一组合点；业务 API 默认拒绝匿名请求 | 通过 |
| 文件系统边界 | 请求只接受 allowed-root-relative path；词法规则后只由 `internal/files` 执行 anchored open；Linux `openat2` 拒绝 symlink、同/跨设备与 self-bind 后代 mount，能力不可用时失败关闭 | 通过 |
| 目录选择 | 直接子目录以 256 项批次流式枚举，页面最多 200；自然排序、query-bound opaque keyset cursor、breadcrumb、冲突标注、取消和不可用父目录均有测试 | 通过 |
| 业务规则与并发 | 名称 NFC 展示、NFKC + full case-fold 唯一键、组件级根重叠、根不可变、强 ETag/If-Match、无变化改名和多 Store 并发冲突有单一 owner 与回归证据 | 通过 |
| 持久化与事务 | WAL、busy timeout、串行有界写；创建在一个短事务提交 library/唯一 creation scan/摘要幂等记录；真实 I/O 不在事务内；migration 只追加且可从前一版本升级 | 通过 |
| 分页与幂等 | 列表按自然名称派生键、名称、ID 做稳定 keyset；游标 AEAD 保护；创建/移除 key 只存 SHA-256，至少保留 24 小时，同 key replay 与冲突语义稳定 | 通过 |
| 离线与失败语义 | root 缺失、身份替换、权限、symlink/mount、ABA 与内部错误均失败关闭并映射稳定安全错误；offline 不当作空库，不清理最后可靠索引 | 通过 |
| 异步移除 | durable removal 先阻止新扫描并请求协作取消；worker 等安全终态后分批清理 SQLite 与 `/app/data/cache`；重启可续作且幂等 | 通过 |
| 原始媒体不变 | 真实认证 DELETE → composition → SQLite → worker 链路对完整媒体树的类型、mode、symlink target 和文件全部字节作前后比较；架构测试固定 removal 不持有媒体写入能力 | 通过 |
| 公开错误与隐私 | 统一错误只返回 code/message/requestId；路径、SQL、errno、stack、Cookie/CSRF、幂等明文和内部错误不进入响应 | 通过 |
| 契约与生成漂移 | OpenAPI 离线结构/引用/pattern/operation 错误码测试、Redocly、摘要锁、TypeScript 与 sqlc 确定性重生成均通过 | 通过 |
| 扫描执行 | 本切片只承诺 creation/manual durable admission 和移除前协作取消；生产扫描 worker 的领取、执行、恢复和容量由 S2-102～107 验收 | 不适用，后续 Gate 阻断 |

## 前端可以依赖的行为

- `GET /api/v1/library-paths`：只返回 `/library` 下服务端批准的直接子目录、breadcrumb、
  选择阻断原因和有界 cursor page，不返回宿主机路径。
- `GET /api/v1/libraries` 与 `GET /api/v1/libraries/{libraryId}`：返回稳定分页、状态、
  最后可靠计数、最新扫描摘要和用于修改的强 ETag。
- `POST /api/v1/libraries`：严格接收 `{name, rootPath}`，要求 CSRF 和
  `Idempotency-Key`；成功原子创建媒体库与 queued creation scan。
- `PATCH /api/v1/libraries/{libraryId}`：只允许改名，要求当前 `If-Match`；MVP 不允许改变根。
- `DELETE /api/v1/libraries/{libraryId}`：只移除配置、索引、任务与缓存，要求
  `If-Match` 和幂等 key，返回可轮询的 removal Location。
- `GET /api/v1/library-removals/{removalId}`：终态在库配置删除后仍可读取，并支持
  `If-None-Match` / `304`。
- 前端只通过生成 client 和唯一 domain adapter 消费这些 operation；不得自行规范化根、
  判断重叠、解码 cursor、拼装 ETag、复制错误映射或推断文件系统状态。
- 在 S2-107 前，前端不得把 queued scan 显示为会被当前版本实际执行，也不得完成依赖扫描
  进度、取消、结果或自动收敛的产品流程。

## 已执行验证

本地在 S2-006 已合并实现和当前审计工作区上实际执行并通过：

```text
make fmt
make arch-check
make generate-check
make lint
make web-check
make openapi-lint
make test
make test-race
make test-integration
make test-e2e
```

`openapi-lint` 通过并保留两个既有 health operation 缺少 4XX 的非阻断警告。
[PR #29 CI run 30240950263](https://github.com/HappyQuQu/foliopath/actions/runs/30240950263)
在最终生产实现上 14/14 jobs 通过，覆盖原生 amd64/arm64 Go 与 race、生成契约、真实应用
容器、Linux mount boundary、runtime/recovery、媒体矩阵、SBOM 和 high-severity npm audit。

## 保留限制与后续 Gate

- `S2-102～107` 仍须实现并验证有界扫描 worker、全部目录/媒体索引、generation finalize、
  取消、离线/失败保留、租约恢复、公平性和容量；扫描当前只有契约与 admission。
- Stage 3 的目录树、资产列表、缩略图和媒体内容尚未 Backend Ready；媒体库计数只能表示
  最后可靠索引，不能被前端扩展成未实现的浏览能力。
- `/library:ro` 正式发布挂载、运行期 unmount、真实版本升级、备份恢复、最终镜像和非回环
  网络发布继续由 Release Gate 阻断。
- 10 万媒体/1 万目录目标是后续扫描/浏览容量验收档，本 Gate 不宣称该规模已经由生产链路证明。
- 改根、多用户/权限、嵌套 mount、多个容器媒体 mount、文件修改/删除、watcher、
  自动去重和分享不属于当前 MVP 媒体库管理范围。

## 交接

- 后端状态：媒体库管理 `Backend Ready`
- 允许的下一步：后端进入 `S2-102` 扫描 worker；前端可建立媒体库 operation 的真实
  client adapter 与契约状态，但依赖扫描完成的用户流程继续等待 `S2-107`。
- 禁止的声明：扫描 Backend Ready、媒体库/扫描 Integrated Done、浏览/缩略图可用、
  LAN/公网可用、容量目标已证明或稳定 MVP 可发布。
- 评审日期：2026-07-27
