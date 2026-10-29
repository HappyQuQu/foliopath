# CR-2026-057：普通目录封面

2026-10-29 用户明确要求普通文件夹也像压缩包目录一样显示缩略图。目标为 POST-MVP-5 下一次发行，当前仅本地分支；FR-BRW-001/005、FR-UI-001～003/006、NFR-UIF-001。这是现有目录卡和只读缩略图的扩展，不改部署、数据持久化、API结构或缓存生成流程。

- 合同：沿用 Directory/DirectoryDetail `coverAssetId`，扩展为物理目录与压缩包内部目录也可返回可用封面。优先当前目录自然文件名排序的第一张可用grid缩略图，再以库内相对路径顺序选择后代中的可用缩略图。压缩包根目录沿用原有成员自然顺序。图片/视频已有grid缩略图均可作普通目录封面。无合适缩略图返回null。
- Owner：SQLite catalog适配器独占候选查询，检查library作用域、当前源fingerprint、当前grid transform、ready缩略图与可读压缩包来源；目录列表和详情复用同一SQL owner。只查当前页/单条详情需要的封面，借用已有目录与路径索引，不读取原文件、不新建封面持久化或每卡HTTP查询。
- UI：BrowsePage DirectoryCard复用现有压缩包封面位置和尺寸，所有目录同一48px封面容器、懒加载；普通目录以Folder图标兜底，压缩包以FileZip图标兜底。失败保留名称、数量与导航；侧栏仍为紧凑目录导航。
- 交付顺序：先接受OpenAPI字段语义并完成SQLite回归，再接生产UI/Story及验证。空目录、只有子目录、同名前缀兄弟目录、跨库、旧源/旧transform/未生成/失败缩略图必须有证据。
- Backend Ready：现有Directory/DirectoryDetail合同语义已更新并重新生成；SQLite真实存储回归 `TestDirectoryCoversUseCurrentScopedThumbnailsAndDescendants`、`TestZIPDirectoryMetadataAndReconcileRequestsFullScan` 通过，覆盖自然顺序、优先直接内容、子目录回退、源/transform过期、已驱逐缓存、空/未生成目录、跨库和不可读压缩包来源。
- 其余验证结果待完成后记录；普通测试与合成素材本地专用，不扩大发行白名单。

## 验证记录（2026-10-29）

- 前端 `cd web && npm run check` 全部通过：依赖审计、生成检查、前端架构、视觉清单、类型检查、80文件/547项测试与Storybook构建。生产FolderCovers Story覆盖普通封面、后代封面、无封面、图片加载失败；组件回归确认失败后名称、数量、导航仍可用。Storybook保留已有chunk大小提示。
- SQLite/catalog/API整包测试通过；源指纹/变换版本失效、缓存驱逐、跨库、相邻同名前缀、字面百分号路径、普通父目录与只有后代的根目录均由真实SQLite存储覆盖。额外回归模拟包根变为不可读、内部保留ready记录，确认旧成员不再作为父目录封面。
- 最终 `make fmt-check arch-check generate-check lint test test-integration test-e2e` 全部通过（E2E目标为application container smoke，不等同全浏览器矩阵）；`git diff --check` 通过。OpenAPI仅扩展已有可空字段的候选语义，结构/端点不变；重新生成客户端并按现有流程计算修订hash，合同检查通过。
- Linux arm64生产镜像构建成功，独立合成预览6020当前镜像ID为 `sha256:d0503f7d3c6a64b18af060cc24a89867654d918127f2a22cfff6b9ebfcf169bb`。容器healthy，容器内database-check通过；媒体只读挂载，数据沿用预览专用Linux volume，未变更222。
- 实际IAB浏览器验证物理文件夹、7z内部直接内容及只有更深后代的文件夹均显示已解码grid封面，空目录保留图标。390/768/1440/1920×1000四档几何/截图无横向溢出，封面48px；移动滚动可见普通文件夹封面。临时viewport已恢复。
- 本地输出：`/tmp/foliopath-folder-web-check.log`、`/tmp/foliopath-folder-backend-tests.log`、`/tmp/foliopath-folder-final-checks.log`、`/tmp/foliopath-folder-preview/image-build.log` 和 `evidence/`。测试、截图及输出依约不入库。
- 当前交付为本地分支实现和6020预览，未提交/发布；同一分支的7z与滚动加载工作保留。

本轮未重新执行100k容量或双原生发行验证，不新增容量/发行宣称。

## 后续正式发布

2026-10-29 用户随后要求提交、推送、合并及发布。此项已随v1.9.0正式发布，main/AI双原生验证与镜像推广均通过，见[发行记录](RELEASE-2026-10-29-v1.9.0.md#正式发布结果)。上文“本地分支/尚未发布”为当时实施阶段状态；不改写历史验证和限制。222未在此发布中升级。
