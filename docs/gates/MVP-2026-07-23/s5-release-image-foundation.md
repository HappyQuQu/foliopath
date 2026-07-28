# Stage 5 发布镜像基础

## 结论

**Go — Stage 5 已启动，`S5-001A` 发布候选镜像基础完成。**

生产 Vite SPA 现在由 `internal/webassets` 嵌入真实 Go 二进制，并与既有 API、health、
数据库、扫描和媒体处理 composition 在同一进程交付。根目录/前端 history route 由 SPA
owner 处理；`/api`、`/health`、缺失静态资源和非 GET/HEAD 请求继续交给服务端路由，
不会用 `index.html` 掩盖 API 错误。

该结论只建立最终镜像的可重复基础，不表示 `S5-001`、Stage 5、Release Candidate 或稳定
MVP 已完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-001A`
- 需求/质量：`FR-DEP-001～004`、`NFR-SEC-001～002`、`NFR-SAFE-001`、
  `NFR-PRIV-001`
- owner：`internal/webassets` 拥有嵌入 SPA 与静态交付；`internal/app` 只负责 composition；
  根 `Dockerfile` 与 `tests/release/image_smoke.sh` 拥有候选镜像构建和 smoke
- 前序 Gate：[Stage 4 Integrated Done](s4-search-media-integrated-done.md)
- 架构：ADR-0001、ADR-0005、ADR-0009；没有改变部署单元、信任边界或依赖方向

## 已固定证据

- 根 `Dockerfile` 使用固定 digest 的 Node、Go 与 Debian 基础镜像：
  - 锁定 npm 安装并构建 Vite；
  - `CGO_ENABLED=1`、`libvips` build tag 构建真实应用；
  - 运行层包含 CA、时区、libvips 与 FFmpeg；
  - 固定 `USER 65532:65532` 和 readiness healthcheck。
- `.dockerignore` 排除开发依赖、生成输出、原型、QA 图片、数据库和运行状态。
- `internal/webassets` 只嵌入构建时生成的 `dist`；仓库仅跟踪 `.gitkeep`，不提交 Vite
  生成文件。普通 Go checkout 仍可构建，缺少生产 SPA 时安全回落到服务端 404。
- `make test-release-image` 以真实候选镜像验证：
  - `linux/arm64` 构建与版本注入；
  - 只读容器根、`no-new-privileges`、`cap_drop: ALL`；
  - `/library` 只读、`/app/data` 持久可写；
  - SPA 首页和 readiness 由同一进程返回；
  - SIGTERM 在 10 秒窗口内以 0 退出；
  - smoke 前后原媒体 sentinel SHA-256 相同。

2026-07-28 本地实际执行并通过：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
make test-release-image
```

`make test-web-e2e` 的锁定 Chromium 结果为 4 passed、2 个按既有 project 条件 skipped。

## 保留限制与下一 Gate

- 后续 [S5-003](s5-trusted-proxy-security.md) 已完成显式可信 CIDR、严格 HTTPS/host/client
  transport、Secure Cookie、Origin/CSRF、客户端身份限流与安全响应头；非回环应用监听
  只能在该 require-proxy 边界下启用。
- 发布 Compose 与原生双架构网络/媒体矩阵后来已由 `S5-001B/002` 关闭；该基础记录本身
  仍不能替代后续证据。
- 镜像最终 digest、SBOM/漏洞/许可证、容量、最终浏览器矩阵和发布签署仍分别由
  `S5-001`、`S5-005～010` 阻断。
- 当前镜像标签和 OCI description 明确为 Stage 5 candidate；不得发布为稳定版本。
