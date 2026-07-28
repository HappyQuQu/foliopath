# S5-003 可信代理与发布 HTTP 安全

## 结论

**Go — `S5-003` 可信代理、Secure Cookie、Origin/CSRF、认证限流与安全响应头完成。**

FolioPath 现在允许在显式可信代理后使用非回环容器监听，同时把公共 HTTPS transport、
authority 和客户端身份收口到 `internal/api` 的单一请求上下文。业务 handler、认证、
Cookie、Origin 和限流不直接解析代理头，也不复制信任判断。

该 Gate 不表示 Compose、双架构候选镜像、RC 或稳定 MVP 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-003`
- 需求/质量：`FR-AUTH-001～004`、`FR-DEP-001～004`、`NFR-SEC-001～002`、
  `NFR-PRIV-001`
- transport owner：`internal/api/transport.go`
- 配置 owner：`internal/app/config.go`
- 认证/CSRF owner：`internal/auth` 与既有 `internal/api` middleware
- 风险：R-010、R-011、R-016
- 前序 Gate：[认证 Backend Ready](s1-auth-backend-ready.md)、
  [认证 Integrated Done](s1-auth-integrated-done.md)、
  [Stage 5 镜像基础](s5-release-image-foundation.md)

## 冻结契约

- 默认仍监听 `127.0.0.1:8080`，不配置代理时行为不变。
- `FOLIOPATH_LISTEN` 只接受数值 IP 和 `1～65535` 端口。
- 非回环监听必须同时设置 `FOLIOPATH_TRUSTED_PROXIES`；值为逗号分隔、非重复的 IP
  CIDR。拒绝主机名、裸 IP、重复/空项、universal `/0` 和 IPv4-mapped IPv6 CIDR。
- 只有直连 peer IP 命中显式 CIDR，才读取一组单值：
  `X-Forwarded-Proto: https`、合法 `X-Forwarded-Host` 和单一数值 IP
  `X-Forwarded-For`。
- 第一稳定版只支持单跳代理。逗号链、多 header、缺失字段、`Forwarded`/`X-Forwarded-*`
  混用、非 HTTPS、非法 authority 或客户端 IP全部返回统一脱敏 `400`。
- 未受信 peer 的全部转发头在进入应用前清除。非回环模式下，非 loopback 且未受信的直连
  请求返回 `trusted_proxy_required`；loopback healthcheck 保持可用。
- setup/login 的 Origin、Cookie Secure、logout 清理 Cookie、HSTS 和认证限流只消费验证后
  transport；客户端不能靠自带转发头改变这些判断。
- 所有响应统一增加 CSP、`frame-ancestors 'none'`、`X-Frame-Options: DENY`、
  `nosniff`、`Referrer-Policy: no-referrer` 和禁用相机/位置/麦克风的
  Permissions Policy。只有验证后的 HTTPS 响应增加一年 HSTS，不声明
  `includeSubDomains` 或 preload。

## 验收证据

- 未受信 peer 的伪造转发头被清除，Origin/限流仍使用直连 transport。
- 受信 peer 的严格 HTTPS/host/client 三元组形成唯一请求上下文。
- 缺失、HTTP、链式 XFF 和混用标准 `Forwarded` 全部在 handler 前失败。
- 非回环 `require proxy` 拒绝远程直连，同时保留 loopback health。
- 真实认证路由通过受信代理完成 HTTPS Origin login，返回 Secure Cookie、HSTS、CSP 和
  deny-frame headers。
- 同一受信客户端第 11 次 login 返回 429；同一代理后的另一客户端首个请求正常，证明
  限流没有退化为“所有用户共享代理 IP”。
- 既有 session-bound CSRF、绝对过期、logout 撤销和回环 HTTP 测试保持通过。

2026-07-28 本地实际执行并通过：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
make test-web-e2e
make test-release-image
```

`make test-web-e2e` 为 4 passed、2 个按既有 project 条件 skipped；候选镜像 smoke 在
本机 linux/arm64 Docker runtime 通过。原生 linux/amd64 与 CI 结果仍归 `S5-002`。

## 保留限制

- 部署者必须让应用端口只对代理或本机开放；CIDR 配置不能替代宿主机 firewall/端口绑定。
- 不支持多跳代理链、CDN 动态地址发现、主机名 trust、PROXY protocol 或客户端证书。
- 当前应用不终止公网 TLS；外部代理负责证书、TLS policy、Range/streaming 和取消传播。
- Compose 和候选镜像实际代理拓扑由 `S5-001B/S5-002` 固定；最终浏览器矩阵由
  `S5-006` 阻断。
