# Gate MVP-2026-07-23 / UIF-S3 / Consumer/UI Ready

- 日期：2026-07-30
- 目标版本：`MVP-2026-07-23` revision 4
- Feature：[FTR-UIF-001](../../features/frontend-prototype-fidelity.md)
- 前置：[UIF-S2 Backend Evidence Ready](uif-s2-backend-evidence-ready.md)
- 任务：`UIF-301～318`
- 需求：`FR-BRW-010`、`FR-UI-010`、`NFR-UIF-001`，复用
  `FR-AUTH-005`、`FR-UI-009`、`FR-UI-001～007`、`NFR-ACC-001`
- 决策角色：产品用户 / FolioPath maintainers / auth、catalog、thumbnail、web owners
- 结论：**Go — 授权 UIF-401～408 进入 Integrated Slice**

本 Gate 只确认生产消费者、共享 UI、状态和输入矩阵已准备好进入完整纵向集成。它不代表
逐页原型比较全部完成，不代表 MVP/RC 可发布，也不解除 `UIF-S4` 和 Stage 5 Gate。

## 消费者与唯一所有权

| Gate 条件 | 已接受实现与证据 |
| --- | --- |
| 只消费已评审合同 | Account、directory `q`、settings/cache summary 与 cleanup 只通过生成 client 和手写 domain adapter；页面没有手写 wire DTO 或 mock 成功分支 |
| server/URL state | TanStack Query 拥有 server state；Browse/Search URL codec 拥有 scope、query、sort 和 cursor 复位；query-key/invalidation 由各 feature 唯一 owner 提供 |
| 分页与虚拟化 | 目录和媒体使用稳定 cursor；`MediaCollection` 使用 TanStack Virtual，100k fixture 只挂载 60 项，不加载或渲染无界列表 |
| 共享壳与导航 | `GlobalHeader`、canonical `AppShell`（Browse shell）和 `ManagementShell` 分别唯一拥有全局入口、目录抽屉和管理分类；普通认证页面只有一个 Header |
| 独立生产页面 | setup/login、Browse、Search、Viewer、General、Libraries、New Library、Scan Status、Storage 和 Account 均由真实路由恢复；首次空态进入真实媒体库创建流程 |
| 状态与错误 | `AsyncState`、`InlineStatus`、`Button loading`、`Toast` 与真实扫描/cache/media 状态覆盖 loading/empty/offline/error/conflict/cancel/pending/success；安全错误不被改写为空或假成功 |

## Reference manifest 与视觉证据

`web/qa/visual-reference-manifest.json` 固定 12 个生产页面到
`prototypes/apple-redesign` HTML、生产路由、适用状态和确定性
Storybook/component/E2E fixture 的映射，并固定：

- `390×844`、`768×1024`、`1265×800`、`1440×900`；
- `zh-CN` / `en`；
- `light` / `dark`。

`npm run check:visual-references` 已进入 `npm run check`，会对缺页、顺序、source、fixture
或矩阵漂移失败关闭。任务中心和维护原型不在本 feature manifest 中。

同状态可视证据：

- [UIF-301 管理壳](../../evidence/uif-301/)
- [UIF-312 Browse 工具栏与移动过滤](../../evidence/uif-312/)
- [UIF-315 Viewer](../../evidence/uif-315/)
- [UIF-316 八类状态](../../evidence/uif-316/)
- [UIF-317 语言/主题/视口矩阵](../../evidence/uif-317/README.md)
- [最终 Design QA](../../../design-qa.md)：`final result: passed`

## 交互、可访问性与恢复

- 全局搜索、管理菜单、目录抽屉、对话框、预览和 Viewer 均使用语义控件与明确焦点返回；
- Pixel 5 touch profile 不依赖 hover；Firefox/WebKit/Chromium、Chrome Stable 和
  forced-colors 均运行相同媒体状态与恢复矩阵；
- `lang`、resolved theme、serious/critical axe、页面 overflow 和 reduced-motion
  `1ms` token 由真实 General settings 页面矩阵检查；
- 离线、取消、失败和后续分页失败保留可靠索引或已加载内容；session 过期回到登录；
- Storage 保存有同步 in-flight guard，双击只产生一个 PATCH；cache cleanup 继续使用
  CSRF 与 Idempotency-Key；
- UI 文案只暴露 library-relative path，不暴露 host path、SQL、stack、密码、Cookie 或
  CSRF secret；原媒体没有任何写操作入口。

## 实际执行的验证

本 Gate 在 2026-07-30 成功执行：

```text
make web-check
make test-browser-capacity
make test-web-e2e
make test-web-release-e2e
make test-web-chrome-stable
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make openapi-lint
make release-docs-check
git diff --check
```

结果：

- 前端 reference manifest：12 pages / 4 viewports / 2 locales / 2 themes；
- Vitest：31 files / 122 tests passed；Storybook production build passed；
- Chromium + mobile：6 passed / 3 skipped；
- Firefox + WebKit + Linux visual：4 passed / 5 skipped；
- Chrome Stable + forced-colors：4 passed / 2 skipped；
- 100k 三引擎容量：mounted items 均为 60；Chromium 59.996 FPS / 727,957,504 B，
  Firefox 56 FPS / 1,357,824,000 B，WebKit 59.952 FPS / 118,571,008 B；全部满足
  ≥45 FPS、P95 ≤34ms、RSS ≤1.5GiB、mounted ≤64 的预算。
- 格式、架构、生成漂移、Go vet/unit/integration、容器 runtime smoke、发布文档与 diff
  检查全部通过；OpenAPI lint 通过并只保留既有 health endpoints 缺少 4xx 的两条 warning。

## 残余边界与下一授权

- `UIF-401` 仍须完成每个 manifest 页面在四档适用状态下的原型/生产同输入比较；
- `UIF-402` 仍须把 Header、管理四页、Browse、Search、预览和 Viewer 扩入
  Linux-owned 视觉回归；
- `UIF-403～406` 仍须完成完整真实纵向链、三浏览器异常矩阵、候选容量与全仓检查；
- `UIF-407` 收敛 PRD/流程/安全/部署/README，`UIF-408` 才能签署 Integrated Slice；
- 在 `UIF-S4` 前不得对外宣称“全部页面与原型一致”或“可发布”。

## 结论

生产前端已具备可审计 reference manifest、唯一共享壳、真实合同消费者、稳定状态、四档
双语双主题和键盘/触摸/多浏览器证据。`UIF-S3 Consumer/UI Ready` 为 Go，下一步进入
`UIF-401～408`；其余发布阻断保持不变。
