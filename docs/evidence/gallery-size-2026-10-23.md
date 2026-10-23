# 缩略图大小与来源文字纠偏验证（2026-10-23）

关联 [CR-2026-034](../changes/CR-2026-034-gallery-size-and-source-icon.md)、FR-BRW-014/015。
环境：macOS arm64、Node 24.22.1 / npm 11.19.1；本机 Docker Linux 引擎。

## 已执行并通过

- `npm ci --prefix web --no-audit --no-fund`、`npm run build --prefix web`。
- `npm run check --prefix web`：high 审计、生成一致性、前端架构、视觉引用、TypeScript、
  48 文件 / 174 测试、生产 Storybook（含 UI/Slider 与递归来源的 Seamless 稳定状态）。
- `make fmt arch-check generate-check lint test test-integration test-e2e`，所有目标执行成功；
  Docker 应用烟测验证认证、持久化、只读媒体和正常停止。
- `FOLIOPATH_WEB_E2E_URL=http://127.0.0.1:4274 npm run test:e2e --prefix web --
  --project=chromium --project=firefox --project=webkit --grep 'seamless masonry'`：48/48 成功。
  2 语言 × 2 主题 × 4 宽度 × 3 浏览器引擎，全部使用合成媒体，生产页面及适配器。
  检查递归来源图标没有文字且可聚焦，滑条 End/Home/方向键、放大后的卡宽大于缩小后、
  列数变化后的横纵零间距（容差 0.08 CSS px）、刷新记忆、默认有界 DOM、无横向溢出、
  图片与卡片同高、键盘媒体焦点、切回网格仍有名称。

原生 range 提供鼠标和触屏拖动；浏览器矩阵直接操作键盘路径，不声称已验证真实触屏硬件。
本机检查不代替原生 Linux 双架构发布工作流；不新增 100k 性能结论。

## 合成状态截图

- [390px](gallery-size-2026-10-23/390.png)
- [768px](gallery-size-2026-10-23/768.png)
- [1280px](gallery-size-2026-10-23/1280.png)
- [1920px](gallery-size-2026-10-23/1920.png)
