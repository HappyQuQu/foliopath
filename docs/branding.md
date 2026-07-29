# FolioPath 品牌标识规范

## 状态与所有权

- 状态：已确认并进入生产界面。
- 确认日期：2026-07-29。
- 对应基线：`FR-UI-007`、`RQ-009 A`、[CR-2026-003](changes/CR-2026-003-brand-identity.md)。
- 设计系统 Owner：`web/src/components/ui/BrandMark`。
- 生产矢量文件：[`web/public/foliopath-mark-tree.svg`](../web/public/foliopath-mark-tree.svg)。
- 原始概念参考：[`web/public/foliopath-mark.png`](../web/public/foliopath-mark.png)。
- 像素一致 SVG 封装：[`docs/brand/foliopath-mark-exact.svg`](brand/foliopath-mark-exact.svg)。
- 原始 V8 整图 SVG 封装：[`docs/brand/foliopath-logo-v8-exact.svg`](brand/foliopath-logo-v8-exact.svg)。

`BrandMark` 是生产代码中唯一的品牌图形入口。页面和 feature 不得复制 SVG、
导出另一份颜色变体或以通用图片/文件夹图标代替它。所有语义尺寸统一使用目录树
SVG，消费方不得自行选择资产。

`foliopath-mark-exact.svg` 将确认后的透明 PNG 自包含嵌入 SVG，适用于必须接收
`.svg` 文件且要求外观像素一致的场景。它不是可编辑的纯 path 矢量源，不能用于改变
轮廓、颜色或玻璃材质，仅作为历史参考，不用于产品界面。

生产文件 `foliopath-mark-tree.svg` 只使用三条圆角矢量路径，不包含 Base64 位图、
阴影、模糊或运行时滤镜。浅蓝主干、青色中层和蓝色当前路径共同表达目录树，并形成
抽象的字母 “F”。它是所有尺寸和主题下的唯一交付源。

`foliopath-logo-v8-exact.svg` 则完整嵌入未经裁切、抠图或调色的 V8 原始概念稿，
包括图形和 `FolioPath` 字标；其嵌入字节必须与
`foliopath-logo-concept-v8-liquid-glass.png` 完全一致。

## 设计概念

标识由三条连续、圆润的目录路径组成：

1. 浅蓝主干表示媒体库根目录；
2. 青色中层表示目录层级；
3. 蓝色折线路径表示当前进入或选中的目录。

图形同时表达 “folio（媒体集合）”、“path（目录路径）” 与产品首字母 “F”，
不直接使用相机、光圈、山景或通用文件夹。

## 视觉语言

- 风格：极简、轻盈、内容优先。
- 主体色：浅蓝、青色、蓝色；不使用黑色或深色主体。
- 材质：纯色矢量线条，不使用滤镜或拟物效果。
- 轮廓：连续圆角，16px favicon 和 28px 页头尺寸下保持清晰。
- 字标：产品界面使用系统字体栈渲染 `FolioPath`，不把文字固化进 SVG。

PNG 中的品牌固有颜色属于媒体资产自身，不进入主题 token。页面布局、尺寸和间距仍必须
使用设计系统 token。

## 使用规则

### 标准尺寸

| 场景 | 组件尺寸 | 用法 |
| --- | --- | --- |
| 浏览器 favicon | 浏览器决定 | 使用 `/foliopath-mark-tree.svg` |
| 应用侧栏、公共页头 | `small` / 28px | 与文本 `FolioPath` 并列 |
| 独立品牌位置 | `medium` / 32px | 使用统一目录树标识 |
| 登录与首次设置 | `large` / 64px | 独立显示，标题负责提供上下文 |

标识周围至少保留约四分之一个图标宽度的净空。不得拉伸、旋转、裁切、改变各层相对位置，
也不得在图形外再增加固定圆角方形容器。

### 背景与主题

- 浅色主题优先放在白色或浅灰表面。
- 深色主题继续使用同一轻色标识，不反相为黑色版本。
- 不在高细节媒体缩略图上直接叠放；必要时先提供由界面语义 token 控制的安静表面。
- 高对比或强制颜色模式下，品牌标识仍是装饰内容；产品名称必须保留为真实文本。

### 可访问性

当 `FolioPath` 文本或页面标题已经提供品牌名称时，`BrandMark` 使用空 `alt` 并从
可访问性树隐藏，避免重复朗读。不得仅靠标识颜色传达运行状态、选择或错误。

## 文件与实现

```text
web/public/foliopath-mark-tree.svg
web/src/components/ui/BrandMark/
├── BrandMark.tsx
├── BrandMark.module.css
├── BrandMark.stories.tsx
└── BrandMark.test.tsx
```

Vite 将品牌 SVG 原样复制到嵌入式生产资源根目录。消费方使用 `BrandMark` 的
`small`、`medium` 或 `large` 语义尺寸，不手写像素尺寸或直接引用资产。

## 验证

- 组件测试确认使用唯一 SVG、装饰性替代文本与隐藏语义；
- TypeScript 类型检查和 Vite 生产构建必须通过；
- Storybook 覆盖三个语义尺寸；
- 视觉回归覆盖登录页、公共页头、应用侧栏以及浅色/深色主题；
- 发布构建必须包含 `/foliopath-mark-tree.svg`，浏览器 favicon 请求不得返回 HTML fallback。
