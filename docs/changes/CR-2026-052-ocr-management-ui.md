# CR-2026-052 OCR按库管理界面

2026-10-26；POST-MVP-5 S2；FR-OCR-004/005；owner：Web AI feature。消费已通过Backend Ready的CR-2026-050管理合同及CR-2026-051搜索合同，不新增API、调度/存储或模型策略。

概览同现有能力卡片显示本库开关、当前源覆盖与任务入口；任务页复用AIProcessingRow，显示真实ready/empty/failed/pending/stale与最近任务的recorded/failed及固定错误码说明。开关不自动提交任务；补齐missing不重跑ready/已记录坏源；retry_failed和clear明确确认；取消保留已提交结果，clear仅OCR并关闭本库OCR。源文件失败目前只有数量，合同不提供逐文件清单，不虚构文件名/原因。模型由部署提供，不增加网页下载或身份编辑。

所有控制对应既有生成client/session/CSRF/ETag/幂等合同。任务不是通用AI operation，不塞入原语义/人物/tag operation表。404显示OCR管理不可用；状态读取失败隐藏旧成功状态并阻止写入；库选择和任务/概览URL沿用现有owner。正式组件及稳定Story同步，合成状态明确标记、写操作不模拟成功。

验证与交付已完成；原6010、原媒体和人工数据保持不动，10源问题继续暂留。无新工具/框架或并发安全层。

## 交付与验证

已完成正式OCR按库管理界面。概览保持开关/覆盖/搜索与任务入口，任务行直接展示服务错误；源失败说明、重试和清理位于“更多”。历史epoch/model任务明确不代表当前覆盖，任务recorded包含源失败，不当作成功数量。未配置/读取失败不显示旧成功或可写开关。

- 只增加前端适配与组件，既有OpenAPI/后端/迁移/模型身份不变。取消前读取实际job ETag，不在网页拼接服务端guard；复用现有unwrap、CSRF、请求键和公共searchKeys。query按挂载页面2秒（运行/清理）或10秒刷新，写后重新读取状态并使搜索缓存失效，无自动写入重试或新并发框架。
- 新增8项组件测试使用真实OCRSettings、TanStack Query与生成client/domain adapter，只在HTTP边界提供有限响应。验证真实body/header、无隐式任务/清理、确认、取消job guard、错误后隐藏缓存状态与412不重放。AISettingsPage既有48项加组件8项共56项通过；AI/搜索定向236项通过，前端全量77文件519项通过，无跳过。
- Chromium/Firefox/WebKit × 中/英文与light/dark × 320/390/768/1440/1920共30项通过；每项覆盖错误任务、概览默认关闭、清理和未配置Story，键盘Enter打开/Escape关闭确认并恢复焦点、axe WCAG2A/AA、无横向溢出与宽屏有效宽度。检查390与1920截图；Story明确合成只读，拒绝写入，不是OCR质量证据。
- 本地Docker arm64 native/race应用probe `sha256:fe14bb9eccf7279ab618312c501b0c905c1c3a81f4d067344f59ac029b63efcb`，原AI `sha256:e995f5748c36ae8de9f3dd86633e4a2f60f3f6d6b7eaaba6616bbdf6ee73da21`：真实管理启用/补齐、三OS进程同job从2续到5、中英文与EXIF各4行、空白ready0/坏源2、实际OCR搜索/清理后不命中、人工标签/源hash/完整性通过；客户端exit0/OOMfalse，AI memory.events max/oom/oom_kill=0，peak2,191,609,856 B。仅单/library只读合成挂载、内部网络无端口，数据库在Linux tmpfs；测试容器/网络/scratch清理。
- 原生probe回归现有后端真实管理合同；浏览器操作使用正式组件与只读合成快照，写入body/ETag由组件HTTP测试验证，并未把Story按钮操作宣称为真实Docker图库写入。OCR重试/取消的后端合同由已有Go测试覆盖；本轮不改其策略。
- `make fmt arch-check generate-check lint test test-integration test-e2e`均通过；`npm ci`、类型/前端架构、519项全量、Vite与Storybook构建通过。Node24.22.1/npm11.19.1，不改依赖；保留既有2个moderate audit与安装脚本提示、大chunk警告，未用force/放开脚本或调整无关依赖；高危audit门结果见机器记录。

## 首轮测试发现

1. Node Request不支持浏览器相对URL，最初HTTP组件测试未到达fetch。只在测试环境提供绝对URL解析的Request，继续使用真实生成客户端，7项首次通过后增加状态刷新/冲突项，最终8项通过；不改生产客户端。
2. Safari原生鼠标点击不保证按钮聚焦，首轮6项将鼠标触发误当成键盘焦点规则。改为聚焦后用Enter打开、Escape关闭，验证已有Dialog的键盘恢复合同；没有新增公共弹窗行为或功能补丁。
3. 完整截图在确认关闭自动滚动后将sticky导航绘于中间。截图前回到页首，最终矩阵复验通过，没有为截图修改产品布局。

## 保留边界

原6010未部署，OCR仍需独立部署配置且库默认关闭；10个原源问题未操作。无新逐文件失败列表API，只有当前源失败数和任务固定错误。90°/版面质量、真实图库验收、历史保留、100k/联合容量、双架构与发行仍独立。下一步可做原6010的停写备份/候选切换及小范围真实图库OCR验收，需明确部署授权后执行。未提交推送。
