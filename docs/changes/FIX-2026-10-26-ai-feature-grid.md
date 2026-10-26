# 智能分析功能设置卡片布局统一

2026-10-26；既有POST-MVP-5 UI维护；FR-UI-001～003/006、FR-OCR-004；owner：AISettingsOverview。用户接受宽屏两列、空间不足单列，修复原搜索/人物双列而标签/OCR整行的布局割裂。

四项卡片进入同一网格，DOM顺序仍为智能搜索、人物分析、自动标签、文字识别。复用既有卡片、间距和操作样式，不新增控件或业务动作；开关/进度/查询与修改均保持现有逐库API合同。列数及窄内容区内边距依据AppShell workspace实际可用宽度，侧栏调宽时重新排布；两列时同行卡片等高、动作区按既有样式靠底。

原Overview Story和新增FourCapabilities稳定状态直接渲染生产AISettingsPage，只有只读合成合同数据，不复制原型、不让预览触发真实任务。验证应覆盖中英文、浅/深主题、320/390/768/1024/1280/1440/1920及宽侧栏，无横向溢出。

6010的pixiv OCR job `ocrjob_16133bb86bc37e890f99a833d2840483`正在运行。本轮先完成代码、预览与候选校验，不重启部署打断识别；任务终态核验后才停写备份并只更新主应用。新OCR策略、Linux数据卷、其他能力身份和全部五Compose必须保留，不能恢复旧宿主data或旧界面资源。


## 验证与交付状态

- Node24.22.1，AISettingsPage/OCRSettings/AutomaticTagSettings共71项既有行为测试通过；类型和前端架构检查通过。
- 稳定Storybook构建通过。Linux Chromium中英文浅/深配对 × 320/390/768/1024/1280/1440/1920共14项通过，检查四张卡片同一网格、同列宽、同行等高、两行两列/统一单列、document/main无横向溢出及axe。
- 1280px同一视口用键盘Home/End将侧栏从192调至480，两列随右侧可用空间变单列，验证按容器而非窗口断点。最终阈值52rem；初轮64rem在常规预览窗口过早收成单列，因此在视觉审查后调整并重跑14项。
- 首轮Docker浏览器被Vite预览allowedHosts拒绝，未改生产安全设置；改用已解析Docker宿主IP访问同一静态预览后通过。
- 冻结当前067c部署源码，仅拷入两份生产布局源及稳定Story。internal/与migrations逐文件SHA相同；保留OCR v2、当前彩色标签云/五档字号/拖宽/自适应，候选`sha256:9748e2928e9b4d19680e46ffd7a57e4ca6cfaddaa6ac094f819c1cfc378b469c`构建通过，并在已停止的合成schema47副本上database-check通过。
- 没有改后台/API/数据/模型/任务合同，未重跑全Go/集成/容量/跨浏览器发行面；不把本轮检查说成这些门禁通过。

当前6010仍为067c主/7739 AI，未发布本轮布局，未重启或改变OCR任务。已有`pixiv`跟进会在原999张任务终态及错误/搜索/保护核验完成后，按`.artifacts/ai-feature-grid-20261026/deploy-plan.json`停写全备份Linux数据卷与私有配置、校验后只切主镜像，AI和全部五覆盖保持；有未解决程序错误时先停留在待部署状态。

新增部署阻断：用户授权并行发布核验后，冻结候选自身的完整前端测试出现失败。前述71项来自当前工作区相关测试，不能算冻结版本全套回归。正逐项区分陈旧断言、fixture缺失和实际行为问题；失败证据保留在`.artifacts/release-readiness-20261026/regression/`。在差异查清并补验前，即使OCR终态也不部署本布局候选，不通过忽略失败或覆盖历史通过记录绕过。

预览：`http://127.0.0.1:6007/iframe.html?id=features-intelligence-settings--four-capabilities&viewMode=story`，生产组件/只读合成数据。[机器证据](../evidence/ui/2026-10-26-ai-feature-grid.json)。未提交推送。
