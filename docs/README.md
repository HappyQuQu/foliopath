# FolioPath 文档

- [全部分支集成到main](changes/MERGE-2026-10-26-all-branches.md)：用户明确要求保留全部开发提交；冲突处理和521项前端／Go检查记录。已发布v1.5.0与6010不变。

- 产品介绍与暗色界面导览：[简体中文](../README.zh-CN.md) / [English](../README.md)，含基础浏览、可选智能能力、安装与[截图来源](screenshots/dark/README.md)。

- [v1.5.0正式发布](changes/RELEASE-2026-10-26-v1.5.0.md)：已合并main，正式双原生镜像验证和Docker标签推广完成；安装、已知问题与延期范围以此为当前记录。下列候选/未发布描述保留各阶段历史。

- [提交内容与本地产物过滤](changes/CHORE-2026-10-26-commit-filter.md)：源码、测试和必要素材入库；私有配置、运行数据、依赖与临时构建结果留在本地。

- [本轮固定候选与交付验证](changes/VERIFY-2026-10-26-release-candidate.md)：最终.3／7de02／54673775，约定本地工程与双原生已收尾；最终嵌入UI231通过／54条件SKIP，双架构OCI／SPDX／SLSA本地保留，未正式发布或更新6010。容量24小时取消，质量／漏洞处理暂缓，Git／CI promotion及打包偏差不冒称通过。

- [当前 AI 候选并行发布验收](changes/AUDIT-2026-10-26-release-readiness.md)：本机工程/当前镜像漏洞扫描已收尾；质量数据暂缓，安全/双架构/容量长稳未通过项仍阻断正式发布。

- [全界面用户文案清理](changes/FIX-2026-10-26-product-language.md)：覆盖浏览/设置/账户/媒体库/日志及智能功能；已备份更新6010，保留数据和识别进度。

- [6010 OCR备份部署与小库验收](changes/DEPLOY-2026-10-26-ocr-6010.md)：pixiv最终998成功/1坏源/0待处理，旧结果/任务保留；实际文字与融合来源核验、完整备份后6010布局更新通过，其余库未启用。

- [OCR按库管理界面 CR-2026-052](changes/CR-2026-052-ocr-management-ui.md)：消费现有管理合同，概览开关与任务操作；默认关闭，原6010部署独立。

- [OCR全文索引与融合搜索 CR-2026-051](changes/CR-2026-051-ocr-fulltext-search.md) / [ADR-0037](adr/0037-ocr-fulltext-fused-search.md)：正式搜索组件/有界行证据；默认关闭，管理UI及6010部署独立。

- [OCR后台与显式管理 CR-2026-050](changes/CR-2026-050-ocr-runtime-management.md) / [ADR-0036](adr/0036-ocr-runtime-management.md)：候选backend，默认关闭；搜索/UI及6010部署另行验收。

- [OCR持久结果/任务基础 CR-2026-049](changes/CR-2026-049-ocr-durable-results.md) / [ADR-0035](adr/0035-ocr-durable-results.md) / [内部存储合同](architecture/ocr-persistence-contract.md)：默认关闭，运行worker/API/搜索尚未接入。

- [CR-2026-048：OCR producer与共享调度](changes/CR-2026-048-ocr-producer.md)：实际Go→HTTP→OCR、native取消与缺cache恢复；旧28文件/身份/输出对照，尚未启用主应用索引或搜索。[ADR-0034](adr/0034-ocr-producer-shared-admission.md)。

- [CR-2026-047：OCR安全图片输入](changes/CR-2026-047-ocr-image-input.md)：独立2048/RGB PNG预处理、共享native准入、原生45项与3场景真实OCR桥接；未接入服务/后台索引/搜索。[ADR-0033](adr/0033-ocr-image-preprocessing.md)。

- [CR-2026-046：OCR独立合同与Go传输](changes/CR-2026-046-ocr-transport.md)：严格身份/文本/故障与共享准入；仅传输消费者，未启用后台OCR或搜索。[内部合同](architecture/ocr-transport-contract.md) / [ADR-0032](adr/0032-independent-ocr-transport.md)。

- [CR-2026-045：OCR 离线推理可行性](changes/CR-2026-045-ocr-feasibility.md)：固定中英文模型与隔离Docker实验；横排通过，90°样本未识别，生产索引/搜索仍未接入。

- [CR-2026-044：人物合并冲突预检](changes/CR-2026-044-person-merge-check.md)：只读原因/版本快照、最多三组照片核对，正式提交仍重新校验；本地Docker证据按记录收口。

- [CR-2026-043：同图人物防误合并](changes/CR-2026-043-people-cannot-link.md)：成对人工归属版本保护、同图关系投影与正式单图入口、受保护撤销；独立本地Docker证据。

2026-10-25：[浏览工具栏合并与层次修复](changes/FIX-2026-10-25-browse-toolbar.md)已落实并更新6010；仅关闭全面审查F05中的浏览工具区部分，其余问题保持未完成状态。

2026-10-25：[最新原型一致性审查：整体尚未通过](changes/AUDIT-2026-10-25-prototype-alignment.md)。22入口盘点，确认9项界面偏差及2项合同/运行问题；不以既有功能测试或实现基线替代原型一致性。

- [CR-2026-042：人物详情审核请求恢复](changes/CR-2026-042-person-review-recovery.md)：未知响应同键核对、源404/照片消失仍可恢复并接统一撤销；本地合成与真实模型Docker证据。
- [CR-2026-041：受保护的人脸整理撤销](changes/CR-2026-041-face-review-undo.md)：既有guarded undo接口的统一工作区入口；单回执、确认和同键重试，本地Docker验证，不是全局历史。
- [CR-2026-040：单图人脸整理与排除并发保护](changes/CR-2026-040-photo-face-review.md)：正式单图确认、冻结修订和同键重试；既有POST-MVP-5事务内保护，本地Docker证据，未部署原6010。
- [CR-2026-039：已命名人物目录、照片预览与主区分页](changes/CR-2026-039-people-directory.md)：既有POST-MVP-5人工整理范围；原型/合同/本地Docker证据在同一记录，非身份或发行通过。

