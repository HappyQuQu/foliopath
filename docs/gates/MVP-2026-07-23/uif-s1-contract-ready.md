# Gate MVP-2026-07-23 / UIF-S1 / Contract Ready

- 日期：2026-07-30
- 目标版本：`MVP-2026-07-23` revision 4
- Feature：[FTR-UIF-001](../../features/frontend-prototype-fidelity.md)
- 前置：[UIF-S0 Architecture Ready](uif-s0-architecture-ready.md)
- 需求：`FR-AUTH-005`、`FR-BRW-010`、`FR-UI-009～010`、`NFR-UIF-001`
- 任务：`UIF-101～109`
- 风险：R-010、R-012、R-016、R-021
- 结论：**Go — 授权 UIF-201～211 后端实现**

## 已冻结合同

| 切片 | 权威 wire | 决定 |
| --- | --- | --- |
| Account profile | `GET/PATCH /api/v1/account` | 独立 account revision/ETag；用户名不可变；PATCH 要求 If-Match、session、CSRF |
| Password | `POST /api/v1/account/password` | 验证当前密码；单事务更新 verifier/account/auth revision；当前 session 提升并保留，其他 session 撤销 |
| Directory q | `GET /api/v1/libraries/{libraryId}/directories?q=` | 可靠索引、全部直接子目录、profile v1 literal substring AND、自然 keyset；query 进入 cursor 指纹 |
| Cache summary | `GET /api/v1/cache` | 只返回 aggregate usage/quota/waterline/free-space/pressure/cleanup；不返回路径或文件 |
| Cache cleanup | `GET/POST /api/v1/cache/cleanup` | 单例 durable async、幂等、active coalesce、重启恢复、清空全部可重建 thumbnail/poster，无历史/取消/补齐/重建 |

所有账户/cache 响应为 `no-store`；写请求使用 session-bound CSRF。账户校验复用
`validation_failed`，错误当前密码复用 `invalid_credentials`，没有扩大旧端点共享
ErrorCode enum。

## 数据决定

后续只追加 `00013` migration：

1. `users.revision`：单调 account ETag owner；
2. `directories.search_name_key`：catalog capability 派生的 profile v1 搜索键；
3. 单行 `cache_cleanup_state`：当前/最近一次 durable cleanup，不是任务历史；
4. 复用 `idempotency_records`，不保存明文 key、密码、路径或文件列表。

目录查询先使用既有 `directories_browse_children` 限定 parent，再执行精确 `instr`；当前不引入
directory FTS、外部搜索服务或新依赖。

## 失败与并发语义

- profile no-op 不推进 revision；错误/缺失 If-Match 分别返回 412/428；
- 改密前任一验证/hash/事务失败都保持 verifier 与全部 session 不变；
- 改密成功时当前 session 与 user auth version 同步推进，其他 session 在同一事务撤销；
- 目录 query 清除、改变或 generation 推进使旧 cursor 返回 `invalid_cursor`；
- cache active 请求合并；相同 idempotency key 回放原表示，冲突失败关闭；
- cleanup 删除派生文件后才移除 ready DB 状态，批次有界；重启恢复 queued/running；
- cleanup 失败保持可靠索引、设置和原媒体，允许新请求重试，不自动生成 replacement。

## 证据

- `api/openapi.yaml`：revision 4 / CR-2026-009，新增 6 个 operation、Schema、错误和追踪；
- `api/openapi.sha256`：更新后的权威摘要；
- `web/src/lib/api/generated/schema.ts`：由 OpenAPI 确定性生成；
- `tests/contract/openapi_contract_test.go`：operation、CSRF、ETag、password session、目录 q、
  cache singleton/privacy 和 ErrorCode 兼容断言；
- [UIF-001 目录查询 spike](../../spikes/uif-001-directory-filter.md)：10k direct children，
  parent index，末尾命中，P50 1.703250ms / P95 1.981167ms；
- `tests/performance/directory_filter_contract_test.go`：100ms P95 跨环境护栏。

执行成功：

```text
make openapi-lint
  valid；仅保留既有 health 4xx 两条 warning
oasdiff breaking --fail-on WARN <变更前 OpenAPI> api/openapi.yaml
  no breaking changes
make contract-check
make generate-check
make arch-check
go test -count=1 ./tests/performance
git diff --check
```

## 授权与阻断

允许：

- `UIF-201～211` auth/catalog/cache service、SQLite adapter、migration 13、HTTP 和 composition；
- `UIF-212～214` 正常/失败/安全/容量证据；
- 已由 S0 允许的 `UIF-301～306` 共享视觉基础继续并行。

仍阻断：

- account、directory q、cache cleanup 的生产前端接入，直到 `UIF-S2 Backend Evidence Ready`；
- 任务中心、missing/all rebuild、系统维护、备份、诊断、AI/OCR/人脸；
- 任何新部署单元、外部数据库/搜索、第二套 auth/cache state machine；
- 宣称整个 FTR-UIF-001 或 MVP 可发布。

## 结论

合同、数据方向、失败语义、兼容性、生成消费者和 10k 查询可行性均已冻结。`UIF-S1` 为 Go，
下一批必须从 capability service 和 migration 13 开始，不能从 React mock 或 handler 内规则
开始。
