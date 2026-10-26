# FIX-2026-10-22：智能分析入口、作用域与任务层级

状态：本轮实现与本地界面复验完成。不是全部 AI 质量或发行验收；完整仓库测试的环境阻碍保留如下。

- 目标版本与阶段：`POST-MVP-5` / `CR-2026-030`，既有 `INT-S3 Consumer/UI Ready` 内的 UI 修复。
- 需求：`FR-INT-001/003/004/005/006～016`；保留 `NFR-INT-002/003/005/006` 的任务、禁用、隔离与真实状态约束。
- Owner：`web/src/features/ai` 拥有设置/审核消费者与查询；搜索/路径、共享壳和组件沿用现有 owner。
- 合同：[`api/openapi.yaml`](../../api/openapi.yaml)、[独立 AI 变更](CR-2026-030-independent-machine-learning-service.md)、
  [S3 Gate](../gates/POST-MVP-5/int-s3-consumer-ui-ready.md)。不新增 API、持久化或部署单元。
- 原型：`AISettingsPage.stories.tsx` 和 `AIReviewPage.stories.tsx` 直接渲染生产组件与标注为合成的合同 fixture。

## 问题

原设置页在能力折叠行、图片维护、人物维护与任务页之间来回跳转，重复显示模型细节。
概览显示一个媒体库，人物维护却可显示全部库；当前库的搜索入口实际跳到跨库搜索。
“任务与维护”归在当前库下，但公开 operation 没有库归属，页面不能按库过滤这些记录。
人物整理混入标签审核，人物状态缺少持续轮询，进一步削弱了入口和处理范围的可预期性。

## 本轮决定与合同映射

| 页面/操作 | 作用范围 | 已有合同与限制 |
| --- | --- | --- |
| 功能设置：智能搜索开关、图片覆盖、搜索结果入口 | 当前媒体库 | `/libraries/{libraryId}/ai/semantic`；更新携带 ETag，启用成功再提交 missing；搜索保留库路径 |
| 功能设置：人物识别开关、覆盖、待整理入口 | 当前媒体库 | `/libraries/{libraryId}/ai/faces` 与 face-clusters；保留 offline/clearing/模型可用性条件 |
| 标签建议入口与 pending/accepted/dismissed 审核 | 当前媒体库 | tag-suggestions、tag-suggestion-reviews；只审核已有建议，不新增自动打标签开关 |
| 标签词表与建议生成 | 词表全局；生成按库 | `/ai/tag-vocabulary` GET/PUT 的 ETag 快照；`/libraries/{libraryId}/ai/tag-suggestions/jobs` 的 missing/all，复用现有 worker |
| 处理任务：图片、视频、人物补齐/重建与清理 | 当前媒体库 | 既有 semantic/jobs、video-semantic/jobs、faces/jobs 和清理端点；独立确认、幂等与 ETag 仍由原 owner 负责 |
| 处理任务：最近记录、详情与取消 | 跨库的可恢复记录 | `/ai/operations/{operationId}`；仅浏览器保存的 IDs 与服务端下载恢复记录，绝非服务器全部历史 |
| 服务与模型：部署信息 | 实例级只读 | `/ai/models` 返回的活动模型、版本、架构、许可与可用状态；Compose 提供配置 |
| 待整理人物、已命名人物 | 分组按库；已命名人物跨库 | face-clusters 按库；`/people` 无 libraryId 参数，需明确标示所有媒体库范围 |

设置页固定使用“功能设置 / 处理任务 / 服务与模型”三个页签，切换时不隐藏导航。
前两页共用 URL 中的媒体库选择；未指定时采用首个已载入库，无效 ID 必须提示重新选择，不能静默
对另一库操作。旧 `view=libraries` 和 `view=faces` 深链接进入处理任务；旧 `view=models` 进入
服务与模型，不恢复下载/代理界面。能力开关和结果入口集中在功能设置，处理操作集中在处理任务，
模型元数据只在服务页。该页另有明确标示为全局可编辑的受控标签词表，不与只读部署配置混淆。

标签审核以独立标题和入口呈现，不再作为人物的页签。当前页全选只影响当前有界结果，状态、分页
或库切换清空选择；批量审核继续检查逐项 outcome，冲突项保留并刷新 revision。标签生成依赖受控
词表及同代次图片索引；本轮补齐既有词表和 missing/all API 的前端入口，明确词表由所有库共享、
任务仅影响选中库。词表最多选择 1,000 项；候选标签列表按页加载，每页最多 200 项，不把当前页
误称全部标签。保存词表不自动提交分析任务，不修改人工标签或已审核结果；全量建议重建须确认。
不提供独立标签启用开关、质量阈值或自动写正式标签。服务端已组合文本编码器、
词表向量与有界生成 worker，本轮不新建另一条生成链；非空建议和质量仍以实际任务与审核结果为准。

人物与图片状态共用查询层刷新规则：building/clearing 每 1.5 秒，其余或无数据每 10 秒；沿用
TanStack Query 默认的后台暂停，不添加独立计时器。任务结束与覆盖完成仍分开表达。
可恢复 operation 进入终态后，查询 owner 同时失效标签建议缓存；失败或取消也可能已有部分结果，
不能只在任务提交时刷新并让审核页和计数长期保留旧值。
标签任务卡还被动观察同一个 operation 缓存，只在本页新发起任务运行期间每 1.5 秒刷新当前库
pending 第一页；切库、卸载、隐藏页、终态和操作读取错误时停止，不轮询整个审核结果集。

## 参考边界

