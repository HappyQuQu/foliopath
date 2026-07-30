# S5-006B Firefox 真实浏览器与 200%/400% 缩放证据

## 结论

2026-07-30，Mozilla 官方 Firefox `153.0.1` 在物理 Mac Studio（Apple M4 Max、
macOS 26.6）上完成真实候选的首次设置、建库、只读扫描、浏览、当前目录过滤、图片预览、
图片 Viewer、真实 MP4 预览/Viewer 播放及键盘操作；随后同一 Firefox 以原生页面缩放
`200%` 和 `400%` 重复扫描状态、浏览、当前目录过滤、预览和 Viewer 的可达性检查。

这份证据关闭 `S5-006B` 的“真实品牌 Firefox 核心纵向链”和“Firefox 真实
200%/400% 缩放”子项。它不代表 VoiceOver/其他读屏、物理触控、移动设备、Safari
真实缩放或最终跨设备视觉签署通过，因此 `S5-006` 仍为 Blocked。

## Firefox 来源与校验

- version：Mozilla Firefox `153.0.1`
- artifact：Mozilla 官方 `mac/en-US/Firefox 153.0.1.dmg`
- size：`154,614,065` bytes
- Mozilla `SHA256SUMS`：
  `e5a7f8f34b16ac5d8d429a1438468f023ad7bf9099fa928db537f45e32159f78`
- local SHA-256：与 Mozilla 值完全一致
- `hdiutil verify`：DMG checksum valid
- `codesign --verify --deep --strict`：通过
- `spctl --assess --type execute`：accepted，`Notarized Developer ID`

Firefox 直接从只读 DMG 运行，使用临时隔离 profile，没有复制到 `/Applications`。
首次运行前经操作者确认接受 Firefox 条款；`Send technical and interaction data to
Mozilla` 已设为关闭，`Automatically send crash reports` 保持关闭，临时密码没有保存。
测试结束后 Firefox 已恢复 `100%`、退出，DMG 已卸载，临时 profile 已移入废纸篓。

## 候选与只读边界

- source commit：`5c3b3c73a1ce32a3777097fb687c707ba914ad41`
- image：`foliopath:s5-supply-chain-5c3b3c7-arm64`
- immutable digest：
  `sha256:8a88d26b6579afea21e4d3d85a1df7b5d45b5f851466c4afd6067d025516457d`
- endpoint：隔离的 loopback `127.0.0.1:4181`
- state：临时 `/app/data`
- media：临时 `/library` 以 `RW=false` 挂载
- fixture：1 张 JPEG、1 个 MP4、2 个目录

完整纵向链结束后，挂载副本与仓库 fixture 的 SHA-256 仍分别一致：

```text
024615e5303b9668f6016916cd25b9d33b91ddec8fe48493ce520fcc0fb0b8f6  kyoto.jpg
ee5e7bbda84c660eabddf53ac9ff5bcaaff88dae8f7ffee9010f9418a0227061  clip.mp4
```

浏览、播放、预览和 Viewer 没有改变原媒体。

## Firefox 100% 核心纵向链

1. 创建临时管理员，拒绝 Firefox 保存密码。
2. 在真实媒体库向导中选择 `/library/travel`；审阅页明确显示只读原媒体和
   `/library` 相对路径。
3. 首次完整扫描完成：2 个目录、2 个媒体、2 个已处理、0 个问题。
4. 打开媒体库根和 `kyoto` 目录，确认目录树、媒体类型、当前目录过滤、布局、排序、
   刷新和两张媒体卡均可达。
5. 当前目录关键字从 `kyoto` 改为 `jpg` 后，URL 持久化为 `?q=jpg`，媒体计数从 2
   收敛到 1，结果只保留 `kyoto.jpg`。
6. 图片预览显示位置、尺寸和大小；完整 Viewer 的关闭、信息、适应窗口、1:1、缩小、
   放大、全屏和相邻导航均可达。实际执行 `I`、1:1、100%→125%→100% 和 `Escape`。
