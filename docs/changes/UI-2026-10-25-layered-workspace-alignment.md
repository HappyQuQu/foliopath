# 分层工作台：正式界面对齐已接受原型

- 状态：2026-10-25 用户授权按 `prototypes/layered-workspace` 对齐整个正式界面；正式组件、文档、Storybook与下述本地验收已完成。
- ID：UI-2026-10-25-layered-workspace-alignment；Owner：Codex / web。
- 目标：POST-MVP-5 既有 UI 消费者完善，沿用 Backend Ready 的浏览/搜索/整理/设置合同。
- 需求：FR-UI-001～003/006/009/010、NFR-UIF-001；关联 FR-BRW、FR-SRH、FR-CUR、FR-INT、FR-AUTH 既有切片。
- 视觉真相：`prototypes/layered-workspace/src` 与本聊天 4187 原型；上一次只接受侧栏，本次扩展到整体视觉。
- 不改变部署、模型身份、媒体权限、持久化、API 或产品能力；不把原型导览/状态选择器或模拟动作移入生产。

## 范围与所有者

| 原型页面 | 正式组件与合同所有者 | 对齐内容 |
| --- | --- | --- |
| 浏览、搜索、收藏、标签 | BrowsePage、SearchPage、CurationPage / catalog、curation | 固定分区侧栏，主区标题与工具，流式画布、目录紧凑行、浮动预览；筛选仍用 URL owner |
| 图片/视频查看器 | MediaViewerPage / shared MediaViewer | 延用完整缩放/键盘/Range 等行为，采用原型深色画布与控件色系 |
| 人物 | AIReviewPage / 全局 people、本库 clusters | 同一导航/背景/字级/卡片；没有封面合同，不从样本照片冒充人物头像 |
| 媒体库、新建、扫描/处理结果 | LibrariesPage、NewLibraryPage、ScanStatusPage、LibraryRecords | 3张独立摘要卡、统一表面/标题/留白与原有安全状态 |
| AI 功能、任务、服务/模型 | AISettingsPage、AISettingsOverview、AIProcessingRow | 图标底色、能力卡、分区表面、页签；保持真实队列/模型状态与防护 |
| 通用、扫描与资源、缓存、账户、日志、关于 | ManagementShell 与 settings/diagnostics/release-info | 复用 AppShell 的既有侧栏/抽屉插槽；一致的管理导航、上下文栏、面板、表单和页签 |
| 登录、首次设置、不可用 | PublicLayout、AuthPage、SystemUnavailablePage | 卡片尺寸/圆角/阴影与留白；认证前不复制原型中可浏览媒体的全局 Header |

`styles/tokens.css` 与 `themes.css` 唯一拥有蓝灰画布、独立侧栏、白色表面、深色蓝灰、字体、间距、
圆角和抬升。基本控件继续由 ui 拥有；RadioOption 提供原生单选语义。共享 MediaPreview 拥有浮动卡和
主要查看器入口；MediaCollection 保持 DOM/query 顺序和有界虚拟化，已可见卡片恢复焦点时不重新居中。
浏览标题/工具区有独立底色，子目录为横向紧凑行；重复的媒体章节标题保留语义而不挤占画布。
窄屏媒体类型组独占一行，防止中文标签逐字换行。预览格式徽标从真实 MIME 显示，不再把所有图片写为 JPG。
ManagementShell 仅组合已经接受的 AppShell 插槽，不另建抽屉/焦点控制器；沿用 ADR-0007，未改变依赖方向。

## 视觉与合同的有意差异

- 原型默认无缝瀑布流；正式产品保留自适应网格默认值及已保存布局偏好。对照时可显式选择相同布局/270px。
- 原型合成元数据、计数、AI完成情况、人物封面和模拟成功不是生产能力。真实信息来自既有 API；
  预览使用真实源图尺寸、格式与文件大小，不伪造 JPEG/8.4MB。
- 继续使用已有虚拟化列数计算、270px目标大小和已保存密度；与原型纯CSS画廊相比，边界宽度处列数可能不同。
  所有媒体页共享同一个预览，浮动卡始终从上下文栏下开始；不为标签另建一套从网格顶部开始的预览。