2026-10-25：[搜索结果首屏密度修复与6010更新](changes/FIX-2026-10-25-search-density.md)：标题、数量与筛选收紧，图片上移，保留覆盖状态、数据和AI任务。

- [CR-2026-038：人物待处理分组目录与真实计数](changes/CR-2026-038-pending-groups.md)：POST-MVP-5既有人工整理切片；状态与本地Docker证据见记录，不代表发行验收。


2026-10-25：[待整理分组逐张确认](changes/CR-2026-037-cluster-review.md)：当前状态守卫、单脸确认、空人物创建分离、分组类型/游标保持；本地Docker与117项浏览器回归完成，原6010未部署此人物版。

2026-10-25：[库菜单浮层与紧凑标签目录修复](changes/FIX-2026-10-25-sidebar-polish.md)已同步6010，筛选/显示语言/分页合同不变，保留融合搜索、AI及数据。

2026-10-25：[人物详情整理与明确确认](changes/CR-2026-036-person-review-safety.md)：既有人脸合同下的明确选择、移动/移出确认及人物合并核对；开发测试与原6010部署状态分别记录，不冒充完整人物质量验收。

- 已执行运维记录：[2026-10-25 6010融合搜索安全更新与原库验收](changes/DEPLOY-2026-10-25-fused-search-6010.md)；此前[分层界面更新与数据/AI任务保留](changes/DEPLOY-2026-10-25-layered-ui-6010.md)。

2026-10-25：[融合搜索与真实命中来源](changes/CR-2026-035-fused-search.md) / [ADR-0031](adr/0031-fused-search-provenance.md)。
搜索默认合并文件名/父目录与图片语义、服务端去重、固定排名与快照翻页；卡片来源和预览依据使用正式wire证据。
语义故障可明确降级；OCR未接入，视频画面仍走既有独立入口。最终本地Docker/640维真实模型联验、必需Go表面、前端446项与三浏览器90项通过；原6010随后已安全更新并完成原库验收。额外整包race超时与资源压力边界如实保留，见记录。

2026-10-25 05:54终态：[6010的153类标签策略](changes/DEPLOY-2026-10-24-photo-tags-153.md)：
真实Docker的92→153＋png-v4换代、非空人工/删除抑制和故障恢复通过；正常停写完整备份m1T7oV逐文件与Linux完整性核验后，
仅切换AI52f4a40c及tag身份cf642648，原主/语义/人物/三Compose保留。新主database-check、13组保护摘要及22365向量hash一致。
四库新身份原任务全部succeeded/attempt1/error=null，8690成功/10已知源失败/0待处理，无新增model_unavailable。
22PNG、13组保护摘要、22365向量及database-check终态复核通过；10源文件继续暂不修复，后台6010-153已确认删除。下方92类结果为历史记录，质量/容量/发行门禁仍保留。

2026-10-24：[自动标签服务恢复](changes/FIX-2026-10-24-autotag-service-recovery.md) /
[ADR-0030](adr/0030-bounded-autotag-service-recovery.md)：短暂tag服务不可用的60秒有界恢复、严格身份与人工保护；
Go/race/全量/集成/runtime E2E及真实模型Docker故障回归通过，备份后已更新6010；8k/pixiv仅missing续补均结束，21:08终态复核。
四库当前身份/当前源共8690成功、10既有源失败、未处理0；无新增任务级model_unavailable，22PNG与13组保护摘要一致、database-check通过。
10项源问题按用户要求暂不修复；本轮跟进停止，不代表AI全部功能或发行验收通过。

2026-10-24：[自动标签大PNG修复](changes/FIX-2026-10-24-autotag-png.md) / [ADR-0029](adr/0029-independent-tag-png-input.md)：
停写全备份/逐文件核验、Go/race/native及真实模型合成Docker通过；12:17已更新6010主镜像和独立tag profile，
四库新标签任务各自动创建一次；18:26的22张PNG标签验收通过。19:32 coser/4k终态，8k/pixiv遇服务不可用，恢复另见上方记录。
人工/抑制/全部语义向量/人物数据摘要保留；12:17部署未重启AI，19:30双容器另有正常停止再启动记录。
该轮不把已通过的语义22张PNG或旧标签成功数计作新标签完成，不修复/伪造那10项源错误；153类在该轮尚未启用，后续切换见页首。

2026-10-24：[大 PNG 语义输入修复](changes/FIX-2026-10-23-semantic-large-png.md) / [ADR-0028](adr/0028-versioned-png-semantic-input.md)：
native/真实模型合成 Docker 通过；原任务全部终态后完成停写全备份，02:02已更新原6010。
09:16四库新operation全部终态：coser 6200/2、4k 706/1、8k 785/7、pixiv 999/0（成功/失败），共处理8700/8700、成功8690、源输入失败10。
22张目标大PNG全部逐张核对新代次向量及当前源指纹；旧8668向量及15张人物/标签/人工整理表摘要与停写备份一致。
剩余10项需外部恢复或纠正源文件，旧标签输入/失败未改；质量/容量/发行门禁不因此关闭。下文为此前快照。

2026-10-23：[搜索模式提示与中文索引续验](changes/FIX-2026-10-23-search-mode-guidance.md)：
搜索输入/结果卡说明与手机 Header 已修复，Docker 三浏览器五宽度 30 项通过；
原 6010 已停写全备份后更新 81887939，原数据与任务断点保留，不冒充融合搜索或 OCR。
新发现 [13 张大 PNG 的语义输入上限](changes/INV-2026-10-23-semantic-png-limits.md)，
定位时尚未修复；后续候选见上方记录，中文索引和旧 92 类标签仍在续跑。

