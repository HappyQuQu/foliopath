# FIX-2026-07-29 管理员密码最低长度

- 类型：已批准 MVP 认证 slice 内的例行可用性修复
- 用户决定：管理员密码强度要求过高，降低最低长度
- 关联范围：`FR-AUTH-001～004`
- 目标版本与阶段：MVP / Stage 1 Integrated Done 后的认证可用性修复
- 关联 Gate：S1 Auth Contract Ready、S1 Auth Backend Ready、Stage 1 Integrated Done
- 受影响不变量：无默认密码；密码不记录明文；认证错误不泄露账号；会话与 CSRF 不变
- Owner：`internal/auth` 是服务端密码接受规则唯一 owner；OpenAPI 描述 wire 约束；
  `web/src/features/auth` 只提供一致的即时输入反馈

## 决定

首次管理员密码从至少 12 个 Unicode 字符调整为至少 8 个，最大仍为 128 个，并继续拒绝
控制字符和无效 UTF-8。不要求大小写、数字或特殊字符组合。密码仍使用现有 Argon2id 参数
保存，登录限流、未知账号虚拟校验、错误脱敏、同源 setup 和安全会话全部不变。

已有管理员和已保存 verifier 不需要迁移；该变化只影响今后首次初始化时接受哪些明文输入。
前端按 Unicode code point 计数，与 Go 服务端的 rune 计数对齐，避免 emoji 或非 BMP 字符
在两端得到不同长度。

## 回归证据

- `internal/auth/identity_test.go`：7 字符拒绝，8 个 ASCII 与 8 个中文字符接受，控制字符、
  无效 UTF-8 和超过 128 字符继续拒绝；
- `web/src/routes/AppRouter.test.tsx`：7 个中文字符显示本地化错误且不请求 setup，补到
  8 个后提交；
- `web/tests/e2e/auth.spec.ts`：真实 Go/SQLite/HTTP 与 Chromium 纵向链使用恰好 8 字符
  的管理员密码完成 setup、后续登录和受保护业务流程；
- `api/openapi.yaml`：`SetupAdministratorRequest.password.minLength=8`，生成 client
  由权威源更新；
- 认证 service、HTTP、Argon2id、session、CSRF 和限流回归继续通过。
