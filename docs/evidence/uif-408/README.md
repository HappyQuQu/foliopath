# UIF-408 Integrated Slice evidence

## 结论

`FTR-UIF-001` 的 `UIF-AC-001～012` 已有实际证据，`UIF-S4 Integrated Slice Done`
判定为 **Go**。该结论只关闭生产前端原型一致性 feature，不等于 MVP Release Candidate
可发布；Stage 5 的最终 digest、物理辅助功能和供应链阻断保持不变。

## 四档逐页视觉复核

2026-07-30 使用 Codex 内置浏览器，在同一简体中文、深色主题和对应功能状态下，分别捕获
最新原型与当前生产 React 路由：

| CSS viewport | 页面数 | 原型图 | 生产图 | 组合审阅图 | 横向溢出 | 结论 |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| `390 × 844` | 12 | 12 | 12 | 3 | 无 | 通过 |
| `768 × 1024` | 12 | 12 | 12 | 3 | 无 | 通过 |
| `1265 × 800` | 12 | 12 | 12 | 3 | 无 | 通过 |
| `1440 × 900` | 12 | 12 | 12 | 3 | 无 | 通过 |

共保存 48 张原型图、48 张生产图和 12 张成对审阅图。页面集合为 `account`、
`auth-login`、`auth-setup`、`browse`、`general`、`libraries`、`library-new`、
`library-status`、`search`、`storage`、`viewer`、`welcome`。可打开的组合入口为
[`visual/index.html`](visual/index.html)，原始证据分别在
[`visual/source`](visual/source/)与[`visual/implementation`](visual/implementation/)；
人工审阅视口分段保存在[`visual/review`](visual/review/)。

生产截图来自当前 Go 后端和当前 Vite 前端，不是 Storybook 或 mock：在临时只读
`/library` 中完成管理员初始化、建库、真实扫描、目录浏览、Search、预览、Viewer、设置、
缓存摘要、账户、退出和重新登录。媒体卡片及 Viewer 中的白色小画面是 synthetic JPEG
夹具内容本身，不是生产 UI 的弹窗或漂移。测试没有读取或修改开发者真实媒体。

逐组审阅没有发现 P0/P1/P2 或需延期的 P3。动态业务数量和 synthetic 媒体内容不会与静态
原型逐字一致，但全局 Header、管理侧栏/移动分类、独立路由、Browse 工具栏、右侧过滤、
单滚动容器、Search 无侧栏、预览/Viewer 和四档响应式层级保持同一设计合同。

这组证据补齐了此前未完成的断点逐页矩阵：`UIF-401` 是 12 页共享
`1280 × 720` 对照，`UIF-317` 是双语/双主题的四档共享状态矩阵；二者都没有被冒充为本次
12 页 × 4 断点的 96 张原始截图。

## 验收映射

| 验收 | 实际证据 |
| --- | --- |
| `UIF-AC-001～003` | UIF-401/402 的壳与稳定基线；UIF-403 的真实管理、路由和操作链；本次 12×4 对照 |
| `UIF-AC-004～006` | UIF-S2 的账户/目录 q/cache 后端证据；UIF-403 的 generated-client 真实链与媒体不变清单 |
| `UIF-AC-007～009` | UIF-312/313、UIF-401/402；本次四档逐页、无横向溢出和人工成对审阅 |
| `UIF-AC-010` | UIF-317/404 的语言、主题、三引擎、axe、键盘、触摸、forced-colors、reduced-motion；物理签署仍归 S5-006B |
| `UIF-AC-011` | UIF-405 的三引擎 100k 有界 DOM/FPS/RSS 与后端 10k/100k 并发 |
| `UIF-AC-012` | UIF-403 的路径/SHA-256 不变；UIF-406 完整仓库验证 |

## 受影响 Stage 5 复验

在 UIF-408 证据完成后实际执行：

```text
make test-web-release-e2e
# 4 passed / 13 applicable or platform skips

make test-web-chrome-stable
# 4 passed / 2 applicable skips

make test-browser-capacity
# Chromium: 59.980 FPS, P95 18.20ms, mounted 60, RSS 724451328
# Firefox:  58.797 FPS, P95 18.64ms, mounted 60, RSS 1349222400
# WebKit:   59.964 FPS, P95 21.00ms, mounted 60, RSS 118243328

make test-e2e
# application container smoke passed

make release-docs-check
make release-readiness-check
# passed: the documented No-Go state is internally consistent

make release-ready
# expected non-zero: unresolved Stage 5 gates and risks remain
```

`make release-ready` 的失败是当前失败关闭合同的正确结果，不是 UIF feature 测试失败。
真实 Firefox、读屏、200%/400% 缩放、物理 OS 高对比和代表性触摸/移动设备仍由
`S5-006B` 签署；自动化结果没有冒充这些物理证据。

## 环境说明

首次临时产品镜像构建在 Go 模块下载阶段两次遇到远端代理 `unexpected EOF`，尚未进入产品
断言。随后只读复用主机 Go module cache 构建相同当前源码，真实浏览器链与上述验证全部
通过。该瞬时基础设施事件没有被记录为产品通过或产品缺陷。
