# FIX-2026-10-26 侧栏调宽后的内容自适应

- 范围：FR-UI-001～003/006、NFR-UIF-001，既有POST-MVP-5媒体工作台维护。
- owner：AppShell/MediaPreview共享布局与browse/search/curation/AIReview页面；不改变API、URL、数据、模型或任务合同。
- 复现：原6010在611px视口、272px侧栏时，类型筛选使页面撑出约45px；仅检查768px及以上未覆盖此情况。
- 根因：响应断点依赖视口，而侧栏调宽改变的是可用内容宽度；scopeControls禁止收缩，Select包装占据100%后又叠加标签宽度。

## 修复

AppShell内容列提供命名inline-size容器。页面根据右侧实际可用宽度调整工具栏、类型按钮、网格及预览布局；
空间不足时控件换行、网格收敛、预览入流。修正Select包装的flex尺寸及人物卡片最小列宽。
不以隐藏页面横向溢出代替布局修复；保留拖宽、键盘、宽度记忆、标签颜色及五档字号。
Storybook继续渲染同一生产组件，合成fixture与真实图库分开。

## 验证

- 修复前611px专项真实实现回归失败，测得45px溢出。
- 6文件91项单测通过；类型、frontend architecture、make arch-check、Storybook build通过。
- Linux Chromium 14项响应矩阵及1项拖动几何回归通过：中文浅色/英文深色 × 320/390/611/768/1024/1440/1920，
  每项覆盖浏览/搜索/人物/收藏/标签，桌面检查已保存、最窄、最宽侧栏；四个媒体页面另检查成功载入图片的预览状态。
- 断言document及main的scrollWidth不大于clientWidth，测试使用有界合成API/媒体，未访问开发者图库。
- 未运行完整后端、集成、容量和多浏览器发行矩阵。
- 6010已部署 `sha256:a49a074afd3453885ab341ad39b1b88847e4c129bd68960101655cefad19cd1b`；AI容器ID/启动时间、配置与挂载不变，主应用健康，部署前/副本/部署后database-check通过。
- 完整停机备份 `/Volumes/extended/foliopath/backup-6010-sidebar-reflow-20261026-F2TyV8`。发现真实OCR任务正在运行，按ADR-0036计划停机移交；冻结检查确认同ID、checkpoint/完成数不回退且attempt退还，重启后同任务running并继续进展。没有取消/重建任务。
- OCR进度相关表自然变化，其他受保护数据摘要不变；见preservation.json，不声称OCR状态位级不变。
- 真实6010窄窗口：browse/search/people/favorites/tags均测得document和main零溢出。最终浏览页viewport/scrollWidth=611/611，main client/scrollWidth=339/339。截图real-browse.png。

证据：`.artifacts/sidebar-reflow-20261026/`，日志`/tmp/sidebar-reflow-*.log`。
