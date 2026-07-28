# CR-2026-002：经认证的局域网 HTTP

## 状态

- 状态：Confirmed
- 变更等级：C3
- 目标版本：MVP-2026-07-23
- Scope revision：Frozen r2；替代 r1
- Change Record ID / 基线事件：CR-2026-002 / 2026-07-28 产品确认
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/app` 配置与 `internal/api` transport

## 用户问题与价值

FolioPath 的主要部署环境是家庭服务器与受信局域网。强制每个部署者先维护 TLS 反向代理，
违背单容器、单端口和低维护成本目标。应用已经内建首次设置、单管理员认证、会话、CSRF 与
退出登录，因此局域网 HTTP 不等于匿名 LAN。

## 范围与决定

- 新增 `FR-DEP-005`，目标版本仍为 MVP，Stage 5。
- 根 Compose 默认把端口发布到所有宿主 IPv4 接口；可用
  `FOLIOPATH_BIND_ADDRESS=127.0.0.1` 收窄。
- 非回环监听不再要求可信代理配置。未受信请求的转发头仍全部清除，限流身份使用直连 peer。
- `FOLIOPATH_TRUSTED_PROXIES` 继续作为外部 HTTPS 代理的显式可选配置；配置后保持严格
  单跳 HTTPS 头校验和代理旁路拒绝。
- 单管理员认证、CSRF、SameSite/HttpOnly Cookie 与业务 API 默认认证保持不变；不增加匿名模式。
- 公网或其他不可信网络暴露、TLS 证书与反向代理属于部署者职责，不增加 FolioPath 部署单元。

## 风险、验证与 Gate

- HTTP 不提供链路机密性；仅适用于部署者确认的受信 LAN。公网暴露必须由外部 TLS/ACL 保护。
- 继续跟踪 `R-010`，新增直接 LAN 配置、伪造转发头清除、HTTP Origin/CSRF、认证 Cookie 和
  Compose smoke 回归。
- 需要重跑 `make arch-check`、Go 测试、集成测试、Compose smoke 与 Stage 5 发布文档检查。
- 架构决定由 [ADR-0010](../adr/0010-authenticated-lan-http.md) 固定。
