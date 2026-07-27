# S4-008 媒体交互矩阵完成记录

## 结论

**Done — Chromium 桌面/移动触摸、键盘焦点、真实视频 Range 与错误降级矩阵已完成。**

本记录完成 `S4-008`，不把 Stage 4 标记为 Integrated Done。搜索 → 非模态预览 → 完整
查看器的完整纵向 E2E 与阶段 Gate 仍由 `S4-009` 负责。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 4 / `S4-008`
- 需求：`FR-MED-002～008`、`FR-UI-001～004`、`NFR-ACC-001`
- 上游 Gate：[S4-007 媒体播放与降级状态](s4-frontend-media-strategy.md)
- 查看器交互 owner：`web/src/components/patterns/MediaViewer`
- 媒体状态 owner：`web/src/lib/media/availability.ts`
- 浏览器矩阵：Playwright `chromium` Desktop Chrome 与 `mobile-chromium` Pixel 5

## 已固定的行为

- 查看器打开时关闭按钮自动获得可见焦点；工具条按钮获得焦点时，`I` 信息开关、
  左右方向键和 Escape 仍属于查看器级快捷键。
- 原生视频、输入、选择、文本域或可编辑内容获得焦点时，不抢占冲突快捷键；Escape 仍提供
  统一退出路径。
- Pixel 5 触摸视口初始收起基本信息，可触摸打开/关闭；离线状态的重新检查、关闭和前后项
  始终可见可达，且没有页面级横向溢出。
- 浏览器加载仓库内真实合成 H.264/AAC MP4。受控内容 endpoint 支持字节切片，测试观察到
  浏览器发出的 `Range: bytes=…`，并以真实 `206 Content-Range` 响应完成 metadata
  加载；poster 和原生 controls 同时保留。
- 不兼容 codec 不挂载播放器；offline 提供重新检查；明确 deleted 不提供无效重试。三种
  状态都保留查看器关闭与有界导航。
- 媒体工作台通过仅用于测试/Storybook 的静态目录使用同一合成 MP4，不把 fixture 放入
  生产 bundle 或 `web/src` 业务 import graph。

## 自动化与设计证据

- `web/tests/e2e/media-matrix.spec.ts`：桌面键盘/焦点、真实 Range/206、codec/offline/
  deleted，以及 Pixel 5 触摸恢复矩阵。
- `MediaViewer.test.tsx`：工具条焦点允许查看器快捷键，视频焦点阻止冲突方向键。
- 两个浏览器项目都检查无页面横向溢出和 axe serious/critical 零违规。
- Product Design 浏览器审查覆盖 1280×800 正常查看器、信息开关、不兼容 codec，以及
  390×844 离线恢复；证据见 `web/design-qa.md` 与 `web/qa/s4-008/`。

## 验证

完成记录创建时成功执行：

```sh
npm --prefix web run check
npm --prefix web run build:storybook
make test-web-e2e
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

## 保留边界

- `S4-009`：搜索 → 预览 → 查看器真实纵向 E2E 与 Stage 4 Integrated Done。
- Firefox、Safari/WebKit 的具体发布版本、真机媒体栈和低性能客户端指标归 Stage 5
  发布 Gate；本记录不把 Chromium 仿真结果外推为这些平台已通过。
- MVP 不转码、不新增下载按钮，也不承诺浏览器无法播放的 codec。

- 评审日期：2026-07-28