2026-10-23：[Node 24 LTS 工具链统一](changes/FIX-2026-10-23-node24-lts.md)：本地版本已为
24.22.1，项目/CI/Docker 构建 pin 同步，附实际验证与运行阶段边界，不重启 6010 的在途任务。

2026-10-23 16:47–16:51：[计划升级任务交接修复](changes/FIX-2026-10-23-planned-job-handoff.md)通过
Go/race、旧版迁移和 3 次真实 Docker 正常重启回归；原 6010 已完整备份后更新 e10d3bef/schema 44，
原任务 ID/断点续跑、数据摘要和原会话保留，中文标签页面实测通过。旧 AI/92 类身份保留，153 类尚未启用。

2026-10-23：[CR-2026-033 中文标签与实用类别](changes/CR-2026-033-localized-photo-tags.md)：
中文/英文/双语展示和别名检索，人工原文/同义保护，独立 153 类策略；本地候选与 6010 部署分开记录。
16:17 更新前发现的第三次运行阻碍已由上方 ADR-0027 修复；不是清零重试计数或重建原任务。

2026-10-23：[CR-2026-032 多语言语义模型](changes/CR-2026-032-multilingual-semantic-model.md)。
按用户 Immich 配置接入 XLM-Roberta-Large-Vit-B-16Plus（640 维），不冒用旧 512 维索引；
真实权重合成 Docker/定向回归已通过，6010 已切换；四库新索引运行/排队，不标全库质量完成。

2026-10-23 最新：[剩余 27 张逐图失败原因](changes/FIX-2026-10-23-face-source-diagnosis.md)已定位；
[ADR-0024](adr/0024-shared-large-image-native-admission.md) 大图共享准入/回收候选通过完整合成 Docker，
四库 8,493 项已全部尝试（8,452 成功/41 源错误），终态后已备份并单次更新原 6010，容量/长期门禁保留。
跟进已结束；源文件恢复要求与最后新增三张诊断见[最终记录](changes/FIX-2026-10-23-face-library-retry.md)。

2026-10-23 07:44 用户要求本地 6010 不限制内存：两服务通过显式
[`compose.ai.unlimited-memory.yaml`](../compose.ai.unlimited-memory.yaml) 取消容器 RAM/swap cap，
默认验收预算不变；不是内存峰值修复，宿主/VM 总量仍受限。[操作与验证](changes/FIX-2026-10-23-main-face-memory.md)。

2026-10-23 06:48 新风险：[主应用人物解码内存 OOM](changes/FIX-2026-10-23-main-face-memory.md)。
已由内核确认 06:31 主应用超 1 GiB 后自动重启，AI 服务未重启，人物补齐已断点续跑；内存修复尚未通过，
不能用当前健康状态或此前 AI 子进程共驻结果盖过本次失败。

2026-10-23 当前：[6010 四库人物补齐](changes/FIX-2026-10-23-face-library-retry.md)已按用户授权启动，
共 8,493 项当前代次工作；旧模型完成数不可替代现模型进度。新增历史代次/空源诊断提示，运行结果与验证分开记录。
后台每 30 分钟跟进，未变化保持安静；下文“未重跑原库”是此前维护切片历史，不代表本轮未启动。

2026-10-23 后续：[scan-worker 修复](changes/FIX-2026-10-23-scan-worker-exit.md)复现了宿主机跨内核访问活跃 WAL
数据库引发的 522 短读；[ADR-0023](adr/0023-same-kernel-sqlite-maintenance.md) 固定容器内只读维护方式与日志脱敏。
6010 已更新 0a0244c0，4 库/8,719 媒体保留，原件与 AI 分析未操作。最终 180 秒混合负载/恢复检查通过：
771 扫描、34,642 读取请求、321 容器内检查，自动重启 0；不替代完整 lint/test、长期、质量/容量/发行门禁，
下文为前一切片历史状态。

2026-10-23：[人物失败诊断与扫描等待](changes/FIX-2026-10-23-ai-failure-diagnostics.md)功能实现/专项通过，
但最终 6010 再现 scan-worker 退出自动重启 1 次，当前恢复 healthy、数据库正常；持续运行验收未全通过。
409 项前端、后端/race/合同/生成、集成及合成 Docker 人物/自动标签联验通过；6010 已升级 schema 43，
原库失败详情只读复核通过。历史未知原因不补造，其他 AI 逐图诊断、质量/容量和发行仍未通过。

