# 6010 人物分析补齐与剩余错误排查

状态：四库任务均已结束，8,493 项全部尝试，8,452 成功、41 张源输入失败；不是全部图片通过。
10:12 已在停写完整备份后单次更新 6010 主应用为 c9f28d7a 内存修复候选，保留 AI/模型身份及不限内存配置。
范围：FR-INT-009/016 / POST-MVP-5，Owner face / SQLite / API / web AI。

## 进度总表（2026-10-23 10:16 +08:00，四库终态与更新后正式页面核对）

| 媒体库 | 本轮提交 | 当前实际进度 | 状态 |
| --- | ---: | --- | --- |
| 8k / lib_3 | 792 | 已处理 792：成功 785、失败 7 | 04:33 任务结束；6 张空原图、1 张非空无效输入，并非全图通过 |
| 4k / lib_2 | 500 | 本轮成功 499、失败 1；加原同代次 207，共成功 706/707 | 04:11 任务结束；剩 1 张无效输入，并非全图通过 |
| pixiv / lib_4 | 999 | 本轮成功 998、失败 1 | 05:14 任务结束；剩 1 张 JPEG 码流格式错误，并非全图通过 |
| coser / lib_1 | 6,202 | 已处理 6,202：成功 6,170、失败 32（均 invalid_media） | 10:03 任务结束，无任务级错误；最后新增 3 张亦已逐张诊断为 JPEG 码流异常，未恢复原文件 |
| 合计 | 8,493 | 本轮处理 8,493（成功 8,452、失败 41） | 无待处理项、四个 operation 均 succeeded；源错误不伪装成功，质量/容量/发行门禁另计 |

本轮新增：[剩余 27 张逐图诊断](FIX-2026-10-23-face-source-diagnosis.md)与[大图内存候选/测试](FIX-2026-10-23-main-face-memory.md)。
25 张 JPEG 重启标记前异常字节、1 张 PNG 用了 JPG 扩展名、1 张 JPEG 截断；不是 27 张都由模型/内存造成。
临时原媒体只读手工诊断与纯合成自动化回归严格分开；原件和人工整理未改，失败数仍按真实状态展示。
最终候选 c9f28d7a 已过完整大图扫描/缩略图/推理联验，无自动重启；1 GiB 压力仍触顶，容量/长期风险保留。
原 6010 已在四库结束后更新为 c9f28d7a；此前等待在途任务结束的决定保留在下方时间线。
本次显式部署不算自动重启；新主容器与未更换的 AI 均 healthy、restart=0，容器内 database-check passed。

