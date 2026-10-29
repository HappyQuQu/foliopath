2026-10-29 / ARCHIVE-S2候选：主镜像新增Debian 7zip 25.01+dfsg-1~deb13u2、prlimit与依赖及包许可证，不增加服务、挂载或端口。7z依旧位于只读/library，目标图片临时文件沿用/app/data/tmp/zip；密码/分卷/自解压包不支持，固实包可能较慢或达到资源额度。

2026-10-29：222 主应用已升级到固定正式 digest 的 v1.8.0；停机完整备份、旧镜像独立恢复校验、在线完整性与前端一致性检查通过。现有 AI 合同兼容，容器未重启。见[v1.8.0 发布与升级记录](changes/RELEASE-2026-10-29-v1.8.0.md#222-主应用升级)。

2026-10-29 / ZIP-S1：不增加挂载或服务，ZIP 仍位于单一只读媒体树；请求临时图片位于 /app/data/tmp/zip，最多两个 64 MiB 成员，关闭回收/启动清遗留。持久缩略图继续占既有缓存配额。新增来源迁移，现有实例升级前仍需停写完整备份；本地验证使用独立合成实例，正式镜像依现有原生双架构门禁发布；升级现有实例需另按运维流程执行。见[CR-2026-055](changes/CR-2026-055-zip-image-browsing.md)。

# FolioPath 候选部署与运维

2026-10-28：222 主应用已由 v1.6.4 升级至固定正式 digest 的 v1.7.0；现有 AI 容器、环境配置及只读媒体挂载不变。停写完整备份、独立恢复校验和在线检查通过，见[发布及升级记录](changes/RELEASE-2026-10-28-v1.7.0.md)。此状态与下文6010本地实例的历史记录分别维护。


2026-10-28：[v1.6.4 已正式发布](changes/FIX-2026-10-28-browse-navigation-dock.md)，主镜像 `evanqu/foliopath:1.6.4` 与可选 AI 镜像 `evanqu/foliopath:1.6.4-ai` 均通过 AMD64／ARM64 原生验证和索引推广。包含快速访问折叠记忆与滚动紧凑工具栏过渡；无 API、SQLite 或模型变更。本次未替换其他实例，本地 6010 合成测试容器继续保留。以下记录为历史发布及部署状态。

2026-10-27：[v1.6.3 已正式发布](changes/RELEASE-2026-10-27-directory-layout.md)，主镜像 `evanqu/foliopath:1.6.3` 与可选 AI 镜像 `evanqu/foliopath:1.6.3-ai` 均通过 AMD64／ARM64 原生验证和索引推广。子目录网格优化已包含；仅恢复现有构建／发行验证必需的 25 个文件，普通测试仍仅本地保留。本次没有替换已有部署实例，本机 6010 仅为独立合成数据测试。以下旧部署记录保留历史状态。

2026-10-26最终发布状态：[v1.5.0已正式发布](changes/RELEASE-2026-10-26-v1.5.0.md)，主镜像 `evanqu/foliopath:1.5.0` 同时支持AMD64／ARM64，正式双原生验证及索引推广通过。6010本次保持已验收rc-20261026.7（主ca3b43／AIa87f73），没有再次更新或停服；数据、任务与五份Compose保留。可选AI镜像按Release中独立平台digest选择，不把主应用标签当作AI标签。升级与回退仍须完整停写备份／独立恢复核验。

以下到“部署目标”之前保留早期部署快照，其中“当前／未发布／未部署”只代表当时阶段；现状态以上段及正式发布记录为准。

2026-10-26当前实例状态：6010仍为主b3419001／AI7739486c，pixiv OCR补齐已终态998成功、1源损坏、0待处理；旧结果、任务、人工标签和删除记录保留，其他库未开OCR，原10源问题继续跳过。完整data位于经停写复制核验的Linux命名卷。状态、启停、备份或更新均须保留同一项目／私有env及**五份Compose**：base、unlimited-memory、multilingual、ocr、data-volume；原件只有单一`/library:ro`，AI不挂data或媒体，禁止恢复启动旧宿主data副本。AI曾为OCR策略重建，后续主应用维护未重启AI。当前固定候选.3已在独立环境完成约定的本地工程／双原生验证和交付准备，未部署到6010，也未正式发布；最终范围／镜像／归档及54项浏览器条件SKIP见候选记录。见[OCR部署](changes/DEPLOY-2026-10-26-ocr-6010.md)和[候选验证](changes/VERIFY-2026-10-26-release-candidate.md)。下文CR记录保留各阶段历史，不能把旧“23张／未部署”当当前状态。

CR-2026-051：migration47新增OCR派生全文表/FTS，启动有界回填已有ready结果，不重新分析、不自动开库；升级仍需遵守已有停写完整备份规则。OCR-only与融合文字检索只在显式部署ocr.identity匹配、库启用后返回文字，不依赖AI当前在线。OCR默认关闭、正式管理界面与原6010切换独立；本片只在合成Docker验证。清理同时移除全文派生文档，关闭则保留结果。

2026-10-26 / CR-2026-050：候选FOLIOPATH_AI_DEPLOYMENT可显式增加ocr.identity（完整ADR-0032字段，输入/输出hash必须匹配本版），与既有Origin共享客户端预算。未提供则不注册OCR管理/worker；配置也不自动启用各库或重新识别。候选AI服务须独立OCR权重只读cache与一次性验证scratch（ADR-0034）。迁移46默认不启用。计划退出使用既有signal标记handoff，普通崩溃保留attempt收费；原6010部署/备份切换不在本片。

2026-10-26 / CR-2026-049：候选迁移00045仅创建休眠OCR派生表，无模型激活/自动任务。OCR正文将属于/app/data备份范围；未来原库部署仍需正常停写完整备份与恢复核验。此次不更新6010、不切模型/任务；清理为逻辑删除，不删除已有备份。

2026-10-25 [浏览工具栏修复](changes/FIX-2026-10-25-browse-toolbar.md)已更新6010；冻结已部署后端，仅更新四个浏览/集合前端文件；完整停写备份、保护摘要及线上静态文件/数据库检查通过，AI容器未重启。

2026-10-25：[搜索首屏修复](changes/FIX-2026-10-25-search-density.md)已更新6010，使用冻结部署源码只替换搜索排版；完整停写备份、程序复现、线上database-check及27组数据/AI保护摘要验证通过，AI容器未重启。

2026-10-25：[库菜单浮层与紧凑标签目录修复](changes/FIX-2026-10-25-sidebar-polish.md)已同步6010，筛选/显示语言/分页合同不变，保留融合搜索、AI及数据。

2026-10-25：6010已应用[分层工作台界面](changes/DEPLOY-2026-10-25-layered-ui-6010.md)。
先逐字节重现旧后端，再仅更换嵌入SPA；主新镜像049571c1，AI52f4a40c不重启。
完整停写备份及45条AI任务/27组保护摘要/资源字节验证见该记录；无schema或模型身份变更。

自动标签短暂服务不可用按[ADR-0030](adr/0030-bounded-autotag-service-recovery.md)在原claim内有界恢复，
首次临时错误后最多60秒、最多2次额外分析，严格校验身份，不依赖Compose启动顺序阻塞核心浏览。
历史终态不会自动恢复：确认模型已恢复后，管理员通过带修订保护的启用设置解除阻止，既有missing仅补未处理部分。
不能清空已完成run、直接改claim表或用all替代补齐；源失败与人工整理保留。

## 独立自动标签 PNG 输入（ADR-0029）

`FOLIOPATH_AI_DEPLOYMENT`的`tag.image_preprocessing`可为空（原v3）或`png-v4`；未知值启动失败。
它与顶层`image_preprocessing`独立，仅改变标签输入和标签身份，不改变语义/人脸或标签词表/模型。
已启用库按新身份自动missing补齐，旧标签不预清空，人工标签和删除抑制保留；不可把旧run算新覆盖。
切换前须无旧活动标签任务，停写完整备份data和匹配私有配置并验证。回退需要配对备份，不只是换旧镜像。
保留base + unlimited-memory + multilingual覆盖、只读原件和原AI镜像。实际部署状态见[本轮记录](changes/FIX-2026-10-24-autotag-png.md)。

## 可选大 PNG 语义输入（ADR-0028）

支持候选中的 `FOLIOPATH_AI_DEPLOYMENT.image_preprocessing="png-v4"`（JSON 字段，不是独立环境变量），
缺省/空值保持 v3，未知值失败关闭。仅语义 PNG 扩到 64 MiB / 40 MP；人物/自动标签与 AI 镜像无需改动。
预处理变化创建新语义代次；旧向量保留但不计新覆盖，需要显式重新分析。等待现有任务终态、停主应用完整
备份 data/私有配置后再切换；保留所有现有 Compose 覆盖，特别是用户选择的 unlimited-memory 与 multilingual。
回退使用匹配镜像/配置/停写全备份，不能仅改回字段就承诺恢复旧活动代次。
本地6010已于2026-10-24在终态和停写全备份核验后切换，四库新代次各提交一次；09:16全部终态，
处理8700/8700、成功8690、源输入失败10。全部22张目标PNG均核对新代次向量及当前源指纹；旧向量与15张保留表摘要仍与备份一致。
剩余源文件由用户在应用外恢复/纠正，不能盲目重试清零；自动标签输入和失败数不随语义修复改变。
主/AI均healthy、restart=0/OOM=false，容器内database-check通过。备份、固定镜像、代次、operation与逐项恢复要求见[记录](changes/FIX-2026-10-23-semantic-large-png.md)。

## 有活动分析任务的计划升级（ADR-0027）

新版本正常 SIGTERM/SIGINT 时，图片语义/人物/自动标签 worker 等处理器和 heartbeat 退出后，
以原 claim fence 交回同一任务，记录 schema 44 审计并撤销该次 claim 消耗；不清零历史 crash 次数。
取消、终态和旧 fence 不复活；强杀/OOM 仍按原 3 次上限恢复。退出非零或超时不能当交接成功。
先确认主应用已停止、完整备份 `/app/data` 与匹配配置，再替换镜像；不要删库或重新全量提交来升级。

从不支持此逻辑的旧版本首升，必须在确认 Docker owner 停止/完整备份后，由 Linux 离线工具
`spikes/job-handoff` 对停写精确 claim 快照执行一次 canonical 交接，不能仅假定旧 SIGTERM 已交回。
工具不进入生产镜像或浏览器 API，不得对运行中的 owner 使用写模式；失败可能已完成 schema 升级，
回退需用完整停写备份和旧镜像/配置。不要从宿主打开原挂载 SQLite（停止后也优先用容器核对）。
旧策略任务在运行时保持旧 AI/身份，新类别策略另行切换。当前 6010 部署/备份与续跑证据见
[交接修复记录](changes/FIX-2026-10-23-planned-job-handoff.md)。

## 中文标签候选（ADR-0026：标签策略）

主应用本地化展示兼容原 92 类服务，不需改旧标签或重跑；153 类识别扩充另需使用固定
`services/inference/Dockerfile.photo-tags` 覆盖已核验的多语言镜像，离线 `python -m photo_policy`
读取新独立标签身份并仅更新部署 JSON 的 tag.identity。搜索/人物字段保持原值，仍只有两个服务。
语言选择归浏览器，不写 Compose。不得只改策略名称而沿用旧摘要；启用库的新身份补齐保留人工和抑制。
更换运行服务前遵循下方停主应用/完整备份、AI healthy 后再恢复的顺序；不要用反复重启打断原库任务。
本地测试候选与 6010 当前部署状态分别见 [CR-2026-033（本地化图片标签）](changes/CR-2026-033-localized-photo-tags.md)。

## 多语言语义覆盖（ADR-0025）

XLM-Roberta-Large-Vit-B-16Plus 640 维需固定覆盖镜像、字节清单与新语义索引，不能只改模型名。
AI 不挂 DB/媒体；新增 volume 仅为可丢弃 scratch，不是备份真相。更新保留 multilingual 及不限内存覆盖。
有在途任务时，先停主应用完成全备份，再更新 AI 等 healthy 后启动主应用，避免恢复任务早于 AI 就绪。
保留旧镜像/配置/完整 data；回退也不重标向量。仅新语义需要补齐，人物/人工标签不清理。
操作、资源与剩余门禁见[CR-2026-032](changes/CR-2026-032-multilingual-semantic-model.md)。

## 本地 6010 不限容器内存（2026-10-23 用户选择）

后续 [ADR-0024 大图共享准入/回收候选](changes/FIX-2026-10-23-main-face-memory.md) 已通过固定合成 Docker，
四库终态后已完成停写全备份并单次替换 6010 主应用为 c9f28d7a；保留本节的不限额覆盖和原 AI/身份，
没有为升级重复提交/清除原分析。健康/容器内完整性/原任务终态复核通过，详见链接中的备份与操作记录。
本地候选部署不是所有容量/长期或发行批准。

本地 `foliopath-ai-original-review` 使用 `compose.ai.yaml` 加
`compose.ai.unlimited-memory.yaml`，显式取消主应用与 AI 的 RAM/swap 容器上限；
不是默认发行预算改变，也不宣称已修复解码内存峰值。Docker Desktop/宿主总容量仍生效。
CPU/并发、输入大小、PID、超时、tmpfs、只读媒体和内部网络不变。
后续同实例更新必须保留该覆盖文件；操作及验证见[AI Compose 说明](ai-compose.md#操作者显式取消容器内存上限)。
下文基准 1024/2304 MiB 和历史测试结果仍属于有界配置，不能套到当前不限额部署。

## 在线 SQLite 检查（ADR-0023，2026-10-23）

Docker Desktop 的 macOS/Windows 宿主机不得直接打开运行中的 bind-mounted SQLite 数据库，即使只执行
quick_check 或使用 mode=ro，也不能假定与 Linux VM 共用锁域。隔离合成库已复现 I/O 短读（522）并使
scan-worker fail-closed 退出。请在拥有数据的主应用容器内执行：

```sh
docker exec foliopath-ai-original-review-foliopath-1 /app/foliopath database-check
```

成功仅输出 `database check passed`；失败非零退出并输出固定错误，不输出原始 SQL、表名或路径。
该命令固定 `/app/data/foliopath.db`、Linux-only、mode=ro、15 秒上限，在一个只读快照执行 quick_check 和
foreign_key_check，不启动应用/worker/模型，不迁移、checkpoint、VACUUM 或尝试修复。readiness 快检不改变。
不要给命令传主机路径；不要在 AI 服务挂载数据库。备份须先停止主应用，再完整复制其数据目录；
仅已停止写入的独立备份副本可以在宿主机检查。下文此前宿主机检查结果只作为历史记录，不能照抄为在线操作。
[故障复现与本地验收](changes/FIX-2026-10-23-scan-worker-exit.md)。

本机 6010 后续已更新主镜像 `sha256:0a0244c0f3d5d9d637cf4ed48746d8d273e723e0a1b136c3ce07bf9325b0c917`，
AI 保持原镜像，无迁移（43 不变）。停写备份以 `backup-6010-scan-worker-verified-20261023-FBogJQ` 为准；
较早的 o6klZU 目录未作可靠备份使用。详情及首轮备份检查失败原因见修复记录。离线 WAL 库的只读数据库连接
可能仍需要目录允许 WAL/SHM sidecar，完全只读目录不等于可以忽略 WAL；不要对活跃数据使用 immutable=1。

## 人物失败诊断升级（2026-10-23 / ADR-0022）

迁移 43 增加派生失败记录，不改原媒体、人工标签或人物关系。必须保留 `/app/data` 的完整停写备份；
回退需先停止新主应用，再配对恢复旧镜像、旧 env 和完整旧数据，不能把 schema 43 数据直接交给旧程序，
也不能通过手删新表降级。AI 服务镜像/模型身份本次不变。

原 6010 使用 `foliopath-ai-original-review` 项目更新，备份为
`/Volumes/extended/foliopath/backup-6010-face-diagnostics-20261023-Q9Ah1V`（旧 env＋schema 42 全数据，
备份 immutable quick_check=ok）。新库 quick_check=ok、外键检查为空、schema 43、4 库/8,719 媒体；
最终主镜像为 `sha256:ce6911fb7c63f4974f21043bdd5463ac33d0a80cce65e598180f6a4731e9f7b5`，
AI 为 `sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。
初检双服务 healthy、restart=0/OOM=false；最终主应用 scan-worker 退出后自动重启 1 次，现恢复 healthy，
主应用 restart=1、AI=0/OOM=false。底层原因未查明，持续运行验收未通过；数据库仍完整、8,719 项保留。
未向原库发起分析/清理，未实际执行回退。固定镜像与回归见
[本次记录](changes/FIX-2026-10-23-ai-failure-diagnostics.md)。本机部署不代表 server222 更新或发行批准。

## 自动标签候选升级（2026-10-22 / ADR-0021）

本节描述 [CR-2026-031](changes/CR-2026-031-automatic-photo-tags.md) 当前实现的部署合同，不声明
任何现有环境已更新或发行已批准。仍使用根 `compose.ai.yaml` 的两个服务，不增加 worker 部署。
主应用 1024 MiB / AI 2304 MiB、AI 私有快照 tmpfs 1200 MiB、禁 swap 与只读挂载预算保持不变；
有界推理子进程属于 AI 容器内部，不是第三个服务。

- 用 `services/inference/Dockerfile` 的 `FOLIOPATH_INCLUDE_TAG=1` 显式构建可选 JoyTag 能力；默认不含。
  这不下载或内置模型权重。只读缓存须具有固定 `tagging/JoyTag/` 三个文件，摘要及完整准备步骤见
  [AI 镜像说明](../services/inference/README.md#bounded-inference-and-automatic-tags-adr-0021)。
- 主应用 `FOLIOPATH_AI_DEPLOYMENT` 可增加 `tag: { identity: ... }`；identity 包括 version=1、
  protocol=`foliopath-tag-v1`、model=`JoyTag`、model/policy/runtime/cache SHA-256。tag 缓存身份独立于
  CLIP/face，runtime 必须匹配同一服务。请从固定候选镜像离线导出预期身份，不接受未验证的在线响应。
  未配置模型时原有标签和普通图库仍可用，自动标签显示等待；没有界面模型下载或按库模型选择。
- 运行时代码/固定 manifest 改变会改变搜索、人脸及标签兼容身份，即使模型文件相同也不能沿用
  原部署 JSON 或把旧索引重标为新代次。更新配对 app/AI 镜像和预期身份后，按新身份重新分析；
  旧索引/人物应用状态依各 owner 保留规则处理，旧完成数不计入新身份，禁止直接改数据库绕过验证。
- 自动标签的 model identity 还包含主应用图片预处理合同。已启用库有界补齐新图、源变更和新模型；
  每库一个活动任务、全局最多 32。关闭/取消保留已提交标签；任务失败、取消或最多三次领取尝试
  耗尽会阻止同模型自动重入，重启/服务恢复不会解除。管理员确认重新识别（all）或显式重新启用
  后恢复调度；同源同模型已记录的逐图失败需 all 重试，不能靠每 30 秒调度无限重试。
- 正常推理槽暂忙不封锁模型：同图最多 3 次尝试（1/2 秒可取消等待），仍忙则原任务延后 30 秒，
  释放后台资源后自动继续；保留同一 operation 和已提交进度，不消耗崩溃恢复次数，不生成任务风暴。
  暂忙探测不把已验证模型标为失联；从未验证成功的模型仍显示等待。
- 升级前按 SQLite WAL 安全方式备份 `/app/data`；migrations 41/42 包含标签来源、人工删除抑制和
  有界任务，不能当缓存丢弃。旧关联升级为 manual；源变更只使 AI 关联失效。回退必须配对旧镜像
  与升级前停写备份，不能把已升级数据库直接交给旧镜像或主动降级删除人工保护状态。

推理按请求冷验证/复制/加载并回收，吞吐和峰值须在同预算实测；参考
[ADR-0021](adr/0021-bounded-inference-and-automatic-tags.md)，最终 Docker/原件 hash/重启证据由 CR 汇总。
下文 P3 的“直接重试”仅适用于原搜索/人物任务；自动标签的失败/取消恢复以上述规则为准。

P5 新交付入口：根 [`compose.ai.yaml`](../compose.ai.yaml) 为独立双服务候选配置，不叠加根单容器 Compose。配置、离线镜像/身份准备和运行限制见 [AI Compose 部署说明](ai-compose.md)。P5 5/5 已通过本地工程验收，[最终证据](evidence/int-001/p5-delivery-security-2026-09-17.md)。下文旧 spike 配置仅保留为历史验证，不再作为正式双服务入口；P6 发行批准不因 P5 完成而自动通过。

P4 大 JPEG 兼容候选采用语义预处理 `remote-image-v2`，因此即使 AI 模型文件不变，主应用的 semantic 空间身份也会改变；升级后须按新代次重新分析图片/视频及生成标签建议，不能将旧向量强行重标为新代次。人脸输入身份不变。这不是自动更新线上部署；旧镜像回退仍配对升级前备份。[修复范围](changes/FIX-2026-09-17-semantic-large-jpeg.md)。

P3 排障补充：AI 中断/超时的失败任务保留历史；服务和精确身份恢复后直接提交新重试，无需切换库开关。若修改 `FOLIOPATH_AI_DEPLOYMENT` 身份，新身份验证失败会使同用途旧 service 模型不可用，而不是继续展示假可用；不会删除旧索引或人工关系。恢复正确配置后等待 availability 刷新再重试。迁移 40 存在 service 人脸记录时禁止有损降级，回退必须使用对应旧镜像和升级前停止写入后的备份，不把升级后的数据库直接交给旧版本。

### 可选人脸启动配置（2026-09-16，开发候选）

现有 `FOLIOPATH_AI_DEPLOYMENT` JSON 可增加 `face` 对象：`identity`（version=1、protocol=`foliopath-face-v1`、model=`buffalo_l`、dimension=512 及 runtime/cache hash）、`detector_sha256`、`embedder_sha256`、`threshold`、`model_bytes`、`license_id`。
`threshold` 使用现有 schemaVersion/profileId/coreSimilarity/edgeSimilarity/minCoreSize/qualitySummarySha256 合同，必须显式提供，不能照抄测试阈值作为质量结论。CLIP 与人脸需绑定同一个服务的 runtime/cache 身份；人脸候选镜像用 `FOLIOPATH_INCLUDE_FACE=1` 构建。
省略 face 时不启用人脸；格式/兼容性错误会拒绝启动，运行期服务离线只影响 AI。配置启用模型准入、代次激活及有界分析/清理 worker；每库仍须通过既有鉴权 API 显式启用并提交分析任务。人物管理与正式界面已有 P2/P3 本地功能验收，非空人脸质量尚未校准，测试 profile 不可视为生产阈值。示例探针仅用于临时测试，见 [启动证据](evidence/int-001/face-startup-2026-09-16.md)及 [P3 记录](evidence/int-001/p3-acceptance-2026-09-16.md)。

人脸分析和清理各一个 worker，与其他后台任务共享资源控制；远程请求与 CLIP 共用客户端并发上限。激活时本地槽满会等待而不消耗失败次数，等待可取消；取得槽后身份 HTTP 校验仍受超时限制。分析中本地槽满或身份匹配的服务端 busy 均保留 claim，由既有人脸队列在租约过期后恢复（默认约两分钟，最多三次 claim），不将图片计为失败或把模型标离线。真正远程不可用会停止当前分析，按库进入等待模型状态；服务恢复后可重新启用/提交任务。媒体仍仅从主应用只读 `/library` 读取，AI 服务不挂载媒体或数据库。

2026-09-16 人脸接入开发：migration 40 扩展内部 service 模型记录，现已补充上述可选主进程配置入口。升级前备份 `/app/data`；一旦存在人脸 service 记录，不支持直接降回 39，应恢复升级前备份，禁止为了降级删除模型/代次。当前开发验证只运行本地临时 Docker 测试库，未升级现有运行环境。

候选主进程现支持 `FOLIOPATH_AI_DEPLOYMENT` JSON，字段为 `origin`、`identity`、
`model_bytes`、`license_id`，严格解析并在启动时拒绝不合法配置。身份需从固定候选
镜像离线读取，不能把 HTTP 实时响应当成可信预期；结构见
[启动合同](architecture/independent-ml-generation-contract.md)。未设置则保留旧路径。
本地测试再叠加 `spikes/int001-remote-ml/compose.app.yaml`：主应用仅发布
`127.0.0.1:6091`，AI 仍仅在内部网络，无媒体或数据库挂载。测试主应用现使用包含
FFmpeg 的实际 distroless 镜像 `foliopath-app:candidate`，
挂载合成 fixtures 和独立 candidate-app-data 卷，不是发行 Compose。
AI 初始离线不阻塞普通图库；每 30 秒核验，恢复后复用活动模型，不清理旧索引。
激活每个可用性版本最多三次持久化尝试，无直接网络回退。

2026-09-15：migration 39 引入 service 模型类型。出现该类型后不能直接降级数据库到 38；
迁移会拒绝，不会删除模型以完成降级。测试中显式回滚失败事务验证数据保留；运行时升级
失败应关闭连接/停止启动，回退须恢复升级前 `/app/data` 备份，不得直接更换旧镜像强行打开。

2026-09-15 ADR-0020 专用 AI 候选镜像使用 `services/inference/Dockerfile`；隔离 Compose
覆盖文件为 `spikes/int001-remote-ml/compose.foliopath.yaml`。AI 不发布端口，只读 `/cache`，
容器内 1 GiB `/verified-cache` tmpfs 保存已核验模型快照，另有 128 MiB `/tmp`，
内存预算 4 GiB（包括快照及模型加载）。健康检查只有校验完成后成功。
此预算不是四核/4 GiB 整机容量验收，也不是最终正式 Compose，根部署未切换。

2026-09-14：新 AI 双服务方向见 ADR-0019。当前可运行的隔离验证 Compose 位于
`spikes/int001-remote-ml/compose.yaml`，固定镜像、仅私网、无端口、只读复制模型。
这是 ML 服务 + 测试客户端，不是主应用已接入的正式 Compose；根 compose.yaml 尚未改变。

## 模型下载代理（POST-MVP-5 / ADR-0017，旧兼容路线）

本节仅适用于旧内置模型下载维护能力，不用于独立 AI Compose 的固定缓存恢复；当前双服务主线见 [AI Compose 部署说明](ai-compose.md)。保留既有 API 兼容性，不表示模型商店仍是主线。

支持显式设置 `FOLIOPATH_MODEL_DOWNLOAD_PROXY=http://192.168.2.231:7890`。
在支持 ADR-0017 的镜像中为 FolioPath 服务增加此环境变量并重新创建容器后生效。
默认跟随部署时，未设置/空值为直连；HTTP_PROXY、HTTPS_PROXY、ALL_PROXY、NO_PROXY 不会影响模型下载。
含 ADR-0018 的镜像还支持「智能 → 模型 → 模型下载代理」：跟随 Docker、直连、自定义。
界面选择优先于部署值；选择跟随配置可恢复 Docker 默认。设置保存在 /app/data 的 SQLite，
重启保留。保存无需重启，下一文件连接生效，在途文件不切换。连接测试不会保存表单或安装模型。
更新前备份 /app/data；本变更增加迁移 00037，旧镜像回退应按既有备份恢复流程执行。
此变量不是入站 `FOLIOPATH_TRUSTED_PROXIES`，也不改变媒体/更新检查等其他请求。

仅接受无认证 HTTP/HTTPS 代理 origin，可指定端口；不接受账号密码、路径、query 或 fragment。
代理失败不会自动直连。HTTPS 证书校验不关闭，代理无需也不应解密模型流量。
代理承担目标域名解析与私网/metadata 地址拦截策略，运营者必须信任它。
固定模型目录仍须明确允许 CDN 重定向主机；例如本次固定 SigLIP tokenizer 源跳转到
`us.aws.cdn.hf.co`，不能只允许 huggingface.co，也不能使用通配符允许所有跳转。

## 状态

本文描述首个可用版本的候选部署方式。Stage 5 `S5-001A` 已建立根 `Dockerfile` 候选镜像：
它构建并嵌入真实 Vite SPA。既有 UID/GID `65532:65532` 候选曾在
linux/arm64 本地验证只读容器根、丢弃 capabilities、只读媒体、健康检查、MVP 媒体矩阵
和 SIGTERM。可信代理边界已通过 `S5-003`；[CR-2026-002](changes/CR-2026-002-authenticated-lan-http.md)
进一步允许经认证的 LAN HTTP，根 `compose.yaml` 已通过同架构
smoke。本机 100k/10k 容量档和 Chromium/Firefox/WebKit 候选自动化也已通过。
原生 amd64/arm64 候选运行、真实升级/回滚与目标容量已经通过；真实 Firefox 核心链及
原生 200%/400% 缩放已通过，物理读屏/触控/移动设备、Safari 缩放、最终不可变 digest、
供应链处置和 Release Candidate Gate 尚未完成，当前候选仍不可作为稳定版部署。
候选运行层已更新到固定 digest 的 Debian 13 distroless，Go 构建层固定为 1.26.5；
这不是安全签署。当前 [S5-007 供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)
的本机 arm64 修复候选为 `0 Critical / 0 High`，但仍被最终干净提交的原生双架构复扫、
provenance 和安全/合规签署阻断。
[ADR-0012](adr/0012-root-runtime-bind-data.md) 已将当前候选改为 root runtime，以支持
Docker 自动创建的 root-owned `./data` bind 目录；既有非 root 证据不能替代新候选复验。

## 模型下载目录

以下为旧兼容维护说明；独立 AI 双服务使用部署配置和只读固定缓存，不要求创建本节目录，不会自动转换或下载缺失权重。

[CR-2026-026](changes/CR-2026-026-ai-model-download-and-ui.md) / [ADR-0016](adr/0016-admin-model-download-catalog.md)
增加“智能功能 → 模型”的下载操作。**仓库没有默认托管模型下载源，根 Dockerfile 也没有编入 AI
推理组件**；页面不能替代这两项部署准备。下载完成只代表文件校验安装成功，不代表已启用或质量达标。

部署管理员在现有数据卷中提供 `/app/data/model-downloads.json`（普通文件，最多 1 MiB），重启后载入。
文件不存在或为空时不配置任何来源；有效 JSON 顶层必须是数组，最多 16 项。`[]` 可显式关闭目录。
格式错误、未知字段、重复键或不兼容清单会拒绝启动；变更前备份配置，启动失败可恢复旧配置。此文件
只能由受信任部署管理员写入，不应通过 HTTP 开放编辑，也不需要为媒体增加挂载。

每项结构由 `internal/aimodel.DownloadDefinition` 定义：

| 字段 | 配置要求 |
| --- | --- |
| `id` | 稳定、不透明、8–128 字符，字母数字开头，仅字母数字/下划线/连字符；换包时使用新 ID |
| `name`、`sourceLabel` | 给用户看的模型名和来源名，不含 URL、凭据或宿主路径 |
| `manifest` | 完整 FolioPath 模型包 manifest；语义 v2 或人脸 v3 必须匹配既有 graph/tokenizer/预处理合同；旧 v1 仅保留兼容读取 |
| `contentHash` | 审核包的固定 64 位小写十六进制内容标识；同一个包不能重复出现 |
| `runtimeArchitectures` | 非空 `amd64` / `arm64` 列表，依据该模型实际验证范围填写 |
| `files` | 文件名到固定 HTTPS URL 的映射，与 manifest 文件列表完全对应；每个文件大小、SHA-256 必须精确填写在 manifest 中 |
| `redirectHosts` | 可选 CDN 主机名数组（最多 8 个），不含 scheme/端口/路径；仅允许实际必要的重定向主机 |

下载 URL 只能使用公开地址的 HTTPS/443；不支持 HTTP、私网镜像、URL 凭据、动态模型代码或环境代理。
需要 CDN 重定向时显式配置其主机；不允许依赖会过期的私有签名 URL。FolioPath 不从 Hugging Face
仓库页自动转换模型，也不把任意 ONNX 当作兼容模型包。下载源与完整包应先由部署者审核；本说明不提供
假示例权重地址来冒充可用源。完整包合同见[语义 v2 ADR](adr/0014-siglip-sentencepiece-tokenizer-runtime.md)
和[人脸 v3 ADR](adr/0015-face-model-package-and-generation-activation.md)。

配置后，管理员在界面下载，服务端通过唯一安装 worker 串行流式传输至 `/app/data/models`，复用现有
空间配额、安全余量、逐文件校验和原子发布。离开页面不会取消任务；进度/最近结果保存在 SQLite，
可跨浏览器查看。主动取消会结束在途连接；服务重启后排队项继续排队，运行中被打断的任务失败关闭，
临时文件按既有恢复流程清理。重试从零开始，不支持断点续传。已安装但未启用的模型不会启动分析。
同模型已有活动下载时另一个幂等键返回冲突，避免重复排队。

离线 `/models:ro` 导入保留在模型页折叠区。目录条目同样用于精确包兼容验证；删除/撤回目录条目需
重启，会使依赖该包的激活验证失败关闭，不删除原始媒体或已保存模型。备份需包含下载配置、数据库和
要保留的托管模型；不要把下载配置中的真实服务信息放入公开日志或截图。

## 部署目标

- 核心模式一个容器、一个应用进程、一个 HTTP 端口；可选 AI 模式为主应用 + 独立 AI 两个容器，仅主应用发布 HTTP 端口。
- 用户通常只映射一个只读媒体根目录和一个可写数据目录。
- 新增 `/library` 下的媒体库只需在 Web 设置中操作，不修改 Compose。
- 媒体原件始终只读；数据库、设置、任务状态和缓存全部位于 `/app/data`。
- 目标发布 `linux/amd64` 与 `linux/arm64` 镜像，并用不可变版本标签升级。

## Docker Hub 自动发布

独立 workflow [`.github/workflows/dockerhub.yml`](../.github/workflows/dockerhub.yml)
负责版本准备和 Docker 发布。推送到 `main` 后，Release Please 根据 Conventional Commits
创建或更新 Release PR；workflow 随即 squash merge 该 PR，在同一轮创建
`vMAJOR.MINOR.PATCH` GitHub Release，并构建包含 `linux/amd64` 与 `linux/arm64` 的
Docker Hub image index。发布流程对两种原生架构执行最终镜像 smoke；完整测试与其他
候选 Gate 证据仍由发布者在推送前完成。

POST-MVP-5 另有只允许手动触发、只读权限的
[`Intelligent media native evidence`](../.github/workflows/intelligent-media-native.yml) 验证入口；它不发布镜像、
不修改部署，也不替代 Docker Hub workflow。只有同一 source SHA 的原生 amd64/arm64 实际成功运行才是
候选证据；workflow 文件存在或失败后上传诊断 artifact 均不算通过。
当该文件尚未合入默认分支时，发布者可对已注册的 `dockerhub.yml` 选择待测 branch/ref，并把手动
`image_tag` 精确设为 `intelligent-media-native-evidence`；该 sentinel 在同一 commit 的 workflow 中互斥跳过
Docker 发布，只调用只读 native evidence workflow。任何其他 `image_tag` 仍属于真实发布路径，不得用于
测试。该桥接不改变最终模型、质量、供应链与批准 Gate。
Docker Hub 发布流程使用仓库 `GITHUB_TOKEN` 完成 PR 和 Release，不连接或更新任何实际部署实例。

只修改 Docker Hub 文案时，可提交 `README.dockerhub.md` 后手动运行
[`Sync Docker Hub overview`](../.github/workflows/dockerhub-overview.yml)。它复用现有 Hub 凭据和固定版本的说明同步 action，只更新 Overview 与简介，不构建镜像、不修改标签或实际部署。文案提交可使用 `[skip ci]` 避免触发完整镜像发布，再通过 `gh workflow run dockerhub-overview.yml --ref main` 同步。

在 GitHub 仓库的 **Settings → Secrets and variables → Actions** 中配置：

| 类型 | 名称 | 示例 / 用途 |
| --- | --- | --- |
| Secret | `DOCKERHUB_USERNAME` | Docker Hub 用户名或组织机器人账号 |
| Secret | `DOCKERHUB_TOKEN` | 具备目标仓库镜像推送及 Overview 更新权限的 Docker Hub access token |
| Secret（可选） | `DOCKERHUB_DESCRIPTION_TOKEN` | 仅用于 Overview 更新的独立 token；未配置时复用 `DOCKERHUB_TOKEN` |

触发与标签规则：

- 推送到 `main` 时自动构建并推送 `latest` 与 `sha-*` 标签。
- 普通 `main` 提交按 `feat:`、`fix:` 和 breaking-change 语义决定下一版本。存在可发布变化时，
  workflow 自动创建、squash merge Release PR，并立即创建 `vMAJOR.MINOR.PATCH` 标签和
  GitHub Release。
- 自动发布的同一次 Docker 构建追加 `MAJOR.MINOR.PATCH`、`MAJOR.MINOR`、`latest` 和
  与实际构建 commit 对应的 `sha-*` 标签；纯技术提交只更新 `latest`/`sha-*`。
- 手工推送 `vMAJOR.MINOR.PATCH` Git tag 仍可构建相同的语义化版本标签，但正常发布应由
  Release PR 驱动，避免版本号与 `CHANGELOG.md` 分离。
- 从 Actions 手动运行时必须指定标签，默认是 `edge`；选择 `main` 时仍按现有规则更新
  `latest`，独立分支测试不更新它。
- 原生 `ubuntu-24.04` 与 `ubuntu-24.04-arm` runner 并行构建，每架构只构建一次最终
  镜像；各限时 45 分钟。同一发布队列不自动取消在途构建。
- 每架构先按 digest 推送并拉取同一镜像，验证源码、版本、平台和发布 runtime smoke。
  两份成功记录匹配且 SPDX/SLSA attestation 完整时，才合并 index 并更新最终标签。
  发布阶段不重建镜像，任一架构失败都阻止最终标签更新。
- GHA layer cache 和 Go/npm cache mounts 分架构持久化；缓存可丢弃，首次仍需完整编译。
  缓存恢复/注入/导出/保存各有三分钟时限，失败会跳过缓存；镜像验收不能跳过。
  协议见 [ADR-0026（原生发布）](adr/0026-native-release-image-promotion.md)，结果见
  [维护记录](changes/FIX-2026-10-23-native-release-builds.md)。
- 构建附带 SBOM 和 provenance；发布后同步专用 `README.dockerhub.md`。默认复用
  `DOCKERHUB_TOKEN`，也可配置独立的 description token 覆盖它。

面向用户的更新日志由根 [`CHANGELOG.md`](../CHANGELOG.md) 承载。合入 `main` 的提交标题必须
使用用户能理解的中文 Conventional Commit，例如 `feat: 媒体库内容变化后自动刷新`；
Release Please 将内容归入 `✨ 新功能`、`🚀 改进`、`🐛 修复` 和 `⚠️ 注意事项`。
`docs:`、`test:`、`build:`、`ci:` 与 `chore:` 等纯技术提交不会出现在用户更新日志。
Release PR 作为可审计的版本与日志变更记录保留，但不等待人工合并。提交者必须在推送
`main` 前完成措辞和适用 Gate 检查；自动生成的版本 artifact 本身不替代发布 readiness 证据。

发布完成后验证：

```sh
docker buildx imagetools inspect evanqu/foliopath:VERSION
docker pull --platform linux/amd64 evanqu/foliopath:VERSION
docker pull --platform linux/arm64 evanqu/foliopath:VERSION
```

输出必须同时列出 `linux/amd64` 和 `linux/arm64`。部署仍应使用明确版本或 digest，而不是
依赖可变的 `latest`。自动发布入口不改变当前 Release Candidate No-Go 判断；只有适用 Gate
全部关闭后才能创建稳定 GitHub Release。

## 候选 Compose

权威候选配置是仓库根 [`compose.yaml`](../compose.yaml)。它提供可直接启动的默认值，
不要求 `.env`：默认使用官方 `latest` 镜像、仓库相对的 `./library` 和
`./data`，监听所有 IPv4 接口的 `8080` 端口。缺失的宿主目录由 Docker 自动创建，
随后可直接运行 `docker compose up -d`。

需要把媒体、数据或端口放在其他位置时，可以直接编辑 `compose.yaml`，也可以复制
根 [`.env.example`](../.env.example) 为可选的 `.env`，使用以下覆盖项：

- `FOLIOPATH_IMAGE`：明确版本或 digest；不得使用会静默漂移的 `latest`；
- `FOLIOPATH_LIBRARY_PATH`：宿主机唯一媒体呈现根；
- `FOLIOPATH_DATA_PATH`：宿主机本地可写数据目录；默认 `./data` 可由 Docker 直接创建；
- `FOLIOPATH_BIND_ADDRESS`：宿主端口绑定地址，默认 `0.0.0.0` 供受信局域网访问；
- `FOLIOPATH_PORT`：宿主端口，默认 `8080`；
- `TZ`：容器时区，默认 `UTC`，例如可改为 `Asia/Shanghai`。

镜像自身已固定容器内监听地址 `0.0.0.0:8080`。普通 Compose 部署只需映射端口，
不需要暴露或设置 `FOLIOPATH_LISTEN`。

官方多架构镜像发布在 `evanqu/foliopath`。README 快速开始使用 `latest`；需要可控升级时，
请在 Compose 或可选 `.env` 中改为明确版本或 digest。需要验证当前源码时，也可以本地构建：

```sh
docker build --build-arg VERSION=stage5-local -t foliopath:stage5-local .
```

并把 Compose 中的镜像改为 `foliopath:stage5-local`，或在可选 `.env` 中将
`FOLIOPATH_IMAGE` 设为该值，然后运行
`docker compose up -d`。Compose 默认将宿主端口发布到所有 IPv4 接口，以
root runtime、只读根、全部 capabilities 丢弃、`no-new-privileges` 和受限
`/tmp` tmpfs 启动。镜像内 healthcheck 已由候选 smoke 验证。

本地 tag 只标识当前源码验证，不是不可变发布引用。长期部署建议使用 Docker Hub 上的明确
版本或 digest。

### 为什么只需要两个 volume

```text
宿主机 /mnt/photos       →  容器 /library   只读允许边界
宿主机 ./data           →  容器 /app/data  可写持久数据
```

Web 设置只能从 `/library` 中选择媒体库。例如，`/mnt/photos` 中存在 `family`、
`work/2026` 与 `archive` 普通目录时，可以分别创建 `/library/family`、
`/library/work/2026` 和 `/library/archive` 媒体库；每个库默认递归包含其子目录，
无需重启容器或再写 volume。

`/library` 是唯一媒体挂载目标。它自身可以是一个 bind mount 或 volume，但不得在
`/library/family`、`/library/work` 等后代路径再嵌套 volume、bind mount 或其他
mount point。下面这种拼装方式**不受支持且会被拒绝**：

```yaml
# 错误示例：不要这样配置
volumes:
  - /volume1/family:/library/family:ro
  - /volume2/work:/library/work:ro
  - ./data:/app/data
```

若媒体分布在多个独立宿主机卷，部署者必须先在宿主机侧把它们提供为一个、对容器
表现为单一文件系统且没有后代 mount crossing 的呈现根，然后只把该根挂载一次：

```yaml
volumes:
  - /host/media-presentation:/library:ro
  - ./data:/app/data
```

FolioPath 不选择、配置或承诺具体 union、聚合文件系统或 NAS 技术；部署者需要验证
其一致性、权限和故障语义。若新的宿主目录不在现有呈现根中，必须先调整宿主机根
布局或根挂载，应用不能在 UI 中扩大容器被授予的边界。完整决策见
[ADR-0009](adr/0009-linux-openat2-single-media-root.md)。

### Linux 路径解析运行条件

Linux 版本从已锚定的 `/library` 目录文件描述符打开每个媒体路径，并要求内核
`openat2` 支持
`RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV`。这同时拒绝越界、
符号链接和 `/library` 下的同设备或跨设备 mount crossing。

内核未实现所需调用/flags，或容器 seccomp/LSM 阻断它们时，FolioPath 必须失败
关闭并把媒体根视为不可用；不会降级为较弱的 realpath、device/inode 或 `os.Root`
检查。发布兼容性取决于实际系统调用，不只取决于内核版本字符串。生产容器不需要
`CAP_SYS_ADMIN`；该 capability 只用于隔离 CI 中构造 mount 边界测试。

非 Linux adapter 仅用于开发与补充测试，不构成同等级 mount-boundary 证明，也
不扩大 MVP 的 Linux 平台承诺。

## 文件权限

- 候选镜像以 root 运行，使 Docker 自动创建的 `./data` bind 目录无需 `chown` 即可写入。
- root runtime 扩大容器被攻破后的影响；权威 Compose 仍使用只读根、丢弃 capabilities 和
  `no-new-privileges`，简化部署不得暴露到不可信网络。
- 容器对 `/library` 及所选普通子目录需要读取和目录遍历权限，对 `/app/data` 需要读写、创建、重命名和同步权限。
- 媒体映射必须使用 `:ro`。应用报“只读”不能替代 Docker 层的只读保护。
- 使用默认 `./data` 时可由 Docker 自动创建；无需预创建、`chown` 或全局可写 `0777`。
- SELinux 主机可能需要专用 volume label；准确选项应在对应发行版验证后加入平台说明。

应用需要临时文件时，应使用受控的 `/app/data/tmp` 或 Compose 中受限的 `/tmp` tmpfs。
候选 smoke 已确认 libvips、FFmpeg 和健康检查可在只读容器根下运行；时区/证书的最终
平台矩阵仍归发布 Gate。

## 数据目录布局

当前布局：

```text
/app/data/
├── foliopath.db          SQLite 主文件
├── foliopath.db-wal      运行时 WAL，可能存在
├── foliopath.db-shm      运行时共享内存文件，可能存在
├── cache/                可重建缩略图与视频封面
└── tmp/                  可清理的受控临时文件
```

`foliopath.db`、`cache/` 与 `tmp/` 已由当前 composition root 固定。WAL/SHM 是 SQLite
运行时文件，停止后的安全备份中可能不存在。所有持久状态必须留在 `/app/data`；不得把
数据库放入媒体 volume，也不得把原媒体复制进缓存目录。

SQLite WAL 依赖正确的锁和同步语义。`/app/data` 必须使用宿主机本地文件系统或明确验证兼容的存储；MVP 不支持直接把活动数据库放在 SMB/NFS 上。单一 `/library` 根可以来自经过验证的 NAS 挂载，但不得在其后代再嵌套其他挂载；断连时应用应把受影响的库标为离线并保留索引。

每次进程启动都会按媒体库 ID 分页安排或合并一次 `startup` 完整扫描。启动时仍处于
running 的旧任务由 durable lease 约束；无论 lease 在进程启动前还是启动后才到期，
worker 都会有界恢复并重新领取。取消、离线、权限失败或异常退出留下的最后可靠索引会保留，
只有后来一次完整成功扫描才发布新 generation 并清理陈旧派生行。

## 首次启动

当前候选流程：

1. 容器验证 `/app/data` 可写、数据库可打开且迁移可以安全执行。
2. 服务启动 API 和前端；未完成初始化时只开放受限的单管理员创建流程，不存在默认密码。
3. 管理员登录后进入设置，在 `/library` 的安全目录选择器中创建一个或多个媒体库。
4. 创建成功后异步启动首次完整扫描，页面显示阶段、跳过统计与计数；应用启动时校准，之后默认每 24 小时完整扫描。
5. 媒体库可浏览后继续在后台生成所需缩略图；默认 10 GiB 配额达到水位时按 LRU 清理可重建缓存。

当前候选已包含内建单管理员、会话、CSRF 和退出登录，可在受信局域网通过 HTTP 使用。
Stage 5 尚未通过，因此仍不能把候选作为稳定发行版或匿名服务；公网和不可信网络必须由
部署者在应用外提供 TLS 与访问控制。

## 健康检查与可观察性

当前候选应用提供：

- `GET /health/live`：进程事件循环可响应，不检查媒体库是否在线。
- `GET /health/ready`：数据库已打开、迁移完成且服务可以接受业务请求；一个媒体库离线不应让整个实例不就绪。
- `GET /api/v1/status`：认证后返回版本和安全的能力/初始化状态。

应用启动时会先准备固定 `/app/data`，打开 SQLite 并执行嵌入 migration，成功后才启动
HTTP 并进入 ready。数据目录不可用、数据库打不开或 migration 失败时进程失败关闭，不会带着
部分 schema 提供业务服务。系统状态路由要求有效管理员会话。

健康端点不得泄露路径、数据库错误或版本依赖细节。候选 healthcheck 的命令、间隔和启动
宽限已经通过本机镜像 smoke；最终 digest 仍须在原生双架构重复。

当前可观察面包括 JSON 标准输出/错误日志、live/ready、认证系统状态、扫描运行/历史和
设置中的缓存配额。日志轮转由容器平台负责。当前没有 Prometheus/metrics endpoint，也不
直接暴露队列深度或宿主磁盘余量；部署者必须在容器/宿主平台监控 `/app/data` 容量。
正常日志不记录宿主机路径、认证信息或媒体工具完整 stderr。

生产 final stage 有意不包含 shell、curl、tar 或包管理器。不要进入运行容器安装调试
工具或改变其不可变闭包；诊断、归档或网络探测应从宿主机执行，或使用固定版本的短生命周期
辅助容器，并只授予完成该操作所需的精确 volume、网络和权限。下文备份命令均为宿主机命令。

## 网络、反向代理与认证

- 应用二进制默认监听 `127.0.0.1:8080`，正式容器镜像默认设置为
  `0.0.0.0:8080`。高级部署可通过 `FOLIOPATH_LISTEN` 或
  `--listen=<IP>:<PORT>` 设置，参数优先于环境变量；两者都只接受一个数值 IP 与
  `1～65535` 端口。
- `/library` 与 `/app/data` 是固定容器边界，不提供改写它们的环境变量或参数。媒体库在
  Web/SQLite 中配置，不能转化为每库一个启动变量。
- 非回环监听可直接服务经认证的受信 LAN HTTP。没有可信代理配置时，应用清除所有
  `Forwarded`/`X-Forwarded-*` 声明，并使用直连 peer 作为客户端身份。
- 外部 HTTPS 代理可显式设置 `FOLIOPATH_TRUSTED_PROXIES`。只接受逗号分隔的精确 IP CIDR；
  不要使用 `/0` 或包含普通客户端/NAT 出口的宽泛网段。非回环监听设置该变量后进入
  代理专用模式，远程直连失败关闭。
- MVP 契约与当前候选只支持单跳代理。代理必须覆盖客户端提交的转发头并发送恰好一个
  `X-Forwarded-Proto: https`、公共 `X-Forwarded-Host` 和数值客户端 IP
  `X-Forwarded-For`。应用拒绝链式、多值、缺失、非 HTTPS或与 `Forwarded` 混用的请求。
- 公网或其他不可信网络必须由部署者使用 TLS 反向代理，并通过回环绑定、私有容器网络或
  firewall 防止绕过代理。TLS/代理不是 FolioPath 的必需部署单元；可信 CIDR 也不是网络 ACL。
- 反向代理需要允许图片和视频响应流式传输、Range、较长读取和客户端取消；不应缓冲整个视频到内存。
- 验证后的代理 HTTPS transport 会启用 Secure Cookie 与 HSTS；状态修改继续要求
  session-bound CSRF，setup/login 要求公共 HTTPS origin 与 host/port 完整同源。

认证产品决策和威胁要求见[安全模型](security.md)与[需求确认清单](requirements-checklist.md)。

## 备份与恢复

### 备份范围

必须备份整个 SQLite family；它包含管理员凭据、媒体库配置、应用设置、扫描历史、索引和
任务状态。停止应用后通常只剩 `foliopath.db`，但运维流程不应依赖这一假设。

`cache/` 与 `tmp/` 可以不备份：其中只有可重建缩略图、视频 poster 和临时文件。保留
`cache/` 可以减少大型媒体库恢复后的重建成本，但不是数据完整性的必要条件。

原媒体不属于 FolioPath 数据备份；用户仍需独立备份 `/mnt/photos`。

### MVP 安全备份流程

在没有内建在线备份命令前，推荐：

1. 在 `compose.yaml` 所在目录运行 `docker compose stop foliopath`。
2. 用 `docker compose ps --status running --quiet foliopath` 确认没有运行中的应用容器。
3. 从 Compose 或可选 `.env` 确认数据目录的精确宿主路径，归档该目录的全部内容；最安全的
   默认是包含 `foliopath.db`、可能存在的 `-wal`/`-shm`、`cache/` 和 `tmp/`。
4. 把归档复制到不与活动数据共盘的受保护位置，记录当前镜像版本/digest，并验证归档可读。
5. 运行 `docker compose start foliopath`，等待 `docker compose ps` 显示 healthy。

示例中的路径必须替换为经过人工确认的精确数据目录，不能把未展开变量、空值、工作区根或
媒体目录作为归档目标：

```sh
docker compose stop foliopath
docker compose ps --status running --quiet foliopath
tar --numeric-owner -C /srv/foliopath-data -cpf /srv/backups/foliopath-data.tar .
docker compose start foliopath
```

不要在运行时只复制 `foliopath.db` 而忽略 `-wal`/`-shm`。未来如提供在线备份，必须使用 SQLite backup API 或受控 checkpoint，并通过恢复演练证明。

Stage 5 候选 smoke 已在应用停止后归档完整 SQLite family，并有意省略可重建的
`cache/` 与 `tmp/`；恢复到新空目录后，管理员初始化状态保留且这两个目录会重新创建。
这证明离线最小恢复路径，不授权在应用运行时复制数据库文件。

### 恢复演练

1. 停止容器并保留当前数据目录，不直接覆盖唯一副本。
2. 把备份恢复到一个新的空数据目录并校正所有者。
3. 使用与备份 schema 兼容的镜像启动。
4. 验证迁移、媒体库配置、扫描历史和抽样媒体浏览。
5. 若宿主机媒体位置改变，先恢复相同的容器内 `/library` 布局；数据库不应依赖宿主机绝对路径。

恢复目标必须允许容器 root runtime 写入。rootless Docker 或启用 user namespace
remapping 时，宿主机实际 ID 可能不同，应以该运行时的映射为准，不能机械执行全局
`chown -R`。先在新目录演练并验证成功，再切换 Compose 数据目录或可选 `.env` 的
`FOLIOPATH_DATA_PATH`；不要解包
覆盖唯一活动副本。

候选镜像的自动恢复演练已经建立并在本机 linux/arm64 通过；原生 linux/arm64 与
linux/amd64 还分别以两个不同的不可变候选 image ID 通过向前升级和“旧镜像＋升级前
离线备份”配对回滚。最终不可变 digest 与供应链签署仍是发布阻断。

## 升级与回滚

### 前后端产物一致性

Go 二进制嵌入 Vite dist；只替换二进制也会替换前端。即便只是后端修复，也必须从同一源码快照
重新生成 dist 后再编译 Go，不能复用未经校验的本地旧 dist。每次部署使用独立镜像版本，记录实际
image ID、源码状态与前端哈希，不能把旧发布标签重新指向不同内容。

切换前对隔离候选执行、切换后对实际服务再次执行：

```sh
node --test scripts/verify-deployed-web.test.mjs
node scripts/verify-deployed-web.mjs http://127.0.0.1:8080 internal/webassets/dist /settings/ai /intelligence
```

地址需替换为实际候选或部署 origin。该命令不登录、不修改服务器，逐文件比较 HTTP 返回字节与
本次新构建目录，并验证 SPA 路由返回同一 index；不证明认证后的功能或真实推理通过。应配合独立
合成环境中的认证/API/页面检查。普通 runtime readiness 不能代替这些检查。

推荐升级流程：

1. 阅读版本说明，确认格式、迁移和平台支持变化。
2. 停止服务并完成可恢复备份。
3. 拉取明确版本标签或 digest。
4. 以相同 `/library` 与 `/app/data` 映射启动新版本。
5. 检查迁移、readiness、媒体库离线状态和抽样浏览，再清理旧镜像。

升级到包含 2026-08-12
[媒体处理韧性修复](changes/FIX-2026-08-12-media-processing-resilience.md)的镜像后，在确认
readiness 正常后对受影响媒体库执行一次“补齐缺失”。它只以既有有界 admission 重排缺失、
缓存丢失和失败的派生项；无需重新扫描文件树或执行“全部重建”。本次修复没有 API/schema/
migration/transform bump，已有 ready 资源保留；真实损坏文件重新处理后仍会失败。

若镜像同时包含 [JPEG 有界容错与 MPEG-TS 派生兼容](changes/FIX-2026-08-12-tolerant-jpeg-mpegts-derivation.md)，
同一次“补齐缺失”会让符合窄 allowlist 的≤100 MP 真实 JPEG 以容错产物恢复
ready/succeeded，并让已索引但实际为 MPEG-TS 的现有视频候选重试服务端派生。
这不改变原文件、schema 或 transform version，不新增 `.ts` 支持；其他 0 字节、伪装、超限
或无法容错的媒体仍失败。JPEG 成功容错的 warning 当前只保留在有界 attempt 审计中，
不是 API/UI 可见的 degraded 状态。

数据库只做向前迁移。升级后直接运行旧镜像可能不安全，因此“回滚”必须同时恢复升级前数据备份，除非版本说明明确保证 schema 向后兼容。迁移失败时服务应停止进入业务就绪，而不是带着部分 schema 继续运行。

当前仓库尚无已发布的稳定 digest，因此本阶段使用两个不同的不可变候选 image ID 验证
升级/回滚流程，并已通过 `S5-004B`。首个稳定版本发布后，每次新增 migration 仍必须以
当时真实前一稳定 digest 和升级前离线备份复跑同一演练。

## 媒体格式与播放边界

| 类型 | 扩展名 | 服务端承诺 | 浏览器行为 |
| --- | --- | --- | --- |
| 图片 | `.jpg`、`.jpeg`、`.png`、`.webp` | 索引、元数据、WebP 缩略图 | 原内容查看 |
| 动图 | `.gif` | 索引、元数据、静态缩略图 | 原内容动画 |
| 视频 | `.mp4`、`.mov`、`.mkv`、`.avi` | 索引、ffprobe 元数据、FFmpeg poster、原文件 Range | 仅当前浏览器原生兼容 codec 可直接播放 |

扩展名匹配不区分大小写。MVP 不转码，也不生成兼容播放副本；支持视频容器不等于支持其中
任意 codec。SVG、HEIC/HEIF、AVIF 与 RAW 不进入 MVP 索引/缩略图契约。

图片扩展名只决定候选分类，处理器还会核验真实格式。非 JPEG 总源像素仍上限 100 MP；
只有真实 JPEG 才有独立 180 MP 上限，并在超过 100 MP 后使用 shrink-on-load，≤100 MP
继续原变换。生产 FFmpeg 使用 external libdav1d 处理 AV1 poster/storyboard 等派生资源；
这不代表浏览器能直接播放 AV1，也不新增兼容播放副本。
真实、≤100 MP 的 JPEG 若在严格路径命中窄截断 allowlist，可单次容错生成经验证的
WebP；原件不会被修复或改写。最小 FFmpeg 包额外包含 mpegts demuxer，但它只处理已被
现有扩展名合同收录、实际为 MPEG-TS 的候选派生；`.ts/.mts/.m2ts` 仍不在支持列表。
这类候选的 `playback_status` 保持 `unknown`，服务端派生成功不是浏览器可直放的承诺。

`POST-MVP-1` 的视频故事板候选复用同一 FFmpeg runtime，为成功探测且至少 2 秒的视频生成
4 或 10 帧 WebP sprite。它不新增端口、环境变量、volume、服务或媒体写权限，仍只写入
`/app/data` 的可重建缓存。当前生产镜像纵向链已通过 VSP-301，但原生 linux/amd64 与
linux/arm64 同提交候选证据仍由
[VSP-302](gates/POST-MVP-1/vsp-302-target-platform.md)阻断，因此不能把该能力描述为稳定发布。

## 当前候选已知限制

- 官方镜像位于 `evanqu/foliopath`；生产部署建议固定明确版本或 digest。
- 发布平台目标是 Linux amd64/arm64。非 Linux adapter 只用于开发；本轮候选已由本机
  原生 arm64 与操作者指定的原生 amd64 服务器完成运行矩阵，最终不可变 digest 尚未签署。
- `/library` 只能有一个顶层媒体挂载，后代不能是 mount point；目录 symlink 不跟随。
- `/app/data` 只支持单实例独占的可靠本地文件系统，不支持把活动 SQLite 放在 SMB/NFS，
  也不支持多实例共享写入。
- 离线备份/恢复及不同不可变候选间的升级/配对回滚已自动化并通过；在线备份不在当前
  自动化承诺内，未来新增 migration 仍须对真实前一稳定 digest 复跑。
- 100k 媒体/10k 目录候选档已在本机 linux/arm64 与指定原生 linux/amd64 服务器通过；
  100k 全量媒体、cache 90%→80% 水位和 Chromium/Firefox/WebKit 的 100k FPS/RSS
  预算也已通过，`S5-005` 已关闭。
- Chromium、Firefox、WebKit 自动化候选矩阵已建立，但 WebKit 不等同于 Safari 真机；
  最终浏览器版本、读屏和物理设备仍待 `S5-006B`。
- 当前本机 arm64 修复候选扫描为 `0 Critical / 0 High`；最终原生双架构 digest 的
  `all` 策略复扫、provenance 与安全/合规签署仍是发布阻断。
- 当前没有 Prometheus/metrics endpoint、内建日志轮转或宿主磁盘余量采集；这些由部署平台监控。
- MVP 只有一个管理员，不支持匿名 LAN、多用户角色、分享链接、上传或原媒体整理。
- 正确性依赖启动、手动和计划完整扫描，不依赖 filesystem watcher；默认计划间隔为 24 小时。
- 查看器不提供完整 EXIF、显式下载按钮或移动端滑动切换。

## 故障语义

| 故障 | 期望行为 |
| --- | --- |
| 单一 `/library` 根断开或变得不可读 | 标记受影响媒体库离线，保留索引，不执行陈旧清理 |
| `/app/data` 不可写 | readiness 失败，拒绝需要写入的操作，记录安全错误 |
| 磁盘不足 | 停止派生任务并保护数据库；不能删除原媒体或静默损坏索引 |
| 扫描中容器退出 | 下次启动标记任务中断，新完整扫描负责收敛 |
| 缩略图缓存损坏 | 标记派生数据待重建，保留媒体索引 |
| 媒体派生旧失败 | 部署修复后执行一次“补齐缺失”有界重排；不重新扫描、不预先删除 ready 缓存 |
| 0 字节、像素超限或真实损坏 | 显示对应稳定原因；永久失败不自动循环重试，原媒体保持只读 |
| 数据库迁移失败 | 不进入业务就绪，提示从备份恢复或修复环境 |

## 发布前部署门槛

- amd64/arm64 镜像通过相同集成与媒体 fixture 测试。
- root runtime、`:ro` 媒体、capabilities 丢弃和只读根文件系统组合经过验证。
- Compose 中的镜像、端口、运行身份、healthcheck 和环境变量全部为真实值。
- 完成数据库备份/恢复、升级失败和磁盘已满演练。
- SBOM、第三方许可证和镜像漏洞扫描纳入发布流程；候选自动化已经建立，但只有最终
  双架构 digest 的全阻断扫描、notices/provenance 与签署完成后才满足此项。
- 明确支持的宿主机、CPU/内存建议和目标库规模，不用未验证的数字宣传。
- 以约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器完成主要容量验收；具体吞吐与延迟只能引用可重复 benchmark。
- 单管理员初始化、登录、会话、退出和 CSRF 测试通过；无认证开发构建不会被文档引导直接暴露到互联网。

## POST-MVP-5 revision 1 模型部署合同

该 revision 保持单容器、单 Go 进程，不增加 worker、GPU、Redis、数据库或网络端口。后端和消费者
已实现，但最终审核模型、双架构、联合容量、供应链与批准尚未通过 S4，因此发行构建仍失败关闭；
以下 Compose 片段是 S4 必须以最终镜像验证的目标配置，不应加入当前 Quick Start：

```yaml
services:
  foliopath:
    volumes:
      - /mnt/photos:/library:ro
      - ./data:/app/data
      - /srv/foliopath-models:/models:ro
```

- `/models` 可省略；省略或为空时应用核心 readiness 保持正常，智能能力显示 model unavailable。
- `/models` 必须是单个只读挂载，其后代不得再含 mount。应用不创建、修改或删除其中任何内容。
- `/app/data/models` 保存 managed package；`/app/data/ai-indexes` 如后续存在，只能保存可重建索引。
  SQLite、staging 与 managed final 必须处于支持原子 rename/fsync 的本地文件系统。
- revision 1 不增加模型源、镜像、代理、凭据或下载环境变量，也不依赖外网。部署者负责通过其可用
  渠道取得与产品内建兼容清单完全匹配的包。
- 发布镜像必须以显式 `onnxruntime` Go build tag 构建，包含精确 ORT 1.28.0 runtime、其
  `libonnxruntime.so.1` SONAME 链接、MIT license、third-party notices 和补充 SBOM component。未带该
  tag 或 runtime 版本不匹配时，核心浏览仍可启动，但模型激活稳定失败为 runtime unavailable；不得
  自动回退到另一 ORT 版本。
- 镜像体积预算不包含模型权重，因为权重不内置。发布说明必须分别列出 runtime 增量、每个审核模型
  包大小、managed copy 临时峰值、embedding/index 目标档以及数据库/临时写安全余量。
- 默认 AI worker 并发为 1，受全局后台 admission 约束；4 CPU/4 GiB 档要求整进程 RSS ≤ 3.2 GiB，
  ordinary browse P95 退化 ≤ 20%。这些是发布门槛，不是当前已经证明的产品承诺。
- 备份优先保存 SQLite。若不备份 `/app/data/models` 和 AI 可重建数据，恢复后必须重新提供 `/models`
  并重建；direct `/models` 从不进入 FolioPath 备份。升级不得在新 generation 可用前删除旧包/索引。
- linux/amd64 与 linux/arm64 必须对同一最终 digest 验证 runtime load、数值容差、取消、盘满、强杀恢复、
  只读来源失效和 100k 完整进程。缺一架构即不发布智能能力。

若 ADR-0014 后续接受，发布构建不得把实时访问 GitHub 作为 SentencePiece/ORT 编译的必要条件。受控
获取步骤先固定来源、hash、许可证和签名材料，编译阶段只消费本地或内部镜像提供的 reviewed archive，
并在解包前重新校验 SHA-256。该规则只约束发布供应链，不增加产品运行时网络、模型下载源或代理配置。

## POST-MVP-5 revision 2 C+D+E 部署与恢复合同

Revision 2 不增加 deployable service、网络模型源或媒体 mount。C/D/E 继续在同一 Go 进程、同一 SQLite
和 `/app/data` 下的可重建 cache 中运行；`/library:ro` 与可选 `/models:ro` 拓扑不变。face 与 semantic
native session 延迟加载、空闲卸载、全局后台并发默认 1，interactive browse 始终优先。

- C 的 vocabulary/review application state 与人工 tag 一并进入 SQLite 备份；pending suggestion 可重建。
- D 不复制原视频或抽第二套帧；只消费已存在的完整 storyboard cache。storyboard 被逐出时 video semantic
  标为 stale/degraded，保留旧可靠 generation，等待 owner 重建。
- E 的 person/manual constraint/audit 必须随 SQLite 备份；observation/embedding/crop/anonymous cluster 默认
  不要求备份。恢复时先恢复应用状态，再以 exact source fingerprint + quantized anchor 重连；歧义进入
  `needs_review`，不得按 bbox 顺序猜测。备份和恢复报告不得输出 biometric payload。
- 管理员关闭 E 时停止新任务并卸载 session，但不隐式删除人工人物状态。派生清除和人工关系清除分别
  演练；后者是不可恢复操作，必须明确备份点与删除审计。
- 最终同一版本的 linux/amd64 与 linux/arm64 镜像必须分别在 native 主机验证 C/D/E runtime load、数值
  容差、取消、强杀、ENOSPC、offline、backup/restore 与清除。QEMU、交叉编译或单架构证据不能代替。
- 4 CPU/4 GiB、100k media/10k directories 的联合档同时包含普通 browse、图片 semantic、tag generation、
  完整 storyboard video semantic 和 face job；整进程 RSS ≤3.2 GiB，browse P95 退化 ≤20%。超过门槛先
  降后台调度/关闭对应 slice，不扩大内存承诺或破坏核心浏览。

发布物必须为最终双架构 digest 产出 SPDX/CycloneDX SBOM、VEX、third-party notices、可验证 provenance
和 model/runtime 再分发签署。缺少 face 模型许可、隐私 intake、合法质量数据或任一 native 架构证据时，
S2C 已 Backend Ready，但 Release 保持 No-Go；核心浏览和已经可安全发布的较小 slice 不被伪装成全范围通过。

### 智能能力安装、升级与故障排查

- 模型来源：产品只接受内建 reviewed catalog 中的精确 package ID、版本、文件大小、SHA-256、许可、
  架构和 runtime 合同。公开仓库、文件名、下载成功或“同系列模型”都不能替代精确匹配；当前没有获准
  发行的最终模型包，也没有产品内下载、国内镜像、URL、代理或凭据设置。
- 离线安装：在宿主机准备独立模型目录，以 `/models:ro` 单次挂载启动；该目录不能位于媒体根下，
  后代不能再有 mount。管理员只能扫描服务端批准目录并选择 opaque candidate；应用不写该来源。
  维护者可用 `make test-intelligent-media-offline` 复验无外网、空 catalog、只读来源和重启边界；该本地
  候选检查不能替代最终双架构发行证据。
- 升级：先停止服务并备份完整 SQLite family 和人物、人工 assignment/exclusion/cannot-link/audit 等
  应用状态，再替换镜像。新 generation 完整验证并可用前保留旧 generation。回滚必须使用“旧镜像＋
  升级前备份”配对，不能让旧镜像直接打开已前向迁移的数据。
- 恢复：人物与人工关系随 SQLite 恢复；catalog FTS 缺失或不一致时在数据库打开阶段自动修复；
  semantic/tag/video/face embedding、匿名 cluster 和派生 cache 可缺省，并通过有界“补齐缺失”任务重建。
  重连有歧义时进入 `needs_review`，不得按路径、bbox 顺序或近似姓名猜测。
- 限制：目标验收档为 4 CPU、4 GiB、约 100,000 媒体和 10,000 目录；整进程 RSS ≤3.2 GiB、普通
  browse P95 退化 ≤20% 是未通过最终双架构联合验证前的 Gate，不是当前性能承诺。后台智能任务默认
  全局并发 1；禁用能力必须停止 admission 并卸载对应 session。
- 故障排查：`model_unavailable` 先检查 `/models` 是否存在、只读、无 symlink/hardlink/后代 mount，
  再核对 catalog 中的精确 digest、大小、架构和 runtime；`model_source_changed` 要求恢复原精确来源或
  重新走审核安装，不能自动换模型；`model_corrupt`、`runtime_unavailable`、磁盘安全余量不足或数据库
  损坏都必须保持智能能力或整个 readiness 失败关闭。日志和诊断只能记录 opaque ID、稳定错误码、计数
  与耗时，不得包含查询、姓名、向量、crop、bbox、媒体/模型路径或 native 原始错误。
- 清除：派生数据清除可重建；人物和人工关系清除不可恢复，必须先建立备份点并做精确二次确认。两种
  清除都不得修改 `/library` 原件。模型来源、隐私、许可证或发布证据缺失时，不要绕过 reviewed catalog
  或启用未签署能力；普通文件夹浏览应继续可用。


## Docker Desktop 数据卷修复（ADR-0038）

Docker Desktop上的真实`/app/data`使用经停写复制验证的外部命名卷，避免WAL共享内存经过宿主文件共享层。使用`compose.ai.data-volume.yaml`作为第五覆盖，设置`FOLIOPATH_DATA_VOLUME`为已准备卷；其他四覆盖及`/library:ro`保留。原生Linux本地bind方案不因本轮改变。

切换前停止主应用并确认正常exit0，对完整data及私有env/所有Compose独立备份、逐文件核验；只从停写源向空Linux卷复制，验证清单、原能力/旧OCR摘要及完整性后再起主。旧目录留作历史副本，不代表在线数据；不能启动旧目录或在运行中复制DB。此后停写备份需从卷导出完整data并核验，恢复到独立空卷后检查，不能把更新后的卷与旧目录双向同步。删除卷或重置Docker前必须已有独立完整备份，回退必须先停主且保留切换后新写入。详见[ADR-0038](adr/0038-docker-desktop-linux-sqlite-volume.md)。

## 自动准备 AI（1.6.1起）

CR-2026-053 / ADR-0039 新增 `compose.ai.simple.yaml`，配对镜像自动安装模型，详情见 [简化自动安装入口](ai-compose.md#简化自动安装入口cr-2026-053已发布)。旧离线部署保持原配置。1.6.1主/AI镜像已通过双原生验证并公开推广；不能用于1.5.0旧主镜像。
