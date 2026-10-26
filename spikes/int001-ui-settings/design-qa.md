# Design QA — 智能设置方案一

- source visual truth: `evidence/selected-design.png`（第一张显示的 ImageGen 结果）
- implementation: http://127.0.0.1:6010/
- implementation screenshot: `evidence/desktop-final.png`
- viewport: 1487 × 1058 CSS px；source/implementation 均为 1487 × 1058 pixels，DPR 1。
- state: zh-CN、light、connected example、family selected、semantic on、face off、details closed。
- full-view comparison: 同一个工具输入同时打开源图与最终截图；不是仅从代码推断。
  采用 fullPage 截图解决浏览器 viewport 截图的缩放模糊；本页高度与目标视口一致。
- focused comparison: 全宽截图中模型名称、开关、说明和页脚可直接阅读，无额外裁剪需求。
- additional evidence: `evidence/mobile.png`（390 × 844 CSS 视口，完整滚动页）；
  `evidence/dark-en.png`（1487 × 1058，英文深色）。移动端 DOM 检查无横向溢出。

## Findings and comparison history

1. 初次渲染发现缺少 ToastProvider 和品牌静态资源。已加入规范 provider，并把 publicDir
   指向现有 `web/public`；没有重画品牌图标。最终截图标志正确，首屏不再空白。
2. 首轮对照 `evidence/desktop.png`：[P2] 模型列偏右、标题与首节间距偏紧、媒体库行偏高。
   调整列比例、顶部留白和行间距；重新截图 `desktop-final.png` 并再次与源图联合比较。
   主要区域顺序/比例、首屏部署详情与页脚均保留。无未解决 P0/P1/P2。
3. 可接受偏差：沿用仓库原有字号、按钮/开关尺寸、蓝色色值、全局页头宽度和侧栏比例，
   不为生成图复制第二套共享组件。源图混用模型的文案已纠正；新增当前媒体库选择是
   必要的作用范围说明，不引入全局批量开启行为。

## Required fidelity surfaces

- Typography：SF/PingFang/system 字体，中央字号和行高；正文清晰，标题/说明/辅助信息层次保留。
- Spacing/layout：同一视口对照，分区顺序/单一主按钮/分隔线与留白保留；窄屏按列折叠，无隐藏控制。
- Colors/tokens：仅消费现有 canvas/surface/accent/text/border 语义色；深色主题未建立第二套机制。
- Assets：复用 FolioPath 品牌 SVG 资源与 Phosphor 图标库；无手写图标或 CSS 图画替代。
- Copy：保留中文主流程，中英切换可用；预览标签始终可见，部署详情明确当前主应用尚未接入。

## Browser interaction checks

- 查看分析进度打开规范 Dialog；停止示例任务更新状态且保留 1,240 / 2,490。
- Escape 关闭后焦点恢复到“查看分析进度”。
- 切换旅行记录并开启智能搜索，再回家庭影像：能力状态按库隔离。
- 展开部署详情、切到不可用场景：开关禁用、警告可见、已有进度保留。
- 英文与深色切换、移动端重排；首屏和滚动区域检查。
- 控制台曾记录初始化 provider 错误和修改时的 HMR 错误；修复并重新载入后无新增运行时错误。

## Follow-up polish / boundaries

Executed checks: prototype `npm run build`、repository `make arch-check`、`git diff --check` 通过。
本轮未运行全量 Go/前端组件/E2E 测试；交互验证为上述真实浏览器操作，不冒称自动化覆盖。

- [P3] 原型采用规范组件，未逐像素复制生成图的字体/壳层尺寸。
- 非完整无障碍审核/屏幕阅读器测试；未实现全局搜索及其他管理页。
- 无真实服务接入，未发布；转正式 UI 仍需后端合同和 Backend Ready。

final result: passed
