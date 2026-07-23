# Stage 1 单管理员认证 Contract Ready

## 结论

**Go — S1-101 Contract Ready。**

初始化、登录、会话查询、退出和 CSRF 的 HTTP 与持久化契约已冻结，可以进入 `S1-102`
管理员初始化与密码存储实现。本记录不表示认证后端已经可用，也不解除认证完成前只能监听回环
地址的限制。

## 范围与追踪

- 目标版本：`MVP-2026-07-23`
- Scope revision：1
- 需求：`FR-AUTH-001～004`、`NFR-SEC-002`、`NFR-PRIV-001`
- 架构：ADR-0005；`internal/auth` 拥有认证规则，`internal/api` 只拥有 HTTP 映射，
  `internal/store/sqlite` 只实现认证 capability 定义的持久化接口。
- 权威 HTTP 契约：`api/openapi.yaml`
- 权威数据契约：`migrations/00002_authentication.sql`

## 已冻结的外部行为

- 仅 health、认证状态、首次 setup 和 login 允许匿名访问；setup/login 必须校验同源
  `Origin`。
- setup 最多原子创建一个管理员；成功同时建立会话，后续请求使用 `setup_closed` 或
  `setup_in_progress`，不泄露账户信息。
- login 对未知账号和错误密码统一返回 `invalid_credentials`。
- 会话使用 `foliopath_session`，要求 `HttpOnly`、`SameSite=Strict`、`Path=/`、有界
  `Max-Age`，HTTPS 时必须 `Secure`。
- 状态修改同时要求会话 Cookie 与 `X-CSRF-Token`；logout 撤销服务端会话并过期 Cookie。
- 五个认证端点具备逐状态稳定错误码映射，认证成功和公共 JSON 错误均为
  `Cache-Control: no-store`。
- setup 用户名保持既有 Unicode wire 兼容性；持久化显示值使用 NFKC，登录键使用 NFKC 后
  Unicode full case folding。login 无法匹配时仍使用统一凭据错误。

## 已冻结的数据不变量

- `users.singleton_key = 1` 且唯一，数据库层阻止并发创建第二个管理员。
- 密码只保存 verifier、scheme 和参数版本，不保存明文。
- 浏览器会话令牌和 CSRF 令牌只保存 32 字节摘要；会话令牌摘要唯一。
- session 包含创建、最近使用、绝对过期、可空撤销时间和 `auth_version`，并级联归属唯一管理员。
- 认证数据属于不可重建的 `/app/data` 内容，必须进入备份、恢复和升级门禁。

## 自动证据

- OpenAPI 离线结构、引用、ECMAScript pattern、认证/CSRF 边界、稳定错误码、防缓存、
  Cookie 与用户名规范化契约测试。
- SQLite migration 测试覆盖从 version 1 真实升级、八个并发管理员插入只有一个成功、
  摘要长度/唯一性、正期限、禁止明文令牌列和用户删除后的 session 级联。
- `oasdiff --fail-on WARN` 相对 `main` 报告无破坏性变化。
- `make generate-check` 固定生成客户端与 sqlc 无漂移。

## 明确未完成

- `S1-102`：认证领域服务、并发 setup、Argon2id 参数和密码 verifier。
- `S1-103`：随机会话/CSRF 令牌、Cookie 发放、绝对期限、轮换、撤销与清理。
- `S1-104`：HTTP middleware、全部业务路由默认拒绝、CSRF、防缓存和限流实现。
- `S1-105`：完整认证故障、安全、并发与时间测试。
- `S1-106`：认证 Backend Ready Gate。

在 `S1-106` 前，不得把认证路由、局域网监听或产品登录流程描述为可用。
