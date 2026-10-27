# 浏览页子目录布局修复

- 需求：FR-BRW-001/004/005、FR-UI-001～003/006、NFR-UIF-001；POST-MVP-5 既有候选维护，沿用 S5-007/S5-008，owner 为 Web BrowsePage。
- 用户接受：保留现有配色和侧栏，优化主区子目录的对齐、占用高度及展开入口。
- 原因：子目录 section、标题和分页按钮共用横向 flex；目录宽度随名称伸缩，标题落到列表中间，按钮被挤出主区。
- 合同：复用当前目录 GET directories、原 cursor、browseUrl、TanStack Query 和公共 Button。没有 API、持久化、架构或部署变化。
- 实现：独立标题、流式等宽网格、名称两行及固定数量位置；原生 title 提供悬停全名，focus-visible 展示完整文字。ResizeObserver 读取 CSS 实际列数，默认只渲染两行；展开呈现已加载目录，继续加载仍由原 query 有界分页负责。收起不清缓存；本地展示不增加 URL 参数或设置。
- 稳定 Story：BrowsePage.ManyDirectories 直接渲染生产页面，使用明确标注的合成中英长名称、空目录与分页合同夹具；不访问真实媒体。

## 验证

- Node v24.22.1；`npm ci --no-audit --no-fund` 成功。
- `npm run test -- src/features/browse/pages/BrowsePage.test.tsx`：15/15 通过；完整 `npm run test`：78 文件、526/526 通过。初次两项旧导航断言因网格的可访问名称分隔空白失败，已将精确文字断言适配为允许空白，完整重跑通过。
- `npm run check:types`、`npm run check:architecture`、`npm run generate:check`、`npm run build`、`npm run build:storybook` 通过。构建保留大 chunk 的提示，不影响成功结果。
- `make arch-check`、`make generate-check`、`git diff --check` 通过。
- 实际 in-app browser / 正式生产组件 Story：390×844、768×1024、1280×900、1920×1080，默认均两行、无页面横向溢出；宽屏 5 列、普通桌面 3 列，窄屏按可用空间降至单列。标题在网格上方，展开/分页在下方。检查展开、加载下一页、收起与长名称键盘聚焦；无真实媒体访问。
- 截图：`.artifacts/directory-layout-20261027/{390,768,1280,1920}.png`。Story 为 `Features/Browse/Directory/ManyDirectories`，合成媒体故意呈现现有“缩略图不可用”状态，不代表真实媒体故障。
- 本次为前端局部布局修复，未执行 `make fmt`、`make lint`、`make test`、`make test-integration`、`make test-e2e` 或跨浏览器完整 E2E；不声明后端、容量、发行或真实部署验证。
- 测试源、截图和测试输出按仓库规则仅保留本地。未部署运行中的实例。
