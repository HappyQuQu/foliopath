# 浏览器拥有视频播放能力判断

## 状态

- 状态：Confirmed
- 类型：既有 `FR-MED-004～006` / Stage 4 媒体策略例行修复
- 目标版本：MVP maintenance
- 日期：2026-08-04
- Owner：`internal/media` 探测事实；`web/src/lib/media/availability.ts` 与共享媒体查看组件

## 问题

服务端曾只把 MP4/MOV 中的 H.264 标为 `playable`，并把其他成功探测的视频编码标为
`unsupported_codec`。前端据此在创建原生 `<video>` 前显示不兼容状态。这把跨浏览器、操作
系统和设备变化的解码能力错误地固化为服务端事实，导致当前浏览器实际可以直接播放的原文件
也无法尝试播放。

## 决定

- 成功探测的 MP4/MOV H.264 继续作为保守的 `playable` 快速路径。
- 其他成功探测的视频返回 `unknown`，不再由服务端宣称当前浏览器不能播放。
- 前端不使用遗留的 `unsupported_codec` 派生值阻止播放器挂载；这也让已有索引无需 migration
  或重新扫描即可恢复尝试。
- 浏览和搜索共享的 `MediaPreview`、完整 `MediaViewer` 始终先挂载认证原内容。只有原生
  `<video>` 报错后才显示播放失败，并允许对当前媒体重试。
- 保持原内容只读、认证 Range、不转码、不 remux、不新增下载能力。

## 影响与证据

- 无 API/schema、migration、部署、信任边界或依赖方向变化；`PlaybackStatus` 枚举为兼容旧
  数据继续保留。
- Go 单元/FFmpeg integration 固定非 H.264 成功探测为 `unknown`。
- availability 单元测试固定遗留 `unsupported_codec` 不再预先阻断。
- 共享预览/查看器组件测试固定先创建 `<video>`，原生 error 后才展示失败状态；浏览器媒体
  矩阵以真实 content 请求证明尝试发生。