参考 Immich 的配置、后台任务与日常结果分层，非像素复制，也不声称双方模型合同相同：
[任务管理](https://docs.immich.app/administration/jobs-workers/)、
[人物识别](https://docs.immich.app/features/facial-recognition/)、
[机器学习配置源码](https://github.com/immich-app/immich/blob/main/web/src/routes/admin/system-settings/MachineLearningSettings.svelte)。
FolioPath 继续由 Compose 提供模型身份，不照搬没有本项目后端合同的可写 URL、阈值或模型下拉框。

## 本轮修改前审查证据

本地截图保存于 `test-results/ai-ux-2026-10-22/`（运行产物，不提交媒体或数据库）：

- `01-before-settings.png`：能力、部署信息和维护入口混排。
- `02-before-maintenance.png`：进入维护后原选库上下文消失，作用范围难以判断。
- `03-before-tags.png`：标签建议没有独立日常入口，与人物页签混合。

以上为本轮采集的修改前证据，不作为修改后通过证明；修改后视觉与本地运行结果在下方另行记录。

## 验证记录

- 已执行：Node `22.22.2` 下 `npm run test -- src/features/ai/pages/AISettingsPage.test.tsx src/features/ai/queries.test.ts`，
  2 个文件、50 项通过（设置页 42 项、查询层 8 项）。页面回归覆盖固定导航/旧链接、搜索与人物入口
  保留当前库、无效库不执行其他库操作、分页库入口、跨库错误隔离、开关 ETag、人物不可用条件、
  全量/清理确认、视频进度与可恢复任务边界；标签词表/审核组件由各自测试独立覆盖，不据本组冒称通过。
  查询回归覆盖两库人物 query key、活动/空闲/不可用状态的
  有界刷新，以及 operation 三种终态的建议缓存更新、同 revision 不重复失效和错误记录不冒充终态。
- 已执行：`npm run check:types`（生产与 tooling/tests）和 `git diff --check` 通过。
- Node 22.22.2：全前端 `npm test` 55 文件、293 项通过；包含设置 42、审核 26、标签设置/适配器/
  查询 31、AI 查询 8 项。`npm run check:types`、`generate:check`、`check:architecture`、
  `check:visual-references` 与 `build:storybook` 通过。引用清单检查不冒充全矩阵截图回归。
- `make arch-check`、`make fmt-check`、`git diff --check` 通过。
- 后端：`GOTOOLCHAIN=go1.26.5 go test -count=1 -v ./internal/semantic ./internal/api ./internal/store/sqlite ./internal/aimodel -run 'Tag|ManagementServiceRoutesEveryDerivedSemanticOperation|ManagementServiceFailsClosedWhenSemanticOwnerIsUnavailable'`
  四包通过。覆盖词表 CAS、生成、审核、零建议、取消/恢复和跨库清理隔离。
- 完整 `make test` **未通过**：`spikes/int001-remote-ml/facedecode` 缺本机 pkg-config/libvips；
  第一次下载集成测试因系统盘约 13 GiB 可用低于 10% reserve 而被正确阻断。核心 internal、架构和
  合同包通过。使用独立扩展盘临时目录运行 `GOTOOLCHAIN=go1.26.5 TMPDIR=/Volumes/extended/foliopath-go-tests.o4JmWv go test -count=1 ./tests/integration/...`
  全部通过，4.212 秒；空临时目录随后移除。未运行完整 E2E/质量/容量门禁。
- 本地 Docker：`foliopath-app:ai-workflow-20261022` 最终镜像
  `sha256:478cb8cfa934210707ac16cd61f4853361fd65ce6e4847ecd2fad53ca24604d2`。
  仅替换 `foliopath-ai-original-review` 的应用服务，沿用数据库、loopback 6010 和只读媒体挂载；
  独立 AI 镜像未替换。构建成功（保留 Vite 大 chunk 非阻断告警），实际页面确认新导航/切库/结果入口。
  最终应用和 AI 两容器均 healthy、restart count 为 0；6010 生产页面未记录 console error。
  开发中 6011 的热更新曾触发 Toast context 失配，整页刷新恢复；不将开发热更新日志伪记为零错误。
- 浏览器：设置、任务、服务、标签四页分别在 390/768/1440/1920px 测量，没有页面级横向溢出。
  另检查英文深色设置页的宽屏与手机；发现手机英文页签滚动隐藏入口，改为自然换行后复验。
  标签空状态的重复解释也在本轮修正并补回归。不把这些抽查称为全主题/全语言自动视觉回归。
- 本地候选词表为空，未为演示擅自填入候选标签或生成真实媒体建议。非空卡片、同照片多标签勾选
  由生产 Storybook 合成 fixture 验证；真实模型的匹配质量并未据此提高或通过。
- 原始媒体只读、既有测试数据和所有无关工作区改动保持原有边界；未据本修复关闭质量、容量、
  自动整组归类或发行 Gate。

## 修改后视觉证据

本机运行产物位于 `test-results/ai-ux-2026-10-22/`（不提交真实媒体、日志或数据库）：

- `docker-{settings,tasks,service,tags}-{390,768,1440,1920}.png`：四页面四宽度。
- `docker-settings-en-dark.png`、`docker-settings-en-dark-mobile.png`：英文深色与窄屏页签复验。
- `06-tags-synthetic-selected.png`：合成色块、三条建议/两张照片与本页全选，不是真实推理输出。
- `07-docker-settings-1920.png`：中文宽屏设置页交付截图。

视觉结论限于所列视图：页面利用侧栏外工作区，设置/任务/模型作用域可辨，手机不隐藏主要导航。
原型直接渲染生产组件；没有保留另一份静态设置或标签页面。
