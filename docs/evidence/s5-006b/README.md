# S5-006B Chrome 200% 物理浏览器证据

## 结论

2026-07-30，Google Chrome `151.0.7922.71` 在物理 Mac Studio（Apple M4 Max、
macOS 26.6）上，以浏览器原生页面缩放 `200%` 完成候选镜像的扫描状态、目录浏览、
媒体网格、预览和完整查看器纵向链。Chrome 工具栏的可访问性树明确报告
`缩放比例：200%`；本次不是修改 viewport 或 CSS 的等效模拟。

这份证据只关闭 Chrome 桌面真实缩放子项。它不代表真实品牌 Firefox、VoiceOver 或其他
读屏、物理触控、移动设备、Safari 200% 缩放或最终视觉签署通过，因此 `S5-006B` 仍为
Blocked。

## 候选与隔离边界

- source commit：`5c3b3c73a1ce32a3777097fb687c707ba914ad41`
- image：`foliopath:s5-supply-chain-5c3b3c7-arm64`
- immutable digest：
  `sha256:8a88d26b6579afea21e4d3d85a1df7b5d45b5f851466c4afd6067d025516457d`
- browser：Google Chrome `151.0.7922.71`
- host：Mac Studio / Apple M4 Max / macOS 26.6
- endpoint：隔离的 loopback `127.0.0.1:4180`
- state：临时 `/app/data`，临时 `/library` 以 `RW=false` 挂载
- fixture：1 张 JPEG、1 个 MP4、2 个目录；完整扫描结果为 2 目录、2 媒体、
  2 已处理、0 问题

测试媒体副本与仓库 fixture 的 SHA-256 分别一致：

```text
024615e5303b9668f6016916cd25b9d33b91ddec8fe48493ce520fcc0fb0b8f6  kyoto.jpg
ee5e7bbda84c660eabddf53ac9ff5bcaaff88dae8f7ffee9010f9418a0227061  clip.mp4
```

测试结束时再次读取挂载副本，哈希未变。原媒体只读不变量没有因浏览、预览或查看器操作而
改变。

## 实际步骤与结果

1. 通过真实首次设置创建临时管理员，通过真实媒体库页面选择
   `/library/travel` 并完成首次扫描。
2. 使用 Chrome 原生缩放把页面调到 `200%`，确认工具栏可访问性树报告
   `缩放比例：200%`。
3. 在扫描状态页确认 Header、全局搜索、账户菜单、管理导航、返回媒体库、完成状态、
   进度、目录/媒体/问题计数和重新扫描入口均可达。
4. 在媒体库根目录确认目录树、当前位置、包含子目录、全部/图片/视频、当前目录关键字
   过滤、布局切换、排序、刷新和子目录卡片均可达，无页面级横向溢出。
5. 进入媒体目录，确认 JPEG/MP4 卡片和预览入口可达；打开 JPEG 预览后确认固定预览、
   关闭、上一项/下一项、打开完整查看器和基础信息均可达。
6. 打开完整查看器，确认关闭、信息、适应窗口、1:1、缩小、放大、全屏和相邻导航均可达。
   实际触发 `I` 打开信息面板，实际切换 1:1，并把媒体缩放从 100% 调到 125% 再返回
   100%；实际按 `Escape` 返回媒体目录。
7. 将 Chrome 页面缩放恢复为 `100%`，关闭隔离测试标签页，保留用户原有标签页不变。

## 截图

截图已裁掉浏览器标签页、地址栏、扩展和书签，只保留 FolioPath 页面，避免把与候选无关的
个人浏览器状态提交到仓库。原生缩放倍率由测试期间 Chrome 可访问性树返回的
`按钮 缩放比例：200%` 记录确认。

- [`chrome-200-scan-status.jpg`](chrome-200-scan-status.jpg)：扫描状态页。
- [`chrome-200-browse-root.jpg`](chrome-200-browse-root.jpg)：媒体库根目录与完整浏览工具。
- [`chrome-200-media-grid.jpg`](chrome-200-media-grid.jpg)：两种媒体卡片与预览入口。
- [`chrome-200-viewer.jpg`](chrome-200-viewer.jpg)：完整查看器及全部主控件。
- [`chrome-200-viewer-info.jpg`](chrome-200-viewer-info.jpg)：`I` 快捷键打开的基本信息面板。

## 未覆盖与关闭条件

- 取得可校验的真实品牌 Firefox，重复核心纵向链；
- 在代表性桌面环境完成 VoiceOver/读屏操作签署；
- 在代表性触控与移动物理设备完成输入、布局、解码和视觉签署；
- 审阅 Safari/Firefox 的真实缩放表现和最终有意视觉差异。

在上述条件完成前，不得将本目录的 Chrome 子证据解释为 `S5-006`、`S5-009` 或 Release
Candidate 已完成。
