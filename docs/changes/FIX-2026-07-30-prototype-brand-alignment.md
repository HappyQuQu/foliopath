# FIX-2026-07-30 原型品牌标识对齐

- 类型：已批准 MVP 品牌 slice 内的例行原型修复
- 关联范围：`FR-UI-007`
- 目标版本与阶段：MVP / Stage 4 界面收敛
- 关联决定：[CR-2026-003](CR-2026-003-brand-identity.md)
- Owner：生产 `BrandMark`；原型只消费同一确认 SVG
- 合同：`docs/branding.md`、`docs/ui-design.md`
- 受影响不变量：不改变 API、持久化、媒体只读或部署边界

## 问题

Apple redesign 原型仍在全局 Header 使用纯文字或 `FP` 缩写，在登录和首次设置使用通用
图片占位图标，无法作为生产品牌实现的视觉基线；欢迎页还残留两套重复 Header。

## 决定

原型复制生产权威文件 `web/public/foliopath-mark-tree.svg` 的原始字节作为静态预览资产，
不得修改轮廓、颜色或比例。全局 Header 在桌面使用 28px 标识配真实 `FolioPath` 文本，
480px 以下只隐藏字标；登录和首次设置使用 64px 独立标识；原型目录使用 32px 标识；
所有页面使用同一资产作为 favicon。欢迎页只保留一条共享全局 Header。

该复制仅服务于 `prototypes/` 的独立静态服务器，不进入生产 import graph；生产界面继续只
通过 `BrandMark` 消费 `/foliopath-mark-tree.svg`。

## 证据

- 原型资产与 `web/public/foliopath-mark-tree.svg` 字节一致。
- `prototypes/apple-redesign/qa/logo-brand-comparison.jpg` 将官方资产、全局 Header 与
  登录页放在同一比较图中。
- `logo-header-mobile.png` 验证 390px 下只保留 28px 图形且无横向溢出。
- `logo-header-dark.png` 验证深色主题继续使用同一轻色标识。
- `logo-welcome-desktop.png` 验证欢迎页只有一个全局 Header 和一个 64px 品牌入口。
