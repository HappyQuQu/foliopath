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
结果。按操作者决定，本轮不等待计费阻断的 amd64 CI；真实 Firefox、读屏、缩放、触摸
和移动物理设备仍由 `S5-006B` 阻断。

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

## 剩余阻断

`S5-006B` 和 Release Candidate 仍要求：

1. Chrome 150 stable matrix 与 Safari 26.5.2 真机结果已经建立；仍须在最终承诺的真实
   Firefox 版本重复核心纵向链；
2. Chrome 150 forced-colors 自动化和三引擎长列表滚动已通过；仍须在代表性桌面/移动物理
   设备验证缩放、读屏、触摸与媒体解码；
3. 将最终视觉差异作为有意设计变更评审，不以无条件更新 snapshot 消除失败。

在这些条件完成前，不得把 `S5-006`、`S5-009` 或 Release Candidate 标为完成。