7. MP4 预览和完整 Viewer 均出现 Firefox 原生播放器；实际点击播放，按钮切为
   `Pause`，播放位置从 `0:00 / 0:01` 前进到 `0:01 / 0:01`。

## Firefox 原生 200% 缩放

通过 Firefox 原生缩放快捷键把页面从 `100%` 调到 `200%`。浏览器可访问性树明确报告：

```text
button 200%, Reset zoom level
```

在该状态实际验证：

- 扫描状态页的全局 Header、账户、管理导航、返回入口、完成状态、进度、2/2/2/0
  计数和重新扫描入口均在可访问性树中；
- 浏览页切换为窄宽响应式导航，但当前位置、包含子目录、全部/图片/视频、当前目录过滤、
  布局、排序、刷新和媒体入口仍可达；
- 预览、完整 Viewer、信息面板、1:1、媒体缩放、关闭、相邻导航和 `Escape` 返回仍可达；
- 截图未出现页面级横向溢出或被截断的关键操作。

## Firefox 原生 400% 缩放

在完成 200% 证据后，把同一隔离 Firefox profile 的页面缩放提高到 `400%`。浏览器
可访问性树明确报告：

```text
button 400%, Reset zoom level
```

在该状态实际验证：

- 扫描状态页仍可访问账户、管理导航、返回、完成状态、2/2/2/0 计数和重新扫描入口；
- 浏览页使用窄宽响应式导航，当前位置、包含子目录、媒体类型、当前目录过滤、布局、
  排序、刷新和媒体入口仍可达；
- 在 `kyoto` 目录输入 `jpg` 后，URL 保持 `?q=jpg`，媒体计数从 2 收敛到 1；
- 图片预览、完整 Viewer、`I` 信息面板、1:1、媒体缩放、关闭、相邻导航和 `Escape`
  返回仍可达；
- 截图未出现页面级横向溢出或被截断的关键操作。

## 截图

Firefox 使用只含本次 FolioPath loopback 标签页的隔离 profile；保留浏览器顶栏是为了
让截图同时记录 Firefox 原生 `200%` 或 `400%` 标记。

- [`firefox-200-scan-status.jpg`](firefox-200-scan-status.jpg)：扫描完成状态与
  Firefox `200%` 标记。
- [`firefox-200-browse.jpg`](firefox-200-browse.jpg)：200% 下的响应式浏览工具与媒体。
- [`firefox-200-preview.jpg`](firefox-200-preview.jpg)：200% 下的媒体预览区域。
- [`firefox-200-viewer.jpg`](firefox-200-viewer.jpg)：200% 下的完整图片 Viewer。
- [`firefox-200-viewer-info.jpg`](firefox-200-viewer-info.jpg)：`I` 打开的基本信息面板。
- [`firefox-400-scan-status.jpg`](firefox-400-scan-status.jpg)：400% 下的扫描完成状态。
- [`firefox-400-browse-root.jpg`](firefox-400-browse-root.jpg)：400% 下的根目录响应式浏览。
- [`firefox-400-filter.jpg`](firefox-400-filter.jpg)：400% 下 `?q=jpg` 从 2 项收敛到 1 项。
- [`firefox-400-preview.jpg`](firefox-400-preview.jpg)：400% 下的媒体预览入口。
- [`firefox-400-viewer.jpg`](firefox-400-viewer.jpg)：400% 下的完整图片 Viewer。
- [`firefox-400-viewer-info.jpg`](firefox-400-viewer-info.jpg)：400% 下的基本信息面板。

## 剩余关闭条件

- 在代表性桌面环境完成 VoiceOver/读屏操作签署；
- 在代表性触控与移动物理设备完成输入、布局、媒体解码和视觉签署；
- 完成 Safari 真实缩放及最终跨设备有意视觉差异审阅。

在这些条件完成前，不得把本目录解释为 `S5-006`、`S5-009` 或 Release Candidate 已完成。
