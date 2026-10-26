# FIX：搜索结果首屏密度

- 状态：2026-10-25已修复并同步6010，真实图库只读核对通过。
- 范围/Owner：POST-MVP-5既有消费者维护；UI-2026-10-25-layered-workspace-alignment、CR-2026-035、FR-SEARCH-001～003、FR-UI-001～003/006、NFR-UIF-001；web/search，复用共享InlineStatus与MediaCollection。
- Gate/合同：既有已接受分层工作台与融合搜索；搜索范围、URL、来源/证据、排序、快照分页及API合同不变，无schema、模型身份或新能力。

## 反馈与修复

用户截图中搜索页重复显示页面标题和结果标题，介绍、筛选条、状态横幅和集合工具多层叠加，图片被推到首屏下部。
正式SearchPage改为一个查询标题，已载入数量与标题并排；范围继续由顶部上下文与左侧工具拥有，不再增加重复的库名眉题。
筛选摘要和刷新在同一紧凑区域；有查询时不再重复介绍段落，排序与来源说明只显示一次。
普通覆盖状态使用已有InlineStatus的plain变体，保留图标、已分析/可分析和失败数；真正不可用、排除或降级仍显示原状态提示。
结果内容采用连续纵向排版，MediaCollection有单独容器，避免其fragment子元素分别继承父布局的额外间距。
标题/筛选保留浅色分区，结果区域流式使用可用宽度；移动端自然换行，刷新按钮不强制占满一行。

“已载入”不是总结果数；真实覆盖失败不隐藏，不改变查询、卡片来源、匹配依据、预览、刷新或分页。
正式Fused results Story继续直接渲染SearchPage和明确合成wire fixture，不增加独立原型副本。

## 验证

- `npm --prefix web run check:types`、`check:architecture`、`build:storybook`及冻结源码Docker生产构建通过。
- SearchPage定向单测12项通过。
- Linux Chromium融合搜索与搜索提示20项通过：320/390/768/1440/1920，zh-CN/light与en/dark配对；保留来源、键盘预览、范围请求、降级、axe与无横向溢出检查。
- 新增首图顶部边界：宽桌面不超过340 CSSpx、平板360px、手机520px，覆盖部分分析失败状态。此值为合成fixture验收，不冒充真实图库截图的CSS测量。
- 既有整体视觉9项更新后以不更新模式再次9项通过；字面搜索视觉fixture显式选择filename，避免使用其未模拟的融合端点。
- `make arch-check`、`git diff --check`通过。未重跑全量Go/集成、真实模型质量、容量或跨浏览器发行门禁；本次不关闭这些独立Gate。
- 真实6010同一搜索“车”：仍载入30项，覆盖785/792、失败7；标题/数量/筛选紧凑，图片在首屏明显上移，来源徽标和图库内容保留。未修改收藏、标签、人物或任务。
- 原/新真实截图：`.artifacts/search-density-20261025/01-before-real-search.jpg`和`02-after-real-search.jpg`；固定Linux宽屏证据位于browser-final/browser-verified，原生IAB截图不冒充1920截图。

## 6010更新与保护

使用上一部署冻结上下文，仅替换SearchPage及其CSS；没有带入当前工作区开发中的人物整理等改动。
旧前端加冻结后端重建与原运行程序逐字节一致，SHA256为
`53a7ad1f648143ab6f95b053eb8c88a7bf89cb051ea7a7cd6b6a32db09a522e6`，再构建新嵌入前端。

新主镜像`sha256:dab672cc4c1f5b1b070fe734e3030ed7c07e8acaf05a4e3274367cb72c15cfef`，主StartedAt为`2026-10-25T07:02:14.05552521Z`。
AI仍为52f4a40c，同一容器与2026-10-24 21:48:25 +08启动时间保持，未重启AI；semantic640、153类标签与两个png-v4 profile不变。
完整停写备份`/Volumes/extended/foliopath/backup-6010-search-density-20261025-OqXsHx`，37376文件/1095937762字节核对一致。
manifest SHA256为`7f08b0230483a78d72705d66f7eed21872afe9995f04329b6a8bd61c3fe29d72`；DB SHA256为`a1414126b410b521534667ac46478b47ba1557f0f57b40abd3a2e3f0615673d5`。
独立Linux备份副本旧/新database-check均通过；线上database-check亦通过，未在宿主打开活跃SQLite。
仅主镜像配置改变，挂载/端口/环境值不变，原媒体只读；6010五个静态文件与六个深链验证通过。
更新前后27组保护摘要、45条AI operation及队列完全一致；8690 ready、10既有源failed、0 pending不变，未取消或重提任务。
