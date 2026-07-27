# Stage 1 单管理员认证 Backend Evidence Ready

## 结论

**Go — S1-106 Backend Evidence Ready。**

单管理员初始化、登录、会话查询、退出、绝对过期、撤销、CSRF 和全部业务 API 默认认证
已经满足 Gate S2。正式前端可以从本记录合入后使用生成客户端连接五个真实认证端点，不再
为这些行为创建 mock 或平行契约。

该结论只授权认证前端进入 Consumer/UI Ready 工作。它不表示认证 UI 已完成，不授权
Stage 2 后端、非回环监听、可信代理、公网或局域网预览、匿名模式，也不表示稳定版本可以
发布。

## 范围与权威来源

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- 需求：`FR-AUTH-001～004`、`NFR-SEC-002`、`NFR-PRIV-001`
- Capability owner：`internal/auth`
- Transport owner：`internal/api`
- Persistence adapter：`internal/store/sqlite`
- Composition owner：`internal/app`
- 权威 HTTP 契约：`api/openapi.yaml`
- 权威数据契约：`migrations/00002_authentication.sql` 与
  `internal/store/sqlite/queries/auth.sql`
- 架构决策：ADR-0001、ADR-0005、ADR-0006、ADR-0008
- 风险：R-010、R-011、R-016

实现与验证由以下已合并切片组成：

- [PR #16](https://github.com/HappyQuQu/foliopath/pull/16)：S1-101 Contract Ready。
- [PR #17](https://github.com/HappyQuQu/foliopath/pull/17)：S1-102 初始化与密码。
- [PR #18](https://github.com/HappyQuQu/foliopath/pull/18)：S1-103 安全会话。
- [PR #19](https://github.com/HappyQuQu/foliopath/pull/19)：S1-104 HTTP、CSRF 与默认拒绝。
- [PR #20](https://github.com/HappyQuQu/foliopath/pull/20)：S1-105 安全、并发与时间矩阵。

## Gate S2 逐项判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 业务规则归属 capability | 身份规范化、setup 状态机、密码策略、会话发放/验证/撤销和期限只在 `internal/auth`；架构测试禁止认证路由重复实现 | 通过 |
| HTTP 不越过 adapter | `internal/api` 只依赖 `AuthenticationService`，只负责 DTO、Cookie、Origin、CSRF、限流和公开错误映射；不查询 SQLite、不解析真实路径、不调用媒体工具 | 通过 |
| 唯一 composition root | `internal/app` 构造密码管理器、SQLite repository、认证 service 和 routes；真实运行测试经过该组合点 | 通过 |
| 真实持久化与事务 | 真实文件 SQLite、WAL、migration、singleton、setup 与初始 session 同事务、回滚、重启恢复、撤销和 last-seen 测试 | 通过 |
| 并发与时间 | 进程内 setup gate、SQLite singleton、8 路真实 HTTP setup 单胜者、32 路 session 并发、绝对期限前 1 ms/等于期限边界 | 通过 |
| 故障与隐私 | 未知账号与错误密码同为 `invalid_credentials`；五类认证操作的内部错误不泄露密码、令牌、SQL 或 `/app/data`；失败/过期/撤销状态稳定 | 通过 |
| 网络与请求安全 | 精确匿名白名单、其余 `/api/v1` 默认认证、同源 Origin、session-bound CSRF、严格 JSON、`no-store`、host-only Cookie、真实 HTTPS `Secure`、直连 peer 有界限流 | 通过 |
| 生成与契约 | OpenAPI 逐状态错误码、Cookie/CSRF、安全边界和 migration 契约测试；TypeScript client 与 sqlc 确定性生成无漂移 | 通过 |
| 后台任务适用性 | 认证切片没有后台任务、文件系统或媒体处理；Gate S2 的取消、离线和媒体并发条款不适用 | 不适用 |

## 前端可以依赖的行为

- `GET /api/v1/auth/status` 匿名且只返回 setup/authentication 状态。
- `POST /api/v1/auth/setup` 需要同源 Origin；成功原子创建唯一管理员并建立会话；重复和
  并发请求使用稳定的 `setup_closed` / `setup_in_progress`。
- `POST /api/v1/auth/login` 需要同源 Origin；未知账号、非法身份和错误密码不允许前端区分。
- `GET /api/v1/auth/session` 由 HttpOnly Cookie 恢复管理员、CSRF token 和绝对期限；
  未登录与已失效会话分别使用 `authentication_required` / `session_expired`。
- `POST /api/v1/auth/logout` 同时要求有效 Cookie 与 `X-CSRF-Token`；成功撤销服务端会话
  并过期浏览器 Cookie。
- 除精确匿名操作外，所有现在和未来的 `/api/v1` 业务路径先经过会话验证；状态修改还先
  经过 CSRF 验证。
- 前端只通过生成 client 和统一 domain adapter 消费这些端点；不得读取 Cookie、复制错误
  码、重新定义 Origin/CSRF 规则或根据未知账号与错误密码显示不同文案。

## 已执行验证

本地在 S1-105 最终实现上实际执行并通过：

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
[PR #20 CI run 30230672999](https://github.com/HappyQuQu/foliopath/actions/runs/30230672999)
在最终提交上 14/14 jobs 通过，覆盖原生 amd64/arm64 Go 与 race、生成契约、真实应用容器、
Linux mount boundary、媒体矩阵、runtime/recovery、SBOM 和 high-severity npm audit。

## 保留限制与后续 Gate

- 当前进程配置继续只接受数值回环监听地址。可信代理范围、转发头、非回环绑定和网络发布
  必须由 Stage 5 安全与部署 Gate 明确设计和验收；本 Gate 不允许绕过该限制。
- 前端仍须完成 Consumer/UI Ready：首次 setup、login、session 恢复、logout、过期跳转、
  中英文、键盘/焦点、错误映射和浏览器测试。
- 认证数据是 `/app/data` 中不可重建状态；正式备份、恢复、真实版本升级和最终镜像仍由
  Stage 5 阻断。
- 密码修改/找回、多管理员、多用户、匿名 LAN、OAuth/OIDC 和细粒度权限不属于当前 MVP。
- Stage 2 媒体库/扫描后端没有被 Stage 0 Gate 自动授权；开始前仍需按治理记录稳定切片、
  owner、风险、契约与验收证据。

## 交接

- 后端状态：`Backend Ready`
- 允许的下一步：认证前端进入 Gate S3，并连接真实 `/api/v1/auth/*` 契约。
- 禁止的声明：认证 UI 已完成、产品可共享预览、Stage 2 已开工、稳定 MVP 可发布。
- 评审日期：2026-07-27
