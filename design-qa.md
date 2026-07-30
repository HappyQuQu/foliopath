# UIF-312 Browse 设计 QA

## 比较目标

- source visual truth：
  - `prototypes/apple-redesign/qa/browse-directory-filter-right.png`
  - `prototypes/apple-redesign/qa/browse-directory-filter-right-mobile.png`
- implementation：
  - Storybook `Features/Browse/Directory`
  - `http://127.0.0.1:6006/iframe.html?id=features-browse-directory--default&viewMode=story`
- 主题 / 语言 / 状态：浅色、简体中文、京都目录、扫描进行中；桌面选中首项并打开固定预览。
- 桌面 source：1842 × 783 px。
- 桌面浏览器 viewport override：1842 × 783 CSS px，device scale 1；浏览器截图提供方实际输出
  1827 × 777 px，比较前等比归一到 1842 × 783 px。
- 移动 source：375 × 812 px。
- 移动浏览器 viewport override：375 × 812 CSS px，device scale 1；浏览器截图提供方实际输出
  360 × 780 px，比较前等比归一到 375 × 812 px。

## 证据

- 桌面实现：`docs/evidence/uif-312/browse-directory-implementation.png`
- 桌面完整组合比较：`docs/evidence/uif-312/browse-directory-comparison.png`
- 桌面工具栏聚焦比较：`docs/evidence/uif-312/browse-toolbar-comparison.png`
- 移动实现：`docs/evidence/uif-312/browse-directory-mobile-implementation.png`
- 移动组合比较：`docs/evidence/uif-312/browse-directory-mobile-comparison.png`

完整组合用于判断 Header、侧栏、面包屑、工具栏、扫描横幅、目录卡、媒体网格和固定预览的
整体比例；工具栏聚焦比较用于确认控件顺序、两侧分组、间距和 sticky 层级。移动组合足以清楚
读取全部关键控件与前三张目录卡，因此未再增加移动局部裁剪。

## Findings

当前没有可执行的 P0 / P1 / P2 差异。

- 字体与排版：实现使用中心字体与字号 token；标题、工具栏小字、卡片名称和辅助计数的层级与
  source 一致，没有阻断性的换行或截断。source 截图选择了“拍摄日期”状态，生产默认仍按
  已批准 URL 合同显示“文件名（自然顺序）”；用户选择日期排序时 URL 和结果可恢复。
- 间距与布局：桌面保持全局 Header、固定目录侧栏、双层顶部上下文、五列媒体网格和约 400px
  固定预览；移动端保持三行工具栏和单列目录卡。页面使用一个 window scroll，没有底部高度
  补偿空白。
- 颜色与 token：画布、表面、弱分隔线、蓝色选中态、状态横幅和预览层级均来自中心 token，
  与 source 的浅色层级一致。
- 图片与资产：正式 `BrandMark` 按已批准产品合同保留；source 只有文字字标的旧截图不再覆盖
  “项目 Logo 必须进入全局 Header”的较新确认。Storybook 使用真实 `unavailable` 派生状态，
  所以媒体内容不是 source 的彩色占位图；卡片裁切、比例和预览槽位仍按同一几何比较，不把
  合成占位色误当成生产资产。
- 文案与内容：当前目录筛选、媒体类型、扫描进行中、目录与媒体标题、预览操作均为正式中英文
  i18n 文案。实现 fixture 的目录数量与 source 不同，但不改变层级或布局。

## Comparison history

### Iteration 1

- [P2] 扫描横幅位于工具栏上方，与 source 的“工具栏 → 扫描状态”顺序相反。
- [P2] 固定预览打开后媒体区域显示六列，source 为五列，卡片密度偏高。
- 修复：把 offline/scanning 状态移到工具栏之后；媒体集合最小列宽从 180px 收敛到 210px，
  skeleton 使用同一密度。
- post-fix evidence：`browse-directory-comparison.png`。

### Iteration 2

- [P2] 375px 下布局/排序/刷新被拆成三个额外行，首屏比 source 多占一行。
- [P2] 移动面包屑隐藏了首个父目录“旅行”，只剩当前目录。
- 修复：布局、排序和紧凑刷新在第三行共享可用宽度；面包屑保留第一个父目录与当前目录，只
  折叠中间深层路径。
- post-fix evidence：`browse-directory-mobile-comparison.png`。

## 交互与运行检查

- 当前目录输入 `伏见` 后，目录和媒体同时进入 URL-backed 过滤，排序切到筛选默认值；
  清除操作恢复完整目录。
- “图片”三态切换后只保留图片结果；恢复“全部”后视频重新出现。
- 双击首项打开固定预览，父列表继续可操作。
- 检查浏览器 error / warning 日志；没有应用 error，只有 Storybook 自身的未来版本
  `PopoverProvider ariaLabel` warning。

## Follow-up polish

- [P3] source 的合成彩色媒体与真实派生失败 fixture 不是同一像素内容；后续 Linux 视觉基线应
  使用正式 synthetic media fixture 固定缩略图字节。
- [P3] 刷新是 `docs/ui-design.md` 明确要求的紧凑操作，因此生产工具栏比旧 source 多一个
  刷新图标；它不改变桌面分组，移动端仍保持三行。

## Implementation checklist

- [x] 当前目录 `q` 同时绑定 directory 和 asset cursor query。
- [x] 媒体类型由孤立漏斗改为可见三态。
- [x] 工具栏右侧依次为目录关键字、布局、排序和紧凑刷新。
- [x] 桌面固定预览、五列网格、单滚动容器与 sticky offset 对齐。
- [x] 375px 工具栏、面包屑和单列目录卡对齐。
- [x] Storybook 稳定状态、类型检查、行为测试和组合截图已生成。

final result: passed