- 原型的按面板模拟“保存/恢复默认”不复制为另一套偏好状态机；正式页面仍按原合同保存/恢复已保存值。
- 原型较浅说明文字在蓝灰底上不满足对比度；正式 muted 使用更深的同色系，危险文字色与填充色分开，
  深色危险按钮文字也满足对比度。
- 移动端管理页与媒体页均复用同一工具抽屉；600px以下展开，768px平板保持紧凑190px侧栏。
- 独立验收实例未配置推理服务：显示真实未就绪/读取失败边界，不能当作 AI 已完成。人物、AI 的成功状态
  由正式组件的明确合成 Storybook 验证。没有触碰既有 6010 实例、真实媒体或活跃数据库。

## 验证记录

- `npm run check:types`、`check:architecture`、`check:visual-references`：通过。
  manifest v2映射23个正式页面/状态入口至已接受 sourceView，保留 Apple redesign 历史证据，不修改历史门禁。
- `npm test`：70个文件、434项通过；包含可见卡片不滚动恢复、原生单选、真实 MIME 徽标和现有100k有界列表回归。
- `npm run build:storybook`、当前源码 Docker生产镜像构建（含Vite/类型检查）：通过；仍有已有大chunk提示。
- `context-sidebar.spec.ts --project context-visual-chromium`：Linux 10项通过，32张侧栏基线不更新复验通过。
- `search-guidance.spec.ts --project chromium`：10项通过，包含320/390/768/1440/1920宽度、中文/英文、语义/文件名请求与键盘路径。
- `visual-regression.spec.ts --project visual-chromium`：9项通过，11张整体视觉基线更新后再次不更新复验通过。
  General/Storage标题期望同步既有正式“Configuration”语义，不重命名产品页。
- `layered-alignment-real.spec.ts --project chromium`：8项通过，每项19个真实后端路由；
  390/768/1440/1920×1000，zh-CN/light与en/dark配对。WCAG2 A/AA零问题、无横向溢出、标题未被顶栏覆盖、
  无JavaScript运行异常；390px额外检查媒体类型文案不逐字换行。四档原型/正式截图及同屏对照已人工审阅。
- `make fmt-check arch-check generate-check lint`：通过（只读fmt-check，不格式化其他在途Go工作）。
- `make test-e2e`：通过，独立临时合成媒体、真实容器启停/重启、只读挂载及媒体不变校验。
- `git diff --check`：通过。

本轮没有修改Go业务代码；未重复全量Go/集成测试。上一轮 `make test` / `make test-integration`
在既有模型下载集成测试被 `insufficient_space` 拒绝（34GiB可用低于约46GiB保留线），
详见[侧栏验证记录](UI-2026-10-24-contextual-sidebar.md)。没有降低保留规则或清理其他数据。
本轮没有重新证明真实推理质量、100k实机容量、完整跨浏览器/四种主题语言组合或双架构发行就绪。

验收实例：`http://127.0.0.1:6012`，容器 `foliopath-layered-review-20261025`，
独立命名数据卷和只读合成媒体挂载。既有6010及真实图库未改动。该实例运行正式生产bundle，不是原型或Storybook副本。
截图与日志：`.artifacts/layered-alignment/`；同屏对照 `comparison/{browse,tags,general,ai}-same-viewport.png`。
最终视觉核对：[design-qa.md](../../design-qa.md)。

后续部署：2026-10-25用户授权后，6010已按[独立部署记录](DEPLOY-2026-10-25-layered-ui-6010.md)更新；
此前“未替换6010”描述属于界面验收完成时的历史边界。部署保留旧后端/AI配置与全部已有任务数据。


## 后续全面复核：整体一致性重新打开

2026-10-25用户继续反馈后，[本轮重新截图审查](AUDIT-2026-10-25-prototype-alignment.md)确认多项结构/入口遗漏，
整体只达到部分对齐。上文历史命令执行和单项修复记录保留；“已完成”不作为当前整体原型一致性结论。
视觉manifest及实现自身的基线不能替代逐页对照，后续以审查清单逐项关闭。
