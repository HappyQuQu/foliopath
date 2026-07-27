# Stage 1 认证前端 Integrated Done

## 结论

**Go — Stage 1 单管理员认证纵向切片 Integrated Done。**

首次管理员设置、登录、会话恢复、退出、会话失效安全返回、公开 readiness 启动故障、
浅/深主题、简体中文/英文和四档响应式状态已经连接真实后端并形成自动浏览器证据。
该结论允许前端进入 Stage 2 媒体库与扫描界面，不表示 FolioPath 已达到发布条件。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-AUTH-001～004`、`FR-UI-001～007`、`FR-DEP-004`、
  `NFR-SEC-002`、`NFR-PRIV-001`、`NFR-OPS-001`
- 后端 Gate：[S1 authentication Backend Ready](s1-auth-backend-ready.md)
- 权威 HTTP 契约：`api/openapi.yaml`
- 前端 route owner：`web/src/routes`
- 认证 feature owner：`web/src/features/auth`
- readiness feature owner：`web/src/features/system`
- 网络 adapter owner：`web/src/lib/api`
- 主题与语言 owner：`web/src/lib/theme`、`web/src/lib/i18n`
- 共享 UI owner：`web/src/components`

## 验收判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 真实 API 集成 | setup、status、session、login、CSRF logout 和 readiness 只经生成 client 与领域 adapter；无 feature raw fetch | 通过 |
| 完整认证旅程 | 一次性 SQLite/空合成媒体根上自动执行 setup → settings → logout → protected route expired return → login | 通过 |
| 安全故障 | readiness safe reason code 映射为阻断恢复页；自动断言不出现宿主路径、`/app/data`、SQLite、堆栈或原始响应 | 通过 |
| 主题与语言 | 浅/深主题、英文浏览器默认、简体中文/英文即时切换、持久化覆盖与 `html[lang]` | 通过 |
| 可访问性 | 语义表单、DOM 顺序、可见焦点、具名状态/通知；Chromium axe serious/critical 为零 | 通过 |
| 响应式 | 390、768、1024、1440px 自动验证，无页面级横向溢出 | 通过 |
| 设计一致性 | `web/design-qa.md` 的 setup、expired login 和 unavailable 同视口并排对照；P1/P2 修复后结果 passed | 通过 |
| CI 固化 | `.github/workflows/ci.yml` 新增独立 Authentication browser E2E，使用锁定 Playwright/Chromium 与一次性真实后端 | 通过 |

## 自动证据

- `web/tests/e2e/auth.spec.ts`
- `tests/e2e/web_auth.sh`
- `web/playwright.config.ts`
- `web/design-qa.md`
- `web/qa/auth-comparison-1440.jpg`
- `web/qa/auth-comparison-mobile.jpg`
- `web/qa/unavailable-comparison-1440.jpg`

本地实际执行并通过：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
npm --prefix web run check
npm --prefix web run build
```

依赖审计保留 `GHSA-qwww-vcr4-c8h2` 的窄例外：该通告只涉及 React Router RSC server
actions，而 FolioPath 是 `BrowserRouter` SPA，不包含 RSC 路由或 server action。
`web/scripts/audit-high.mjs` 在接受该通告前扫描并拒绝 RSC mode 标记；其他 high/critical
发现仍阻断 CI。

## 保留限制

- Stage 5 仍负责最终 Chrome/Firefox/Safari 目标矩阵、可信代理、非回环网络配置、
  Secure Cookie 发布拓扑、最终镜像和发布级视觉回归。
- 当前 Integrated Done 只覆盖认证、基础通用设置和启动 readiness；媒体库、扫描、
  浏览、搜索、预览和查看器仍按 Stage 2～4 Gate 推进。
- 密码修改/找回、多管理员、多用户、匿名 LAN 与外部身份提供商不属于冻结 MVP。

## 交接

- 后端：认证 Backend Ready。
- 前端：认证 Integrated Done。
- 允许的下一步：`S2-201` 媒体库列表与状态，并复用现有 app/provider/design-system owner。
- 评审日期：2026-07-28。