2026-10-23：[任务页补验](changes/FIX-2026-10-22-ai-task-workspace.md#补验与修复2026-10-23)完成紧凑异常态、英文布局与真实重跑焦点修复；402 项前端及最终镜像本地 Docker 自动标签完整联验通过，6010 已更新。另保留旧候选一次 scan-worker 退出原因待查，不宣称发行通过。

2026-10-22：[智能分析任务页统一列表](changes/FIX-2026-10-22-ai-task-workspace.md)按用户选择的第 1 张设计实施，
接受态原型复用正式组件；395 项前端回归、本地 Docker 与原 6010 页面复验通过。质量/容量/发行门禁不变。

当前实施：[CR-2026-031 自动图片标签与库内标签云](changes/CR-2026-031-automatic-photo-tags.md)。
2026-10-22 用户接受 [V2 原型](changes/PROTOTYPE-2026-10-22-ai-settings-simplification.md)，并授权先实施
自动标签、单图增删与标签云。T1/T2 库内统计和生产标签云已扩展到 T3/T4：独立按库自动标签、AI/人工来源、
删除抑制/手动恢复及有界任务已实现，合同归 [ADR-0021](adr/0021-bounded-inference-and-automatic-tags.md)。
生产 Go/集成/contract/arch 的 `CGO_ENABLED=0` + 扩展盘临时目录命令通过；最终前端 63 文件 387 测试、
types/生成/架构检查与 Storybook 通过，不等于无限定 `make lint` / `make test` 全过。
[同预算模型实测](../spikes/automatic-tags/SCHEDULED-RESULT.md) 18 请求峰值 2084.23 / 2304 MiB、无 OOM，
真实 TCP 断连回收通过。最终镜像双服务故障 smoke `fp-auto-tags-83f1e5ee` 全部通过，含推理中重启、
取消后 35 秒标签不变且不重试、真实停 AI/显式恢复、关闭持久化及原件 hash 不变。真实 UI 的按库开关、
人工增删/转人工、重新识别和重启后来源保留通过；四档宽度无溢出，最终新 tab console error 为 0。
早轮脚本 PID 列错误及普通标签云缓存形状错误已修复并重跑，历史不抹除，详见 CR。
**6010 后续已更新；普通照片质量、容量、AMD64、完整浏览器矩阵与发行验收未完成。** 最终结论统一见 CR 和
[当前进度](features/intelligent-media-discovery-remaining-tasks.md)，原型不是功能证据。

当前界面工作：[智能分析入口与作用域简化](changes/FIX-2026-10-22-ai-information-architecture.md)
（2026-10-22，历史层级修复）。固定功能设置、处理任务、服务与模型三个页签；本轮 CR-2026-031 接通正式图片标签入口，旧词表/审核不再是设置主流程；
本轮逐项证据见[进度表](features/intelligent-media-discovery-remaining-tasks.md)。历史发行结论保持下述状态。

历史 P6（2026-09-17 冻结候选）：**3/3 核对完成，Release No-Go**；当时七项完整工程检查通过，供应链和最终范围已复核，但漏洞、模型授权/provenance 及延期质量容量门禁仍未通过。见 [P6 最终复核](evidence/int-001/p6-final-verification-2026-09-17.md)。同候选 P5 为 **5/5 交付通过**，见 [P5 证据](evidence/int-001/p5-delivery-security-2026-09-17.md)和 [AI Compose 配置说明](ai-compose.md)。当时已完成 [server222 原生 amd64 双容器部署](evidence/int-001/server222-p6-deployment-2026-09-17.md)，不表示本轮自动标签代码已部署；私有部署不改变 Release No-Go。

当前 P4：**本轮验收通过，已完成**（2026-09-17 用户确认）。[约 100 样本冻结候选收口台账](features/intelligent-media-discovery-p4-closeout.md)记录通过范围；人工质量校准与原规模要求延期，不记测试通过。

最新：[冻结候选工程验证](evidence/int-001/p4-frozen-candidate-2026-09-17.md)已完成本地/原生功能、8 组跨架构排序观察、升级回退和有限故障恢复。P5/P6 工程阶段均已执行，正式发行仍为 No-Go。

## 历史增量记录（状态以以上当前台账为准）

最新修复：[大图语义读取取消响应](changes/FIX-2026-09-17-semantic-read-cancellation.md)。先复现取消后仍读完源文件，再修复块间取消及无进展保护；9 项本地 race 和 100 样本功能回归通过。最新候选的 amd64/升级回退尚未重跑，以下双架构结论对应上一固定镜像，不混写。

2026-09-17 最新：[大 JPEG 语义修复及升级补验](evidence/int-001/p4-semantic-large-jpeg-2026-09-17.md)。本地与 server222 原生同一 100 样本完整功能通过，原 36 项语义失败消除；两版镜像合成旧库升级/重启/配对备份回退通过。质量、稳定性与发行未完成，P4 仍 0/8。以下保留早前结果。

最新：[大图内存修复双架构复验](evidence/int-001/p4-face-memory-2026-09-16.md)。新测试内存分配下，本地/server222 同样本人脸 90/90、视频 10/10，无 OOM，原服务健康恢复；36 张图片语义超限仍失败，整体门禁未通过。保留[早前 OOM 失败](evidence/int-001/p4-small-sample-2026-09-16.md)，不是发布或完整质量通过。

当前 P4 执行范围：[约 100 样本验证调整](changes/CR-2026-09-16-p4-small-sample-validation.md)。使用用户指定的本地/远端只读目录，大规模验收延期而非通过；原完整口径进度不变。

P4 已开始：[质量与容量执行记录](evidence/int-001/p4-quality-capacity-2026-09-16.md)，0/8 关闭；基础目录容量修复复验通过，联合 AI/向量/长期稳定性与外部质量数据缺口分开记录，总计仍 18/34。用户已授权 server222 临时停服进行原生 amd64 隔离测试：[维护及测试记录](evidence/int-001/p4-server222-native-2026-09-16.md)，不是部署更新。

最新已完成：[P2/P3 全项复查与修复验收](evidence/int-001/p2-p3-final-recheck-2026-09-16.md)。P2 6/6、P3 9/9；最终本地 Docker 镜像业务/故障/恢复/备份回滚、237 单测、三浏览器及 32 张独立视觉比较通过。当前 **18/34 关闭、16 项保留**，质量/容量/发布未完成。以下均为历史增量。

历史：[人物移动修复与完整复验](evidence/int-001/face-reassignment-2026-09-16.md)。人物原子移动、移出后重新归类修复，234 单测、Docker race/业务及三浏览器在线/离线通过，当时 A05/D03 重新关闭、P3 8/9、17/34。D04 已在上方最新复查关闭。

P2 较早增量：[人物/匿名组生产服务组合](evidence/int-001/face-people-composition-2026-09-16.md)，当时非空闭环未验收，不代表当前状态。

此前：[P1 复查遗漏修复与复验](evidence/int-001/face-busy-repair-2026-09-16.md)。两条 busy 遗漏修复，专项回归及本地镜像探针通过后重新关闭 A02/A03；当时清单 3/34 关闭。保留首次关闭被撤回的记录。

AI 实施方案：[未完成项解决方案与执行计划](features/intelligent-media-discovery-completion-plan.md)，按 P1～P6 覆盖剩余 34 项，先接人脸主链路。

当前 AI 开发入口：[未完成任务清单与进度总表](features/intelligent-media-discovery-remaining-tasks.md)（2026-09-16，34 个剩余工作包；旧清单保留为历史，暂停/豁免事项单列）。

最新（09-16）：[远程人脸激活调度](evidence/int-001/face-deployment-worker-2026-09-16.md)，独立运行时分支及 service 人脸模型迁移已实现；尚非主进程真实激活。

最新：[人脸安全图片适配](evidence/int-001/face-runtime-adapter-2026-09-15.md)，四格式真实离线零人脸及 face/faceonnx 回归通过；代次准入与主进程组合仍待完成。

最新：[Go 人脸服务传输](evidence/int-001/face-go-transport-2026-09-15.md)，共享并发但隔离模型身份，本地 Docker 合同与真实零人脸请求通过；主应用人脸任务仍待组合。

最新：[独立人脸服务合同与验证](evidence/int-001/face-service-contract-2026-09-15.md)，可选 buffalo_l 私有接口及本地 Docker 零人脸推理通过；主应用组合、非空识别和人物验收仍未完成。

最新：[跨库审核清理隔离](evidence/int-001/tag-review-cross-library-2026-09-15.md)，混合请求失效、另一库审核和正式标签保留的 SQLite 回归通过。

最新：[新词条编码请求取消恢复](evidence/int-001/tag-text-request-cancellation-2026-09-15.md)，本地实际服务等待取消及重试通过，明确非 ONNX 内核中断证据。

最新：[标签非空评分与提交取消](evidence/int-001/tag-job-processing-cancellation-2026-09-15.md)，合成向量验证提交边界，未扩展为真实推理验收。

最新：[标签生成实际等待取消](evidence/int-001/tag-job-runtime-cancellation-2026-09-15.md)，非空任务取消及恢复通过，正式标签保留。

最新：[标签生成取消边界](evidence/int-001/tag-job-cancellation-2026-09-15.md)，准入取消与终态优先修复通过，非空实际运行仍待复验。

最新：[非空自动标签审核闭环](evidence/int-001/tag-review-chain-2026-09-15.md)，实际模型建议→接受→清理成功，正式标签保留。

最新：[自动标签生成首次接入](evidence/int-001/tag-generation-wiring-2026-09-15.md)，实际任务成功 1/1；零建议不等于非空审核闭环完成。

最新缺口：[自动标签生成未接入主应用](evidence/int-001/tag-generation-gap-2026-09-15.md)，本地正式任务接口 404；审核清理测试不能代表自动标签功能完成。

最新：[标签审核清理实际镜像验收](evidence/int-001/tag-clear-runtime-2026-09-15.md)，新本地镜像取消及重试通过，非空批次仍属组合测试证据。

最新：[标签审核清理批次取消](evidence/int-001/tag-clear-batch-2026-09-15.md)，非空批次取消收尾及正式标签保留通过。

最新：[标签审核清理准入取消](evidence/int-001/tag-clear-admission-2026-09-15.md)，用户取消有界收尾、停机保留恢复；非空批次交错待补。

最新：[取消修复运行镜像回归](evidence/int-001/cancellation-runtime-refresh-2026-09-15.md)，图片/视频/扫描/清理四类本地实际链路通过。

最新：[清理批次取消修复](evidence/int-001/clear-batch-cancellation-2026-09-15.md)，部分/最终删除批次后取消均通过；不回滚此前已删除的派生数据。

最新：[图片处理器终态组合回归](evidence/int-001/image-processor-terminal-2026-09-15.md)，已提交向量保留、编码失败后取消通过；清理终态交错仍待修复。

最新：[图片终态取消优先](evidence/int-001/image-terminal-cancellation-2026-09-15.md)，修复后续成功/失败收尾对已接受取消的错误处理，Docker 回归通过。

最新：[扫描与清理实际混合取消验收](evidence/int-001/scan-clear-admission-runtime-2026-09-15.md)，新本地 Docker 镜像已验证取消及恢复，批次交错仍待覆盖。

最新：[扫描准入取消收尾](evidence/int-001/scan-admission-cancellation-2026-09-15.md)，复用扫描既有取消规则，保留可靠索引；实际混合场景仍待验证。

最新：[语义清理准入取消](evidence/int-001/clear-admission-cancellation-2026-09-15.md)，区分用户取消与停机，扫描未改动。

最新：[图片等待资源取消实际验收](evidence/int-001/image-admission-runtime-2026-09-15.md)，混合脚本支持图片/视频双向取消。

最新：[图片回填准入取消收尾](evidence/int-001/image-admission-cancellation-2026-09-15.md)，复用通知及原 CAS，扫描状态机未修改。

最新：[实际混合任务等待期取消](evidence/int-001/mixed-admission-runtime-2026-09-15.md)，真实图片运行、视频取消和后续恢复通过。

最新：[视频资源等待期取消收尾](evidence/int-001/video-admission-cancellation-2026-09-15.md)，包含失败复现、修复及其他任务类型边界。

最新：[后台资源取消准入回归](evidence/int-001/background-admission-2026-09-15.md)，共享额度与取消检查通过，实际混合任务和等待期取消收尾待验收。

最新：[旧任务记录实际浏览器验收](evidence/int-001/missing-task-records-runtime-2026-09-15.md)，本地镜像、中英提示和刷新持久化通过。

最新：[旧任务记录提示修复](evidence/int-001/missing-task-records-2026-09-15.md)，明确不存在与临时失败分开，移除仅影响浏览器记录。

最新：[浏览器多库差异进度验收](evidence/int-001/multilib-browser-2026-09-15.md)，含切换/刷新及旧任务残留待修项。

最新：[多库视频调度与取消隔离](evidence/int-001/multilib-cancellation-2026-09-15.md)，含真实 Docker API、重试和按库检索验证。

最新：[视频运行中取消实际镜像验收](evidence/int-001/video-cancellation-runtime-2026-09-15.md)，两种取消路径及后续补齐已在本地浏览器验证。

最新：[视频取消收尾与跨库隔离回归](evidence/int-001/video-cancellation-2026-09-15.md)，修复取消与完成/失败竞态，保留停机恢复语义。

最新：[视频操作按钮与实际浏览器闭环](evidence/int-001/video-actions-ui-2026-09-15.md)，含补齐、确认重建和覆盖率真实性。

最新：[图片未记录到真实回填的实际镜像验收](evidence/int-001/image-unrecorded-runtime-2026-09-15.md)，新数据卷中验证未知→真实 1/1，图片与视频状态分离。

最新：[图片未记录状态](evidence/int-001/image-unrecorded-2026-09-15.md)，与已验收的离线保留进度区分。

最新：[图片模型离线不清零覆盖率](evidence/int-001/image-offline-coverage-2026-09-15.md)，包含实际离线/恢复和拒绝新任务验证。

最新：[视频状态实际镜像及自动恢复验收](evidence/int-001/video-status-runtime-2026-09-15.md)，含自动刷新修复及图片离线 0/0 的未完成项。

最新：[视频只读状态与界面接通](evidence/int-001/video-status-ui-2026-09-15.md)，含未知/离线/降级语义与本地 Docker 回归边界。

最新：[独立服务开关与图片进度范围修复](evidence/int-001/settings-service-toggle-2026-09-15.md)，含 Docker 回归与尚未接通的视频 UI 边界。

最新：[视频删除真实闭环和设置提示复核](evidence/int-001/video-removal-app-ui-2026-09-15.md)，包含人脸路由未接入的真实边界。

最新：[视频移除后的保守覆盖率](evidence/int-001/video-removal-2026-09-15.md)，明确区分待处理重算与逐资产失败归属。

最新：[视频源变化/新增候选](evidence/int-001/video-invalidation-2026-09-15.md)，删除与精确失败归属另列未完成。

最新：[图片新增/移除覆盖率](evidence/int-001/semantic-membership-2026-09-15.md)，与视频尚未完成的计数更新区分记录。

最新：[扫描后图片覆盖率修复](evidence/int-001/semantic-invalidation-2026-09-15.md)，含本地 Docker 过期/恢复闭环及未覆盖边界。

最新：[主进程图片/视频链路](evidence/int-001/app-chain-2026-09-15.md)，包含实际发现的路由缺陷、测试环境缺项及覆盖率待修项。

2026-09-15 最新：[主进程部署配置与离线恢复](evidence/int-001/deployment-startup-2026-09-15.md)。run.go 已接入；下列独立子片的“尚未接入”为当时状态，不再代表当前组合进度。发行、完整产品链路仍待验收。

2026-09-15：[共享激活 worker 与 app 适配验证](evidence/int-001/deployment-worker-adapter-2026-09-15.md)，独立适配通过，正式启动配置尚未接入。

2026-09-15：[服务代次原子激活](evidence/int-001/deployment-activation-2026-09-15.md)事务及真实推理链路已验证；主应用调度未接入。

2026-09-15：[service 模型记录及迁移验证](evidence/int-001/service-model-record-2026-09-15.md)，仅记录与隔离合同完成，主应用激活仍未接入。

2026-09-15：[视频流式编码与真实四帧持久化检索](evidence/int-001/remote-video-stream-2026-09-15.md)已验证；主应用配置与服务代次激活仍待完成。

方向已接受：[ADR-0020 专用 AI 镜像](adr/0020-ai-runtime-delivery-proposal.md)，[内部服务合同](architecture/independent-ml-service-contract.md)。

2026-09-15：[真实离线 AI 回填、数据库重开与检索](evidence/int-001/independent-ml-persisted-chain-2026-09-15.md)已在本地 Docker 通过；生产部署准入仍待完成。

2026-09-15：[旧运行时空间隔离修补](evidence/int-001/independent-ml-legacy-boundary-2026-09-15.md)，本地 Docker 回归通过，独立 AI 主应用接入仍未完成。

SQLite 全包竞态已在本地 Docker 完整复验通过（首次超时后改为 15 分钟预算）；[完整记录](evidence/int-001/local-docker-regression-2026-09-15.md)。主应用独立 AI 接入仍待完成。

本地 Docker 测试入口与本轮结果：[2026-09-15 Docker 回归](evidence/int-001/local-docker-regression-2026-09-15.md)。

2026-09-15：[流式回填与 SQLite 事务提交](evidence/int-001/independent-ml-stream-backfill-2026-09-15.md)已验证；app 仍未切换独立 AI。

独立 AI：[查询/图片共同代次校验](evidence/int-001/independent-ml-bound-encoder-2026-09-14.md)已通过单元与真实 SQLite 组合测试，app/worker 尚未切换。

独立 AI 数据增量：[空间绑定持久化合同](architecture/independent-ml-generation-contract.md)与 migration 38 已实现；不代表新查询/回填已接通。

2026-09-14：[四格式安全编码到真实离线 ML 链路](evidence/int-001/independent-ml-encoded-chain-2026-09-14.md)通过，主应用代次/查询/回填仍未接入。

独立 AI 图片入口：[流式输入与编码合同](architecture/independent-ml-image-contract.md)。独立 imagevips 适配不代表查询/回填或代次迁移完成。

2026-09-14：[独立 ML 缓存/容器复核](evidence/int-001/independent-ml-cache-audit-2026-09-14.md)，仅本地基线和隔离推理，不代表生产身份准入。

独立 AI 增量：[代次兼容性合同](architecture/independent-ml-generation-contract.md)已实现纯检查及合成测试；模型事实核验和生产接入仍待完成。

2026-09-14 正式智能设置已按方案一重排既有接口消费者；[视觉与测试记录](../design-qa.md)。不代表新独立 AI 服务已接入。

2026-09-14 AI 新方向：[ADR-0019](adr/0019-independent-machine-learning-service.md) /
[CR-2026-030](changes/CR-2026-030-independent-machine-learning-service.md)。独立服务本地探针已验证；
生产接入未完成。模型商店旧路线暂停，历史交付状态不能当成新架构验收。

内部通信子片：[独立 ML 传输合同](architecture/independent-ml-transport-contract.md)，
Go adapter 位于 `internal/inference/remoteml`；尚未接入主应用代次。
验证记录：[Go 传输子片](evidence/int-001/independent-ml-go-transport-2026-09-14.md)。
选定的 AI 设置[方案一交互原型](../spikes/int001-ui-settings/README.md)已本地运行，
与生产代码导入图隔离；参见其[视觉检查](../spikes/int001-ui-settings/design-qa.md)。

最新模型代理界面扩展：[CR-2026-029](changes/CR-2026-029-admin-model-proxy-settings.md)；属于 POST-MVP-5，不代表已部署。

本目录保存 FolioPath 的产品基线、架构约束、交付计划和验收证据。文档按用途分层：
先从本页找到当前权威入口，再进入专题索引；不要从文件数量或日期推断文档是否仍然有效。

当前状态：MVP 与后续视频预览、自动发现、运维可观测性、收藏/标签切片已有生产实现；各版本
是否达到稳定发布仍分别由对应 Gate 决定，不能由实例部署替代。MVP 冻结范围是
[MVP scope revision 4](releases/MVP-2026-07-23-scope-r4.md)，最新已实现公共合同基线为
[`POST-MVP-4`](releases/POST-MVP-4-scope.md)。本轮综合更新见
[2026-08-13 更新说明草案](releases/2026-08-13-integrated-update-notes.md)。
2026-10-23 的 Post-MVP 显示优化见[无缝瀑布流](changes/CR-2026-033-seamless-masonry.md)。
同日发布工具维护见[原生构建与 digest 验证](changes/FIX-2026-10-23-native-release-builds.md)。
Node 与 Docker 前端构建工具链同步见[Node 24 LTS 维护](changes/FIX-2026-10-23-node24-lts.md)。

同日显示纠偏与大小滑条见[CR-2026-034](changes/CR-2026-034-gallery-size-and-source-icon.md)（Confirmed / Post-MVP 1.4.x）。

## 首先阅读

2026-09-08：[模型下载与智能界面扩展 CR-2026-026](changes/CR-2026-026-ai-model-download-and-ui.md)
实现了部署目录驱动的界面下载、后台进度/取消/重试与智能概览。默认下载源仍未配置，当前发行镜像
仍无 AI 推理组件；该扩展不把历史 S4 未验证项转为通过。配置见[模型下载目录](deployment.md#模型下载目录)。

| 目的 | 当前入口 | 说明 |
| --- | --- | --- |
| 了解产品 | [项目 README](../README.md) | 面向用户的能力、部署和限制 |
| 确认产品范围 | [产品需求](product-requirements.md) | 需求语义；冻结范围以 scope manifest 为准 |
| 了解系统结构 | [架构索引](architecture/README.md) | 模块、运行视图、前端边界、追踪和适配度检查 |
| 查看 API | [OpenAPI](../api/openapi.yaml) | 已实现 API 的权威合同 |
| 设计或修改界面 | [界面设计规范](ui-design.md) | 导航、组件、响应式、可访问性和动效 |
| 部署或运维 | [部署](deployment.md) | 容器、卷、权限、备份、恢复和升级 |
| 查看用户更新 | [更新日志](../CHANGELOG.md) | 自动维护的用户可见新增、改进、修复和注意事项 |
| 查看当前工作 | [开发任务清单](task-list.md) | 当前进度、阻断和下一步 |
| 判断能否发布 | [版本与 readiness](releases/README.md) | scope、候选状态和发布说明 |

## 按文档类型浏览

| 类型 | 索引 | 用途 |
| --- | --- | --- |
| 架构 | [architecture/](architecture/README.md) | 当前系统结构、模块所有权和治理 |
| ADR | [adr/](adr/README.md) | 只追加的架构决策及替代关系 |
| Feature | [features/](features/README.md) | 独立用户能力的范围、合同和交付阶段 |
| 变更记录 | [changes/](changes/README.md) | Change Record 与已批准 slice 内修复 |
| Gate | [gates/](gates/README.md) | Architecture、Contract、Backend、UI 和 Release 判断 |
| 版本 | [releases/](releases/README.md) | 冻结 scope、readiness 和发布说明 |
| 验收证据 | [evidence/](evidence/README.md) | 可复核的视觉、浏览器和容量证据 |
| Spike | [spikes/](spikes/README.md) | 可行性实验；不能代替生产 Gate |

当前另有一套不影响冻结 MVP 的 [FTR-INT-001 本地智能媒体发现提案](features/intelligent-media-discovery.md)：
已形成[技术架构](architecture/intelligent-media-discovery.md)、[spike 计划](spikes/int-001-ai-feasibility.md)
和[任务清单](features/intelligent-media-discovery-task-list.md)，并补充审核下载源与 `/models:ro` 离线
模型获取方案；`POST-MVP-5` [revision 2](releases/POST-MVP-5-scope-r2.md) 已冻结 A～E 完整范围；
INT-S0 与 [INT-S1](gates/POST-MVP-5/int-s1-contract-ready.md)仍只对 A+B 为 Go；
[INT-S2A](gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
[INT-S2B](gates/POST-MVP-5/int-s2b-backend-evidence-ready.md) 与
[INT-S2C](gates/POST-MVP-5/int-s2c-privacy-ready.md) 均为 Backend Ready / Release No-Go；
[INT-S3](gates/POST-MVP-5/int-s3-consumer-ui-ready.md) 已为 Consumer/UI Ready / Release No-Go；
[INT-S4 current](gates/POST-MVP-5/int-s4-current.md) 已验证完成恢复、原件只读与发布文档 3/11；产品用户按
[CR-2026-025](changes/CR-2026-025-int-s4-unverified-closure.md)将其余 8/11 未验证关闭，S4 终态为
Closed Unverified / Release No-Go。按
[CR-2026-022](changes/CR-2026-022-s2-backend-release-gate-separation.md)，最终真实质量、native 双架构、
联合容量、供应链与隐私/合规签署由 S4 持有。国内镜像仍没有承诺；S4 Go 前 reviewed catalog、face
composition 与发行 UI 继续失败关闭。

## 界面原型与实施

- [分层工作台：正式界面对齐原型](changes/UI-2026-10-25-layered-workspace-alignment.md)：用户接受整体视觉，生产组件/真实后端合成验收与 Storybook 同步。

- [按功能切换上下文侧栏（生产实施）](changes/UI-2026-10-24-contextual-sidebar.md)：用户已接受，生产组件/Storybook/回归保持一致；[原型](changes/PROTOTYPE-2026-10-24-contextual-sidebar.md)作为历史接受证据。

## 当前权威文档

### 产品与体验

- [产品需求](product-requirements.md)
- [需求确认记录](requirements-checklist.md)
- [用户流程](user-flows.md)
- [界面设计规范](ui-design.md)
- [品牌标识规范](branding.md)
- [路线图](roadmap.md)
- [术语表](glossary.md)

### 工程与运行

- [系统架构](architecture.md)
- [目录与依赖约束](project-structure.md)
- [数据模型](data-model.md)
- [API 设计说明](api-design.md)：解释资源边界；wire 合同以 OpenAPI 为准
- [安全模型](security.md)
- [部署](deployment.md)
- [测试策略](testing-strategy.md)
- [供应链与许可证审查](supply-chain-review.md)

### 计划与风险

- [开发任务清单](task-list.md)
- [后端开发清单](backend-task-list.md)
- [前端开发清单](frontend-task-list.md)
- [开发就绪评审](development-readiness.md)
- [可行性研究](feasibility-study.md)
- [风险登记](risk-register.md)

已完成计划仍可作为交付记录，但不应优先于当前产品、架构、OpenAPI、migration 或 Gate
事实。计划状态发生变化时，应更新计划本身或对应专题索引，避免创建新的平行总清单。

## 历史文件如何处理

- scope revision、ADR、Change Record、Gate 和发布证据是审计历史，采用只追加策略。
- 被替代的文件保留原路径，并在文件头或所属索引中指向当前替代项。
- 已完成任务清单保留任务与证据映射；后续工作进入新的 feature 或版本清单。
- `docs/evidence/` 和 `web/qa/` 中相同内容可能代表不同验收状态，不能仅按哈希去重。
- 只有未被任何当前或历史记录引用、且不承担审计含义的生成物或草图才可删除。

## 状态含义

- **已接受 / 已确认**：当前实现必须遵守。
- **冻结**：只追加新 revision，不原地改写范围。
- **Go**：适用检查已经实际执行并通过。
- **Conditional Go**：只允许记录中明确列出的有限下一步。
- **No-Go / Pending**：不得宣称切片或版本完成。
- **提案 / 未来**：不授权进入当前生产实现。
- **已被替代**：保留历史，但实现应遵循其替代文件。

发生冲突时，依次检查冻结 scope、已接受 ADR、OpenAPI/migration、当前 Gate 和产品需求；
不要静默选择其中一份，也不要通过删除旧记录消除冲突。

## 变更同步表

| 变更类型 | 至少同步更新 |
| --- | --- |
| 用户可见功能、格式或配置 | `README.md`、产品需求、相关用户流程、Change Record |
| 媒体库路径或删除语义 | 产品需求、架构、数据模型、安全模型，必要时新增 ADR |
| API 路径、字段或错误 | `api/openapi.yaml`、生成客户端、相关合同测试 |
| 页面、组件、响应式或动效 | 界面设计规范、相关用户流程、视觉证据 |
| 数据表、索引或迁移语义 | 数据模型、migration，必要时新增 ADR |
| Docker、权限、备份或升级 | README、部署、安全模型 |
| 测试门槛或支持矩阵 | 测试策略、readiness、README |
| 可行性假设、spike 或风险状态 | 可行性研究、风险登记、开发就绪评审 |

2026-10-26：[标签侧栏层次修复](changes/FIX-2026-10-26-tag-sidebar.md)。

2026-10-26：[侧栏调宽后内容自适应修复](changes/FIX-2026-10-26-sidebar-reflow.md)，现有界面响应验收修复，API与任务合同不变。

2026-10-26：[标签云排列修正](changes/FIX-2026-10-26-tag-cloud-layout.md)，按用户澄清恢复自由换行词云。

- [OCR限制与SQLite恢复维护](changes/FIX-2026-10-26-ocr-limits-sqlite.md)：本地候选、停写数据保护与后续新OCR代次。

- [智能分析四功能统一布局](changes/FIX-2026-10-26-ai-feature-grid.md)：生产组件预览与响应式验证，待现有OCR任务结束后更新6010。

- [干净源码、正式 CI 与 6010 更新验收](changes/VERIFY-2026-10-26-formal-ci-6010.md)

- [长媒体库名称上下文栏修复](changes/FIX-2026-10-26-context-name-wrap.md)

- [自助AI部署变更与验证](changes/CR-2026-053-simple-ai-deployment.md)：开发分支，配对镜像尚未发布。