07:44 本地实例按用户明确要求改用 `compose.ai.yaml` + `compose.ai.unlimited-memory.yaml`，
两服务 HostConfig.Memory=0/MemorySwap=0；Docker Desktop 约 7.75 GiB 总量仍生效。
镜像/模型、CPU/并发、输入大小与安全边界不变；不是默认预算验收通过。
完整停写备份和验证见[当前运行选择](FIX-2026-10-23-main-face-memory.md#当前运行选择2026-10-23-0744-0800)。

**06:31:21 主应用发生已确认的 cgroup OOM，exit 137、自动重启 1 次；不是 SQLite 短读，AI 容器未重启。**
当前 healthy/database-check passed 仅证明已恢复，不能盖过这次失败。
独立合成资源排查见[主应用人脸内存记录](FIX-2026-10-23-main-face-memory.md)；当时未增大额度，
07:44 用户已另行选择取消容器限额。均未重复提交任务或修改原图。
07:22 已在纯合成大图/常驻堆组合中复现相同生产解码路径 exit 137、OOM=true；不是原库具体资产归因。
回收候选仍触及完整 1 GiB 上限，未验收、未部署；详情见该内存记录。

上述为时间点快照，不是最终结果。人物任务只有有界 worker 实际处理；重启后旧租约未到期可暂显示 running，
过期后恢复排队。03:52 时本轮处理 328（成功 327、失败 1），8k 断点 280 保留；
04:19 时处理 967（成功 963、失败 4）；04:11 4k 结束后 8k 从断点继续并于 04:33 结束，
随后 pixiv 接续，于 05:14 结束并接续 coser。04:48 时已处理 1,661（成功 1,653、失败 8）。
05:19 时已处理 2,364（成功 2,355、失败 9）。未重复提交同一库或清除结果。
已获用户同意并创建本聊天 heartbeat `6010`，每 30 分钟核对新错误/终态、更新本文；状态未变化保持安静。

## 已确认的新失败与边界

- 8k 当前记录 `invalid_media`：`ha-long-bay-7680x4320-vietnam-mountains-sea-8k-16348.jpg`（ast_7038）。
  新 API 样本 sizeBytes=0，6010 正式界面已显示“源文件为空（0 B）”；不是模型超时或 worker 中断。
  空文件没有可解码内容，程序无法补造，须用户在外部恢复有效原图后扫描/补齐；未改动原件、伪造成功或无限重试。
- 04:19 新确认 8k 两条 `invalid_media`：ast_7387 `seat-tarraco-fr-phev-7680x4320-electric-cars-suv-2020-cars-8k-22022.jpeg`
  在新详情明确为 0 B；ast_7374 `red-dead-redemption-2-7680x4320-poster-4k-8k-18205.png` 为非空但无法解析的输入，
  独立停写快照大小 37,340 B、无可用尺寸；不能仅凭该码进一步断言具体损坏字节或格式误命名。
- 4k 本轮唯一 `invalid_media`：ast_6669 `0K6029149_0.png`；独立停写快照大小 246,125 B，无可用尺寸。
  正式查看器亦显示“无法读取媒体内容 / 媒体探测失败，文件可能已损坏或内容与扩展名不一致”，缩略图也失败。
  已核对不是任务级错误、模型超时或 worker 重启；尚不能进一步区分损坏与扩展名不符，不自动改原图或再次盲目重试。
- 04:48 新诊断已确认余下 4 张 8k 输入均为 `invalid_media`、sizeBytes=0：
  ast_7449 `subaru-crosstrek-7680x4320-suv-2021-cars-8k-22565.jpeg`（04:21），
  ast_7510 `sage-femme-7680x4320-catherine-deneuve-catherine-frot-8k-13951.jpg`（04:24），
  ast_7574 `abstract-7680x4320-3d-colorful-8k-21244.jpg`、
  ast_7576 `first-man-7680x4320-ryan-gosling-8k-20430.jpg`（均 04:26）。
  8k 的 7 条现有失败现均有本轮逐图记录：6 个空文件与 1 个非空无效 PNG，不再仅依赖旧失败猜测。
- pixiv 唯一 `invalid_media`：ast_8705 `Alcxome_Forest radio_p1_89806011_35562628_2631x1488.jpg`（05:13）。
  正式查看器原件可被 Chromium 容错显示（naturalWidth=3000、naturalHeight=1501），不能因此推断码流有效。
  从已认证正式 content 端点获取只读诊断副本后，标准 JPEG 头解析成功（3000×1501），完整像素解码明确返回
  `invalid JPEG format: missing 0xff00 sequence`。这是有具体码流格式错误的输入，不把它误判成模型/内存故障，
  也不为容错展示而放宽解析安全边界或改写原图。源副本为 4,389,774 B，与既有停写目录元数据相符。
  用户需恢复有效原件后再扫描/补齐；未盲目重试。此为用户授权故障诊断，不是以真实媒体作自动化回归。
- coser 05:51：27 个 `invalid_media`，没有任务级错误；接口最多列出 20 个样本，不能把它当完整失败清单。
  两个代表样本 ast_231 `1 (58).jpg` 与 ast_552 `42_2023_12_18_222003.jpg`（分属两组失败文件）
  均能被 Chromium 和标准 Go JPEG 容错解码，尺寸为 1179×2556；不能据此直接断言程序误报。
  使用现部署同版 libvips/生产 DecodeFaceImage 对正式 content 端点取回的只读副本做本地手工诊断，
  两者均在 native 输出像素时拒绝；具体警告为 `Corrupt JPEG data: 263 extraneous bytes before marker 0xd0`。
  已定位这两个源文件 JPEG 重启标记前存在异常数据，而非模型推理超时/OOM。没有关闭 FailOnError 或写回原件强行通过。
  尚不能把同一底层字节原因套到其余 25 条，后续继续按新诊断核对；其余图片仍在处理。
- 质量校准、完整回归/容量/发行门禁与其他 AI 能力的错误详情不在本次通过范围。

## 已确认的范围与问题

- 2026-10-23 03:37 起，经正式界面依次提交 8k、4k、pixiv、coser 的 missing 任务，未选择全量重建、清除或改动原件。
- 4k 同代次保留 207 条成功，提交 500；其余三库原设置绑定 retired generation，补齐按现部署切到 active generation：
  8k=792、pixiv=999、coser=6202。共 8493 条需要处理，不能把旧代次完成数冒充当前模型已完成。
- 上述基线通过已停写的独立 verified 备份只读核对，不从 macOS 打开活跃 Docker 数据库。
  该备份中 8k 的 7 条旧失败有 6 个零字节文件，另 1 个 PNG 未有可用尺寸；须用本次新失败记录确认，不能补造旧原因。
- 运行中的任务受现有 worker/推理内存限制，会持续较久；排队/运行不等于完成。未把旧计数清零当作修复通过。

## 维护切片

1. faces GET 增加可选 generationCurrent：同一个只读快照比较库绑定代次与全局 active 代次，GET 不修改设置。
   UI/稳定 Story 标明“旧模型结果”，提示补齐将按现模型处理，不静默重标旧向量。
2. 失败样本增加可选 sizeBytes（已有目录元数据，无新持久化）；0 字节且 invalid_media 时直接提示“源文件为空”。
   损坏原图需操作者恢复后扫描/重试；应用不得自动修改原始文件、跳过后伪装成功或无限重试。
3. 后端/OpenAPI/生成客户端先落地并回归，再同步生产界面/Story/中英文与进度文档；不新增模型、部署单元或重试状态机。

## 本轮已执行验证

- PASS：`make fmt arch-check contract-check generate-check`、`git diff --check`；OpenAPI 为可选增量，已同步生成客户端/摘要。
- PASS：`go test ./internal/face ./internal/store/sqlite ./internal/api ./tests/contract`，同包 `go vet`；
  三项 SQLite 新诊断/代次用例 race，`TestFaceAnalysisRecoversCheckpointAfterProcessKill` 集成专项。
- PASS：补充 `go test ./internal/... ./cmd/...` 全部生产 Go 包。
- PASS：前端类型、架构、65 文件/411 项测试、生产组件 Storybook 构建；旧模型/未知代次、0 与缺省大小区别均有断言。
- PASS：本地固定主镜像 `sha256:cd1435af697066c1b8a960aad3922c71e1be458c9ad58dc3cf67fc030692f4b7` 的
  `fp-face-diagnostic-2a8826bc`：真实推理与合成图、空/损坏输入大小和原因、缺失重试只处理 2 张失败图，
  幂等重放、短暂 scanning 控制分支、重启后诊断保留、库隔离/隐私、原件哈希不变、无 OOM。
  临时容器已清理，合成数据保留于 `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-face-diagnostic-2a8826bc-VgfNst`。
- PASS：同固定镜像 scan-worker 补充 smoke `fp-scan-worker-b6dbb6ed`，60 秒、258 扫描、11,020 读取请求、
  2,181 客户端取消、108 容器内检查、自动重启 0；显式重启恢复、缺失/损坏预期负例及原件 hash 通过。
  两行 database check failed 为该脚本预期负例，非原 6010 失败。临时容器已清理，合成数据留在临时目录。
- PASS：6010 原位展开诊断在 390/768/1440/1920 宽度可读且未横向溢出（文档宽 375/753/1425/1905，
  留有浏览器滚动条），均保留长文件名/恢复提示；[四档截图目录](../evidence/ui-audits/2026-10-23-face-retry/)。
  视口已复原；仅本轮中文浅色局部复验，不冒称全部主题/语言矩阵。
- 本轮没有把专项通过冒称完整 `make lint/test/test-integration/test-e2e` 或完整浏览器矩阵；既有 native/临时盘限制仍见 scan-worker 记录。

## 6010 更新

- 仅主应用更新为上述 cd1435af 镜像；AI 镜像保持 fa7750b5、原容器不重启，无新迁移，schema 43。
- 停主应用后完整复制 1.1 GiB data，复制命令结束后才恢复：
  `/Volumes/extended/foliopath/backup-6010-face-retry-20261023-KCsyE8`，含 previous.env/current.env 和 data。
  只对该独立停写副本用 immutable 只读检查：quick_check=ok，foreign_key_check 无行，4 库/8,719 媒体保留。
  主库未从 macOS 打开，更新后容器内 database-check=passed；双服务 healthy、restart=0、OOM=false。
- 停写快照保留四个 operation：
  8k `aio_c5695e57b36d31ab40b2af54d3364071`，4k `aio_2441af32c4b499bf0f029510038555b9`，
  pixiv `aio_485f2eef88145dedd0f8f82e61431afd`，coser `aio_eadc82c025aaef877757b446c26fee2d`。
  浏览器已核对更新后 8k 断点与空源提示，4k 数量递增；没有重新建立 operation。
- [6010 失败详情截图](../evidence/ui-audits/2026-10-23-face-retry/6010-empty-source.png)。
  这是现库运行复核，不是拿原媒体作自动化测试；未改动原图、人工标签或人脸整理，不涉及 server222。

四库最终状态已在本文顶部及下方 10:16 记录补齐；此前“尚未完成”均为对应历史时间点。

## 后台跟进 04:19

双容器 healthy、restart=0、OOM=false；主容器 database-check 再次 passed。
本次仅只读核对正式 UI、容器健康/完整性及既有停写备份元数据，更新进度；未部署、重启、提交新任务或改动原件/人工整理。
没有生产代码变更，不重用上一轮测试作为新执行结果。4k 任务已结束但覆盖有 1 张失败，其他三库仍运行/排队，继续保留跟进。

## 后台跟进 04:48

正式 UI 确认 8k operation 于 04:33 结束，792 项均已尝试，785 成功、7 个 invalid_media；任务级错误为空。
4k 终态不变，pixiv 接续至 369/999（失败 0），coser 尚未领取。总 2 库结束、2 库仍在运行/排队，非全部完成。
双容器 healthy、restart=0、OOM=false；容器内 database-check 再次 passed。
本轮仅读取正式界面与健康/完整性检查、更新进度；未提交新任务、盲目重试空文件、重启或部署，也未改动原件/人工整理。
无新生产代码变更；文档 `git diff --check` 通过。后台跟进继续，不能将现库运行复核当作新的自动化回归或发行证据。

## 后台跟进 05:19

pixiv operation 于 05:14 结束，999 项均尝试、998 成功/1 invalid_media；按上述原图副本只读诊断定位到 JPEG 码流错误，
不是任务级错误。coser 已接续到 73/6202、失败 0；4k/8k 终态不变，四库尚未全部结束。
双容器 healthy、restart=0、OOM=false，主容器 database-check 再次 passed。
未部署/重启/重复提交，未修改原件或人工整理，无生产代码变更。
诊断副本保留在本机 Downloads，临时只读 JPEG 解码命令位于 `/private/tmp/foliopath-face-source-diagnostic-xeYlMb/main.go`，
没有加入产品或测试目录，不作为自动化回归通过证据。文档 `git diff --check` 通过，继续保留后台跟进。

## 后台跟进 05:51

coser 持续推进，747 项已尝试（成功 720、invalid_media 27），其他三库终态不变；双容器 healthy、restart=0、OOM=false，
主容器 database-check 再次 passed。没有部署、重启、重复提交或修改原件/人工整理。
针对新出现的输入失败，仅从认证的正式媒体接口保存两个副本供手工根因诊断；未把真实媒体放进自动化测试或产品镜像。
诊断镜像基于同版构建/运行依赖，临时容器 network=none、只读、1 CPU/512 MiB，仅通过 internal/files
读取一次只读 `/library` 副本树，没有生产 DB/模型/网络挂载，也没有调用人脸推理。
生产解码与逐阶段检查均复现 JPEG 标记异常，临时容器已自动移除；诊断工具与副本留于
`/Volumes/extended/foliopath/face-source-manual-diagnostic-YFkuvk`（副本 sample-001 / sample-002 分别对应 ast_231 / ast_552）。
本次未修改生产代码，未宣称完成程序修复或新的合成回归通过；文档 `git diff --check` 通过，继续跟进最后一个库。

## 后台跟进 06:18

coser 正常递增至 1,360 项已尝试（成功 1,333、invalid_media 27）；失败数量、分类和可见样本无新增，
任务级错误为空，其他三库终态不变。双容器 healthy、restart=0、OOM=false，容器内 database-check passed。
仅只读复核并更新进度，无新任务、部署/重启、源文件或人工整理改动；文档 `git diff --check` 通过。
仍在处理中且无新可行动变化，本次保持静默，后台跟进保留。

## 后台跟进 06:48：新主应用 OOM

发现主应用 restart=1，StartedAt=06:31:21，当前 inspect 的 OOMKilled=false。
进一步读取 Docker Desktop 内核/daemon 历史日志，明确确认同一个主容器被其 1 GiB 内存 cgroup 杀死、exit=137、非手动重启；
不把当前 OOM 布尔值当作无历史 OOM 的证据，也不错误归因为 scan-worker/SQLite。
coser 同一 operation 已续跑，逐图失败仍 27、任务级错误为空，其他三库终态保留；容器内 database-check 再通过。
当前未改生产代码或部署，开始在独立临时合成 JPEG 上比较解码峰值和线程清理；结果记录另见内存维护记录。

07:00 追查补充：合成 33 MP 的 120 次对照和 96.8 MP 的峰值初测完成；线程清理/仅限 Go 堆均不足以证明完整应用
1 GiB 下安全，不上线未经验证的参数。六个私有已退出探针容器清理，输入/工具保留；原任务继续推进，主 restart 保持 1，
AI 为 0，双服务 healthy，database-check 再通过。文档 git diff --check 通过；内存修复仍开放，后续 heartbeat 继续读取内存记录推进。

## 后台跟进 07:22

coser 推进到 2,716 项已尝试（成功 2,687、invalid_media 29），其他三库终态不变。
新增两个同类输入失败未在最多 20 个早期样本中显示；不由旧样本反推新增资产/字节原因。
任务级错误为空；主 restart=1、AI restart=0，双服务 healthy、容器内 database-check passed。
本轮纯合成常驻堆资源探针已复现 OOM，回收组合仍触顶，不算修复通过；
详见[主应用内存记录](FIX-2026-10-23-main-face-memory.md)。
未改动原服务、任务、源文件或人工整理，没有新的生产/完整回归通过声明。
本轮执行四组 CLI 合成对照和在线检查、文档 diff 检查；未重跑全套 make。
四个已退出、镜像/状态/所有权核对后的临时探针容器已删除；合成输入、工具和镜像保留，可重建复测。
原 6010 没有因本轮主动重启，跟进继续。

## 用户要求取消内存限制（07:44–07:46）

用户明确授权不再限制内存。已增加仅覆盖 RAM/swap 四个字段的 Compose 文件；基准文件不变，
后续同项目更新必须同时传入两个文件。先验证临时 Docker 0/0→cgroup max/max 与 256 MiB 合成分配，
再停止主应用并完整复制 data/env/Compose；停写备份 quick_check=ok、外键无行、4 库/8,719 项。
重建同版 AI 后待 healthy，再恢复同版主应用；无迁移、原图/人工整理改动或任务重提。
两服务 healthy、内存/交换上限 0，AI cgroup max/max、主容器内 database-check passed。
UI 人物进度从重建前 3,195 继续到 3,213/6,202（成功 3,184、失败 29），四条原任务记录保留，任务级错误为空。
旧主容器 OOM/restart=1 的历史证据仍保留；新容器 restart=0 不构成故障已修复证明。
验证：Compose config --quiet、定向 TestAICompose、完整 make arch-check、本地 Docker 无数据合成探针和原实例运行检查通过；
仅部署配置/架构测试/文档变更，没有重跑完整应用集成、E2E 或发行验收。临时探针容器已移除，原 data 和停写备份保留。

## 后台跟进 07:48

coser 原任务继续递增到 3,257 项已尝试（成功 3,228、invalid_media 29）；与 07:46 相比无新增失败，
任务级错误为空，其余三库终态保留。两服务 healthy、当前新容器 restart=0/OOM=false、Memory=0/MemorySwap=0，
主容器内 database-check passed。历史有界配置的 OOM 不因这次健康检查关闭。
仅只读运行检查和进度文档更新，未提交/重建/清理/重启或更改原件、人工整理与内存配置；
没有新生产代码或自动化回归通过声明。后台跟进继续，普通运行进度静默。

## 后台跟进 08:18

coser 推进至 3,921/6,202（成功 3,892、invalid_media 29），失败数量/类型及最多 20 个可见样本未变，
任务级错误为空；其他三库终态不变。两服务 healthy、当前新容器 restart=0/OOM=false、Memory=0/MemorySwap=0，
容器内 database-check passed。即时 Docker 内存采样主应用约 198.8 MiB、AI 514.6 MiB，不作为峰值或长期容量通过证据。
仅只读复核与文档更新，未重提任务、部署/重启、扫描/清理或更改原件、人工整理、模型/额度。
本次无生产代码变更；文档 git diff --check 通过，不复用此前测试冒称本轮重跑。四库尚未全部结束，继续静默跟进。

## 用户查询实时状态 08:47

正式 UI：coser 4,551 项已尝试，4,522 成功/29 invalid_media，任务级错误为空；其余三库原任务终态保留。
本轮四库共 6,842/8,493（约 80.6% 已尝试，不是全部成功率/项目完成率），6,804 成功、38 失败、1,651 尚未尝试。
两服务 healthy、Memory=0、当前容器 restart=0/OOM=false，主容器内 database-check passed。
38 条尚未解决：6 个 0 B 原件、2 张未进一步区分损坏/误命名的 PNG、1 张已确认码流错误的 pixiv JPEG、
29 张 coser invalid_media（仅两个代表样本已完成具体 JPEG 字节原因诊断，其余 27 张不补造精确原因）。
放开容器上限后的运行恢复不等于解码内存峰值已修复；源文件恢复、其余逐图根因与质量/容量/发行门禁不因此通过。
本次仅只读状态/完整性检查与进度文档更新，无任务提交、重启/部署、原件/人工整理改动或新增自动化回归通过声明。

## 后台跟进 08:49

coser 继续至 4,583/6,202（成功 4,554、invalid_media 29），无新增失败、任务级错误或重启；
两服务 healthy、Memory=0/MemorySwap=0，主容器内 database-check passed。其他三库终态保留。
仅只读界面/容器检查及文档更新；未提交、部署/重启、清理或改动源文件/人工整理，继续静默跟进。

额外只读核对 07:43 已停写完整备份 GDxoRm（不是活跃 DB）：同一 coser operation 的 29 条失败记录
均与备份源指纹匹配。接口 20 样本之外的 9 项现已有完整资产 ID/元数据清单：

| 资产 ID | 文件名 | 备份大小（B） | 备份尺寸 |
| --- | --- | ---: | --- |
| ast_568 | 41_2023_12_18_222356.jpg | 1,481,520 | 1179×2556 |
| ast_577 | 46_2023_12_18_222548.jpg | 1,308,301 | 1179×2556 |
| ast_588 | 49_2023_12_18_222611.jpg | 1,573,450 | 1179×2556 |
| ast_590 | 48_2023_12_18_222108.jpg | 1,574,566 | 1179×2556 |
| ast_595 | 45_2023_12_18_222758.jpg | 1,605,869 | 1179×2556 |
| ast_597 | 43_2023_12_18_222500.jpg | 1,494,305 | 1179×2556 |
| ast_707 | 1 (37).jpg | 724,401 | 1179×2556 |
| ast_2596 | 002.jpg | 2,483,400 | 未成功探测 |
| ast_2715 | 076.jpg | 23,006,170 | 8742×11656 |

29 条均为非空输入，但尺寸/文件组并不完全相同；尤其最后两项不能套用前两个代表样本的 JPEG 标记原因。
本次没有打开或重新解码原媒体，元数据清单不是其余 27 张精确根因已解决的证据，也不把历史备份当实时数据库。
文档 git diff --check 通过，无新生产代码或完整回归通过声明。

## 用户要求继续逐图排查与内存修复（09:31 进度快照）

用户授权的剩余 27 张已逐张复现并保存明确原因，见上述独立源诊断记录；没有把旧两个样本原因套到全部文件。
内存先只做人物回收的候选在完整合成应用仍 OOM，已继续修正为所有大图 native 路径共享一个可取消槽，
并在关闭 ImageRef/线程资源后回收 Go/native 空闲页；未知 caller reader 不被新增读取，不改像素/模型身份/容错策略。
固定候选 c9f28d7a 在带 96.8 MP 合成 JPEG 的完整 Docker 联验通过（d886e552），原件 hash、真实模型、
缺失只重试负例、幂等、扫描等待、显式恢复重启均验证；显式重启前无自动重启。
原库 09:31 为本轮 7,782/8,493 已尝试，711 尚未尝试，当前原容器健康、database-check passed，
本轮未主动重启/部署/重提或清理，不把候选通过说成当前线上已修复，也不宣称四库全部完成。

## 后台跟进终态与单次更新 10:16

正式页面与本次停写独立备份共同确认四个原 operation 均 succeeded，任务级错误为空，
coser 于 10:03:35 结束（原时间戳 1792721015477），6,202 项均尝试。
本轮 8,452 成功/41 invalid_media；加保留的 207 个同代次结果，当前模型共 8,659/8,700 张可分析图片成功。
不是 8,493 全成功，不把 19 个非图片媒体或旧模型结果混入成功率。

最后新增 ast_5973/6001/6005（040.jpg、044.jpg、037.jpg）逐张只读诊断：
三者 native 像素读取均报 `16380 extraneous bytes before marker 0xda`，Go 容错 JPEG 可解，
生产严格路径为 invalid face output；不是空文件/模型超时/OOM。完整工具、结果与恢复要求见逐图记录补充。
41 条剩余来源要求：6 个 0 B 原件需恢复；2 张非空无效 PNG 需有效替代源；1 张 pixiv JPEG、
30 张 coser 标记异常 JPEG、1 张截断 JPEG 需恢复规范完整来源；1 张错扩展名 PNG 需外部更正格式标记。
没有自动修改、重编码、改名或重复提交这些输入。

四库终态后停止主应用，确认 exited，完整复制 data 结束后才更新，备份目录：
`/Volumes/extended/foliopath/backup-6010-face-final-20261023-0ziOXX`（含旧 ai.env、两份 Compose 和 data）。
仅对该 detached immutable 副本检查：quick_check=ok、foreign_key_check 无行，4 库/8,719 媒体，
四个原 operation/当前代次成功数均保留；人工标签/收藏/people/anchors/删除抑制各为 0。
没有从 macOS 打开活跃 SQLite，也没有以恢复备份覆盖现库。

随后仅将主应用固定至 `sha256:c9f28d7a6f4deb1d7d83bc7b73f0fd45e41609d546ac77a3b9c9543b652d9ec2`，
AI 仍 fa7750b5 原容器，Compose 同时使用 unlimited-memory 覆盖，主/AI RAM 与 swap 均 0（无容器上限）。
两服务 healthy/restart=0，主容器 database-check passed；更新后登录有效，逐库 UI 785/792、706/707、998/999、6170/6202，
4 个原任务完成记录均在、无新提交。新启动常规扫描不冒称新 AI 验收，人工整理与原件未主动改动。
已有合成 Docker 测试证据沿用对应固定候选，不伪称本轮重新执行；本轮仅部署/手工只读诊断/健康与终态复核，
文档 diff 检查通过。1 GiB 压力仍触顶，长期/容量/完整回归/发行门禁未关闭。
四库跟进条件已满足，应用确认 heartbeat 6010 已删除；等待用户在应用外恢复有效来源后再有针对性地扫描/补齐。
收尾再次核对：新主容器 dff3ff159bcb… 与原 AI f276fc1d60de… 均 healthy/restart=0、RAM/swap=0，
主容器 database-check 再次 passed；退出手工诊断容器已移除，私有结果/工具/镜像保留。
