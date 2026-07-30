# S5-006 浏览器质量候选 Gate

## 结论

**Conditional Go — `S5-006A` 自动化矩阵已建立并在本机通过；`S5-006` 尚未完成。**

锁定的 Playwright 1.61.1 现在分别运行日常 Chromium 产品纵向链，以及 Firefox、WebKit
共享的查看器稳定状态矩阵。Linux Chromium 另有固定 1280×800、英文、深色、
reduced-motion 的离线查看器视觉基线。2026-07-28 本机 Firefox 151、WebKit 26.5 和
Chromium 149 功能矩阵通过；同一固定 Playwright Ubuntu 容器中的视觉生成后独立复跑通过。
Safari 26.5.2 已在本机物理 Mac 上通过真实候选的登录、建库、只读目录选择、首次扫描和
状态页纵向链。已安装的 Google Chrome 150 又以隔离测试 profile 通过完整 desktop media
matrix；日常 Chrome profile 的 1Password 注入仍会中断 UI 控制，但不影响产品自动化
结果。2026-07-30 又把 200% 缩放的等效重排护栏加入全部桌面项目并在本机通过。按操作者
决定，本轮不等待计费阻断的 amd64 CI；真实品牌 Firefox、物理浏览器缩放、读屏、触摸和
移动物理设备仍由 `S5-006B` 阻断。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-006A`、`S5-006B`
- 需求/质量：`NFR-ACC-001`、`NFR-COMP-001`，以及已冻结的
  `FR-MED-001～008`
- owner：前端负责人拥有稳定状态与视觉基线；QA/发布负责人拥有最终浏览器和物理设备矩阵
- 合同：`web/playwright.config.ts`、`web/tests/e2e/media-matrix.spec.ts`、
  `web/tests/e2e/visual-regression.spec.ts`、`tests/e2e/web_auth.sh`、
  `.github/workflows/ci.yml`
- 风险：R-014、R-015、R-016
- 架构影响：没有改变共享前端架构、部署单元、持久化或信任边界，不需要 ADR

## 自动化边界

日常 `make test-web-e2e` 保持一次性真实后端的 Chromium/Pixel 5 纵向链，避免多个浏览器
依次争用“首次创建唯一管理员”的有状态 fixture。新增 `make test-web-release-e2e` 在完全
模拟的 API 边界执行：

- Firefox 与 WebKit：查看器初始焦点、信息面板键盘切换、Escape 返回、unsupported
  codec、offline、deleted、横向溢出和 axe serious/critical；
- Chromium：同一状态矩阵，并继续独占真实 MP4 `Range`/`206` 播放与 React Router
  瞬时序列导航断言；
- 所有桌面项目：以 `640×400` 有效 CSS 视口模拟 `1280×800` 桌面 200% 浏览器缩放，
  验证媒体卡真实焦点入口、查看器主焦点、缩放/信息/关闭控件、无页面横向溢出和 axe
  serious/critical 为零；该确定性代理不冒充物理浏览器缩放签署；
- visual Chromium：固定 viewport/theme/locale/reduced-motion 后精确比较
  `offline-viewer-dark.png`，覆盖顶栏、焦点环、恢复卡、信息面板、相邻导航和底部提示。

真实 MP4 播放没有强行跨引擎断言：系统 codec 能力不是 FolioPath 的规范化输出。产品拥有
的 unsupported/offline/deleted 呈现则由三个引擎共同验证。Firefox 不保证测试直接写入的
React Router `history.state.usr` 在 reload 后保留，因此这种内部状态注入只留在 Chromium；
公开可观察的键盘与退出行为仍由三个引擎共同覆盖。

CI 在 Ubuntu 24.04 安装 lockfile 对应的 Chromium、Firefox 和 WebKit，顺序运行两套入口，
并在失败时保留 HTML report、截图、trace 和 video。视觉基线由官方
`mcr.microsoft.com/playwright:v1.61.1-noble` 环境生成及独立复跑，避免把 macOS 字体
栅格结果提交为 Linux 基线；非 Linux 本机执行发布矩阵时明确跳过这一个平台所有的精确
像素比较，Firefox/WebKit 功能矩阵仍执行。

## 2026-07-28 本机证据

| 检查 | 结果 |
| --- | --- |
| `npm --prefix web run check:types` | 通过 |
| Playwright project enumeration | 15 个用例、7 个 project，边界符合预期 |
| Firefox 151 / WebKit 26.5 desktop matrix | 各 1 通过；各 1 个移动专用用例按预期跳过 |
| Chromium 149 desktop media matrix | 1 通过；移动专用用例按预期跳过 |
| Linux visual baseline generation | 1 通过并生成 1280×800 PNG |
| 独立 Linux visual comparison | 1 通过 |
| `make web-check` | 首次 1 个既有路由测试偶发失败；单文件与完整第二次运行均通过（25 文件、75 测试） |
| `make arch-check generate-check lint test test-race test-integration` | 通过 |
| `make test-web-e2e` | Chromium/Pixel 5 共 4 通过、2 个非适用项目用例跳过 |
| `make test-web-release-e2e`（macOS） | Firefox/WebKit 各 1 通过；2 个移动专用与 1 个 Linux 视觉用例按预期跳过 |
| Safari 26.5.2 / 物理 Mac | 登录、建库、`/library` 只读选择、9 项首次扫描和状态页通过；0 个扫描问题 |
| Google Chrome 150.0.7871.187 / 物理 Mac | `make test-web-chrome-stable` 使用已安装 branded Chrome 与隔离 profile：普通及 forced-colors desktop media matrix 各 1 通过，两个移动专用用例按预期跳过 |
| 日常 Chrome profile | 页面打开；1Password 注入期间 UI 控制连接两次中断，不作为产品失败或通过证据 |

## 2026-07-30 200% 等效重排证据

新增 `200 percent equivalent desktop reflow keeps browse and viewer controls reachable`
后，本机实际执行：

| 检查 | 结果 |
| --- | --- |
| `make test-web-release-e2e` | Firefox、WebKit 的等效重排用例通过；完整命令 6 passed / 13 applicable skips |
| `make test-web-e2e` | Chromium 等效重排及既有真实后端纵向链通过；完整命令 7 passed / 4 applicable skips |
| `make test-web-chrome-stable` | 品牌 Chrome Stable 与 forced-colors 的等效重排用例均通过；完整命令 6 passed / 2 applicable skips |
| 断言范围 | 媒体卡焦点入口、查看器主焦点、缩放/信息/关闭控件、无横向溢出、axe serious/critical 为零 |

同日尝试从 Mozilla 官方 Firefox 153.0.1 发布目录取得品牌版 Firefox；当前网络连接反复
断开，三次续传只到约 6%，没有形成可校验应用，因此不产生真实 Firefox 通过或失败证据。
Playwright Firefox 结果继续只记作引擎自动化，不冒充品牌版物理浏览器。

## 剩余阻断

`S5-006B` 和 Release Candidate 仍要求：

1. Chrome 150 stable matrix 与 Safari 26.5.2 真机结果已经建立；仍须在最终承诺的真实
   Firefox 版本重复核心纵向链；
2. Chrome 150 forced-colors、三引擎长列表滚动和五个桌面项目的 200% 等效重排已通过；
   仍须在代表性桌面/移动物理设备验证真实浏览器缩放、读屏、触摸与媒体解码；
3. 将最终视觉差异作为有意设计变更评审，不以无条件更新 snapshot 消除失败。

在这些条件完成前，不得把 `S5-006`、`S5-009` 或 Release Candidate 标为完成。
