# FIX-2026-10-26 标签云排列修正

- FR-UI-001～003/006、CR-2026-031/033、POST-MVP-5既有UI维护；owner：curation前端。
- 用户明确要求自由错落的标签云，而非逐行列表。只调整LibraryTagCloud.module.css：词块随文字宽度横向排列并自然换行，数量紧跟名称；保留五档字号、灰绿蓝配色、双语、选中/悬停、DOM/键盘顺序及当前库统计。
- GET /api/v1/tags和标签资产合同、URL、有界分页及任务/数据均不变。不是新增随机后端排序，也不使用绝对定位。
- Storybook继续使用生产CurationPage/LibraryTagCloud及已标记合成fixture，词云随侧栏宽度自适应。

验证：9项组件单测、TypeScript、frontend architecture、make arch-check、visual reference manifest、Storybook build通过。
Linux Chromium 18项通过（10个长名称/分页/语言/滚动/axe和8个词块同排/换行/调宽/URL检查）；视觉基线更新后不更新模式18项复跑通过。
断点覆盖320/390/611/768/1440/1920，中英文与浅/深主题配对。全部浏览器测试使用合成API与媒体。
完整后端、集成、容量及跨浏览器发行矩阵未重跑。

已部署6010：`sha256:a768f58be3a1c615f19b2c04c3dbfe2410c11ead6110dff05c30bfb2c6d9ac32`，仅主镜像变更；AI容器ID/启动时间、挂载和环境不变。
完整停机备份`/Volumes/extended/foliopath/backup-6010-tag-cloud-layout-20261026-F0ssZV`，旧/新镜像对独立Linux副本database-check均通过。
真实库74个标签形成多列词云，五档字号/三种颜色保留；611px下scrollWidth=clientWidth=611。

## 部署期间观察到的既有后端异常

部署前观察到原主进程SQLite WAL读取SIGBUS并自动恢复，另一次停机检查读到旧claim未移交而中止部署。
这不是本次CSS修复已解决的后端问题。没有手改任务、attempt或结果；恢复后重新读取当前任务，再使用ADR-0036完成计划停机交接。
最终冻结副本中同OCR任务queued、checkpoint8200/已记录457、attempt退回2；启动后同ID推进至checkpoint8204/已记录461，随后因固定错误码output_limit进入failed。
因此只确认已有进度保留及任务曾续跑，不能宣称它仍在运行或全部完成。没有自动重建/重试OCR。
最终校验使用停止后的完整副本，避免重复在线database-check；除OCR任务/结果/搜索文档自然变化外，30组受保护摘要一致。
证据`.artifacts/tag-cloud-layout-20261026/`，日志`/tmp/tag-cloud-layout-*.log`。
