# FIX-2026-07-31：管理页按钮前景色覆盖

- 类型：已批准管理中心/UIF 切片内的例行视觉与可访问性修复
- 影响：存储设置主按钮、账户页危险按钮，以及管理页 `.row` 内的其他共享按钮
- 不变量：共享 `Button` 是按钮颜色唯一 owner；页面样式不得覆盖其内部文字包装节点
- 根因：`ManagementPage.module.css` 的 `.row span` 后代选择器误命中 Button 内部 `<span>`，
  覆盖了 `primary` / `danger` 的 `--color-on-accent`
- 修复：选择器收紧为 `.row > div > span` 与 `.usageHeader > div > span`，只设置说明文字
- 回归：共享按钮实现不变；设置页、类型检查、前端架构和完整前端测试保持通过

