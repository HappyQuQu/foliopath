# 浏览侧栏折叠与滚动工具栏

- 需求：FR-BRW-001/004/005、FR-UI-001～003/006、NFR-UIF-001；POST-MVP-5 既有 UI 维护，沿用 S5-007/S5-008。
- 用户请求：快速访问区域可折叠；下滚后保留操作并有过渡，不把完整标题区直接固定在屏幕上。
- owner：DirectoryNavigation 统一折叠入口，shared storage/preferences 拥有偏好；BrowseControls 拥有本页吸顶、收紧和占位；媒体/预览的公共 owner 继续负责各自焦点与定位。
- 合同：复用已有服务、URL/query、Button、图标、tokens 和 ResizeObserver；无 API、服务端持久化/架构、部署单元或模型变化。本地展示偏好不增加服务端设置。
- 行为：默认快速访问展开，收起后目录树获得空间并记住状态；隐藏导航不参与 Tab。滚过标题后只保留原有一套操作，标题视觉退场，紧凑栏以现有动效 token 淡入/轻微位移；返回顶部恢复。保留完整控件与文档占位，不复制工具栏、不改变筛选值和虚拟列表顺序。减少动态效果偏好取消过渡。
- 正式 Story：CollapsibleNavigationAndDock 使用明确标记的合成合同夹具，直接渲染 BrowsePage。

## 验证

- `npm run check:types`、定向 27 项测试、完整 `npm run test`（78 文件 / 528 项）、`npm run check:architecture`、`npm run generate:check`、`npm run build` 与 `npm run build:storybook` 通过。
- `make arch-check generate-check`、`git diff --check` 通过；未运行 `make fmt`、`make lint`、Go 全套单测/集成、完整 E2E 或正式双原生发行验证。构建保留既有 chunk 大小提示。
- 正式 Dockerfile ARM64 构建成功。先正常 exit 0 停本地合成测试实例，完整备份命名卷，逐文件核验独立恢复副本，恢复副本 Linux `database-check` 通过，再只换测试镜像。线上静态包逐字节校验、容器内 `database-check`、99 张图片 / 46 个目录（含根）、目录分页及原件前后哈希核验通过。
- 实际 Docker 页面 390/768/1280/1920px 的首屏、下滚和回顶部检查通过：无横向溢出、只保留一个大小滑条。吸顶收紧前后媒体区文档 Y 完全一致，四档紧凑高度分别为 231/191/111/111px；保留原始占位，透明区域点击仍落到媒体卡。
- 快速访问折叠后刷新仍保留；滚动时筛选、类型、布局、大小、排序和刷新保持原 DOM。侧边预览打开/关闭后焦点回到未被遮挡的原卡。中英与浅深样例可读；正常过渡 0.2s，模拟减少动态效果时 animation 为 none，检查后已取消模拟并恢复中文浅色。控制台无 error/warn。
- 本机 `http://127.0.0.1:6010/libraries/lib_1/browse` 的独立合成测试实例已更新并保留运行；该本地验收阶段没有更新远程实例或发布新版本。镜像为 `foliopath:local-browse-dock-20261028`；证据、完整测试数据备份和截图位于 `.artifacts/browse-dock-20261028/`，均不提交。
- 普通测试源、截图和输出仅本地保留，不扩大 25 文件发行白名单。

## 发布授权

用户已验收本地效果，并于 2026-10-28 明确授权提交 PR、合并及发布新版本。沿用现有 Release Please、双原生镜像验证和不可变 digest 推广，不扩大测试文件的 25 项发行例外。正式发布结果如下。

## 正式发布结果

- [PR #66](https://github.com/HappyQuQu/foliopath/pull/66) 已合入 main，merge `01ca9e71b21f6962caf4ee9a1739f9656c039dc9`。现有 Release Please 自动合并 [PR #67](https://github.com/HappyQuQu/foliopath/pull/67)，发布 [v1.6.4](https://github.com/HappyQuQu/foliopath/releases/tag/v1.6.4)，冻结源码 `aa86dc1eb64fdca897e7c4fb348e00d4394b8ced`。
- [正式流水线 40636758513](https://github.com/HappyQuQu/foliopath/actions/runs/40636758513) 首轮 success：main/AI × AMD64/ARM64 四个原生构建验证及两个索引推广任务全部通过。每个组件每种架构仅构建一次，验证同一不可变 digest，推广未重建。AI 两平台的记录均包含身份、安装恢复、下载中断、离线重启及真实合成推理检查。
- 主镜像 `evanqu/foliopath:1.6.4`、`:1.6`、`:latest`、`:sha-aa86dc1` 均为索引 `sha256:e7b87320919672f28fd85166ca6f6366acfd32742db1e98a81ed2de794e1e9e7`。
- AI 镜像 `evanqu/foliopath:1.6.4-ai`、`:1.6-ai`、`:latest-ai`、`:sha-aa86dc1-ai` 均为索引 `sha256:0573493f286037acf55b938e2d02d2a94c889c8ed2ac5e4610e2a8f9cf5f8a29`。
- 本机通过 Docker Hub tag API 独立读取八个标签，逐组件索引及平台 runtime digest 一致，均支持 linux/amd64、linux/arm64；SPDX SBOM/SLSA 与 native descriptors 的完整保留由原有 CI publish owner 核验通过。本机未重复执行独立 registry descriptor 核验。
- 自动发布路径未运行仅用于手动 verification_only 的 source-check/verification-image jobs；不声明普通完整测试在 CI 执行。合并前本轮另执行的前端类型、架构和生成检查通过。原 528 项本地测试和实际 Docker 证据仍按上述边界记载，不扩大为完整 E2E、容量或质量证明。
- 25 文件发行输入白名单不变。原始流水线下载、日志和输出仅留本地 `.artifacts/browse-dock-20261028/`。本次没有更新其他实例，本地合成测试容器继续运行。此收尾文档使用 `[skip ci]`，不触发镜像重建或改写标签。
