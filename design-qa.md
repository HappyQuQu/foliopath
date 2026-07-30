# FTR-UIF-001 设计一致性 QA

## 本轮范围

- 页面：生产 `GeneralSettingsPage` 与共享 `GlobalHeader`、`ManagementShell`。
- 视觉真相：`prototypes/apple-redesign/qa/management-general-final.png`。
- 桌面实现证据：`docs/evidence/uif-301/management-general-implementation.png`。
- 桌面组合对照：`docs/evidence/uif-301/management-general-comparison.png`。
- 移动实现证据：`docs/evidence/uif-301/management-general-mobile-390.png`。
- 移动结构对照：`docs/evidence/uif-301/management-mobile-comparison.png`；左侧参考为同一管理壳的媒体库状态，右侧为通用设置状态，因此只比较 Header、管理导航、边距和响应式层级，不比较业务内容。

## 固定状态

- 主题/语言：浅色、简体中文。
- 桌面视口：1440 × 900，截图像素比 1。
- 桌面视觉真相和实现截图均为 1440 × 900px；CSS 视口 1440 × 900、截图密度归一化为 1。
- 移动验收档：390 CSS 宽度；内置浏览器扣除宿主框架后的内容截图与聚焦参考均为
  375 × 812px，截图密度归一化为 1。
- 交互状态：系统主题、跟随浏览器语言、自适应网格、默认固定预览开启、无未保存更改。
- 数据：Storybook 使用已评审契约形状的确定性管理员会话，不进入生产 import graph。

## 比较历史与修复

1. 首次比较发现内容被额外限制为窄列，且浏览偏好、保存条缺失，属于 P1 布局和功能缺口。已移除页面级最大宽度，补全四项偏好、脏状态、恢复与保存行为。
2. 首次比较发现语言默认状态和预览开关与参考状态不一致，属于 P2 状态缺口。已加入“跟随浏览器”持久化语义，并以真实交互保存预览默认值。
3. 移动检查发现管理导航单行横向滚动，影响层级和触摸浏览，属于 P2 响应式缺口。已改为两行自然换行。
4. 移动检查发现 `.brand span` 同时隐藏了 `BrandMark` 内部图形，属于 P2 品牌资产缺口。已只隐藏字标并保留 28px 官方目录树标识。
5. 通用设置原先在 feature 内维护开关样式。为保持唯一组件所有权，已抽为共享 `Switch`，并补行为测试与 Storybook 状态。

## 最终检查

- 布局：桌面 Header、260px 管理侧栏、内容起点、分组卡片和保存条与参考保持同一层级；没有重复 Header 或重复媒体库入口。
- 字体与间距：标题、分区标题、行高、卡片内边距和行分隔保持参考密度；品牌字标是用户明确要求的增补。
- 色彩与表面：使用中央语义 token；浅色背景、边框、圆角、悬浮保存条和蓝色强调与参考一致。
- 图标与资产：使用项目唯一 `BrandMark` 和 Phosphor 图标，不包含 CSS 绘图、临时 SVG 或占位品牌。
- 响应式：移动 Header 保留品牌图形、全局搜索和账户入口；管理导航两行换行；内容、卡片和保存操作没有页面级横向滚动。
- 交互：已验证全局账户菜单、设置入口、退出动作、预览开关、保存及保存后无脏状态；Escape/焦点恢复由 AppShell 测试覆盖。
- 可访问性：语义 Header、search、nav、main、label、select、`role="switch"`、可见焦点和 reduced-motion 均保留；移动目标尺寸不小于既有 token 约束。
- 浏览器控制台：本轮桌面检查未发现页面错误。
- 遗留：本报告只关闭通用设置和共享管理壳的 P0/P1/P2；其余生产页面仍按 `UIF-307～317` 和 `UIF-401～408` 逐页验收，不能据此宣布整个 feature 完成。

final result: passed
