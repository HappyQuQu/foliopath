# OCR处理限制与本地SQLite读取异常

**最新终态（2026-10-26 15:20 +08后验收）：pixiv 998/999 ready、1张源损坏、0待处理、0过期。** 原六张超限图全部当前身份/当前源ready；仅补15张的任务已结束，未重新计算原984张。6010已更新OCR坏JPEG分类和四卡统一布局，完整备份、保护、实际搜索及容器内database-check通过。质量数据按用户要求暂缓，正式发布仍No-Go。

以下保留各阶段的历史记录；早期“运行中／未部署”仅代表当时状态。

2026-10-26；POST-MVP-5 S2既有维护；FR-OCR-001～005、FR-SEARCH-001～003、NFR-OPS-001/INT-010。Owner：imagevips/OCR producer/本地部署。用户授权修复、隔离Docker验证后续补，原件及已完成结果保留，10个已知源问题继续跳过；不连接server222或提交推送。

## 已确认原因和候选范围

- 原5张JPEG头/源有效，但PNG无过滤编码产生8.5～10.0 MB，超过既有8MiB输入合同。PNG行过滤仅改变无损编码，不改像素、尺寸或OCR身份；合成1900×1900 JPEG在旧实现复现8,686,398 B超限，候选需证明解码后逐像素一致。
- 原任务checkpoint8204之后的8205派生图在隔离只读模型诊断中产生373个原始轮廓，旧策略把原始轮廓上限等同于128条有效文字行，尚未筛选就拒绝。候选独立给原始轮廓4096上限，保留最终128行/8192字、裁剪像素16Mi、736检测分辨率和既有调度。新的输出策略/producer指纹必须另立，不能给旧479张重新贴新身份；用户已明确同意保留旧结果并对pixiv全部999张按新身份识别。
- 小库120秒并发扫描/读取/在线检查（511次扫描、21910次请求、213次检查）未复现。但64MiB合成库在宿主bind下复现PROTOCOL(15)，第二次出现`_walFramePgno` SIGBUS；固定Go1.26.5仍复现PROTOCOL(15)。同一程序在Linux卷通过1280次更新/160次独立进程检查，因此按[ADR-0038](../adr/0038-docker-desktop-linux-sqlite-volume.md)移除宿主共享存储层；不宣称已逐次证明全部历史退出。

## 数据与交付门

原续补failed/output_limit及479 ready/5输入限制/515pending保留；人工标签、删除抑制、人物、语义均不动。修复测试使用合成媒体/独立数据；真实问题图只作经安全文件边界的只读诊断。以下部署已完成停写备份/逐文件核验及单次新任务提交。不得把源无文字、未处理或旧身份结果计为新策略成功。


## 已执行验证

- 新合成JPEG在旧实现失败，在新实现通过；PNG从8,686,398 B减至5,539,976 B，解码逐像素一致，输入合同SHA不变。libvips的OCR相关测试通过。对原5个问题输入做只读诊断，过滤后约6.0～7.6 MB均在8MiB内，源指纹保持；不是拿真实媒体做单元测试。
- 新producer的16项Python合同/HTTP/调度/轮廓测试通过，144个微小噪声轮廓可在筛选后得到0文字框，超过4096的原始轮廓仍拒绝。原8205的373轮廓真实只读诊断不再被原始轮廓门提前拒绝，得到ready/0行；0行不能算文字准确率。
- 固定真实权重的隔离合成Docker通过：完整app/认证/API/预处理/持久化、三OS进程同job恢复、两个坏源与空白分类、OCR-only/融合HTTP搜索、clear后无命中及非空人工/源hash/完整性。AI镜像7739486c、probe6784e9c；OOM/max/oom_kill均0，测试预算4GiB，不等于联合4GiB容量通过。
- Linux数据卷120秒应用回归：22,634请求、522次扫描、220次同容器database-check，无非计划重启；另显式重启恢复通过。较大WAL复现源和日志留在私有artifacts，比较测试使用同SQLite1.54.0与Go1.26.5，不以Go1.27初轮记录代替固定版本结论。
- `make fmt`、最终fmt-check/arch-check/generate-check/lint及runtime E2E通过。Go全量执行的唯一失败是宿主模型安装集成测试触发实际磁盘10%预留（主盘和外置盘均低于预留），没有放宽生产空间策略；完整integration包在隔离Docker/3GiB tmpfs重跑通过。其余Go包及三个OCR相关完整包通过。无UI代码修改；未重跑全产品浏览器/质量/容量/双架构发行矩阵。

## 停写备份、Linux卷与当前部署

主应用正常exit0后，完整data/私有env/六份配置文件备份到`/Volumes/extended/foliopath/backup-6010-ocr-recovery-20261026-ymqn7wf2`，37,376文件、1,096,416,994 B；源/备份逐文件相同，清单SHA `e061a12189fb4a3ea9d8cba82c014c2022b6d3669c48da3b0775c857ea34289a`。完整复制到外部Linux卷`foliopath-ai-original-review-data-linux-ymqn7wf2`后再次逐文件核对，旧/新主镜像database-check均通过。旧宿主data留作历史副本，不再用于在线审计或启动。

当前主`sha256:067c1159eaa616820a38fcefc17ff12b8b1b534ac25ee7cc1a1f3402a7b0406d`于13:03:20+08启动；AI`sha256:7739486c053d5d1107a57546eb88a9ea17cc939dd1585af265483b5f2d3a6572`于13:03:07启动，本轮AI确已重建。语义640/face/153tag逐端点身份不变；五份Compose/base+unlimited-memory+multilingual+ocr+data-volume、单/library:ro、AI无数据库/媒体及无限内存/交换配置均核验。

初版候选使用较早old-dist，部署前资源核对拦住；最终从当前a768冻结web构建，JS `index-BHO1qG7g.js`/CSS `index-BCJs_lhG.css`与更新前真实浏览器一致，保留标签云/五档字号颜色/拖宽/自适应。第一次复制工具误用无shell的精简主镜像，未执行任何复制即失败；改用固定Debian离线辅助容器，只从已停写备份向空卷复制并核验，未复用未验证副本或启动空库。

切换前后31组固定保护摘要通过：27组旧能力、原23张、原cancelled任务、旧OCR484行代次和两个原job均保持。此处原库人工/抑制0，不冒充非空保护测试。用户通过原有效管理员会话关闭再开启**仅pixiv** OCR，创建epoch4、新身份`89e3e889c5d124b914aa524fb593bde83e56f94016c0ef702a9e3f503231ea64`，只提交一次missing job `ocrjob_16133bb86bc37e890f99a833d2840483`（upper8719）。原479成功/5失败留在旧身份，原failed/output_limit和cancelled终态不复活。

收尾快照：新任务running，completed23/failed0/attempt1，当前源/新身份ready23、failed0、pending976，不是999张完成。10源仍无OCR写入，其他库未开OCR。后台跟进`pixiv`将核对六个特殊输入、新代次终态、旧数据/容器/实际搜索后再更新本记录并汇报；期间不重复提交或重启。

[机器证据](../evidence/ocr/2026-10-26-ocr-limits-sqlite-recovery.json)，私有完整记录`.artifacts/ocr-repair-20261026/`。未连接server222，未提交推送。

结束本轮前快照：新job仍running，当前新身份ready70/999、failed0、pending929，checkpoint7790、attempt1、error空。原先超限7789已按新身份/当前源ready并产生2行真实结果；其余五个特殊输入仍待任务处理，不提前算通过。31组固定保护仍全部相同。

2026-10-26 13:51 +08并行验收快照：同一新job running/completed526/failed0/attempt1、checkpoint8246、error空，当前源/新身份ready526、failed0、pending473、stale0；347张无检出文字，其余179张532行。六张7789/7923/7987/8073/8194/8205均当前源ready，实际行数2/1/0/2/0/0，不把零行称作准确识别。31组固定保护不变、10源无OCR写入、原主database-check通过，双healthy/restart0/OOMfalse；仍等待999终态与真实搜索。用户另授权整版回归、质量和供应链三任务并行，见[验收记录](AUDIT-2026-10-26-release-readiness.md)，不重启部署干扰本任务。

## 新终态与JPEG损坏分类修复

14:32 +08原新job终结为failed/preprocess_failed：当前身份984 ready、0资产失败、15pending；不是全库完成。648张未检出文字，其余336张共1059行。六张原特殊输入均当前源ready，31组固定保护保持。未复活或重提终态，也未执行布局部署。

按internal/files安全边界在原主Linux锁域做15项有界只读诊断：8705为3000×1501/4,389,774 B JPEG，头有效但完整标准解码报告`invalid JPEG format: missing 0xff00 sequence`；原件SHA256为`8319b41bf19d99d5f3a212cfccfe861ae6308ce5fe7350c0378915c8d670ef4a`，当前源指纹保持。其余14张均标准解码成功。该文件是原10项以外的新源损坏，不改原件、不标成功。VIPS在同时间记录tile解码警告，但当前OCR预处理将所有lazy-native失败都映射为程序故障，导致后续14张被阻断。

拟在既有FR-OCR输入失败维护范围修正：仅native处理失败且完整JPEG标准解码给出明确格式/截断错误时归类invalid_media；完整有效JPEG或未知native故障仍为preprocess_failed，IO和取消优先级不变。校验发生在输入预处理owner的失败分支，沿用既有64MiB/40MP和大图准入，不增加第二调度器或改变成功像素/模型身份。先用合成JPEG坏熵流、有效图/未知native故障与取消做真实libvips回归；通过后停写全备份，只部署OCR分类修复，按既有显式missing机制补15项，保留984结果和旧failed终态。布局在该次补齐终态、错误/搜索核验后再更新，不夹带改变其他能力。

## JPEG分类修复、补齐和最终验收已完成

仅在OCR预处理native失败分支对JPEG做完整标准解码：明确`jpeg.FormatError`或截断才记源损坏；有效JPEG/未知native错误仍为处理错误，取消与IO边界保留。成功像素、OCR身份、任务代次、其他能力均不变。合成native/race首轮46通过、2项因缺渐进式fixture跳过，随后独立生成fixture将这2项补跑通过，共48项；有效图/坏熵流/取消分类覆盖。真实模型合成Docker三进程同job恢复、坏JPEG后继续、文字/空白分类、全文/融合HTTP和非空人工保护通过，未拿真实媒体作测试fixture。

15:06 +08仅主部署`7ff761c3`，完整停写备份在`/Volumes/extended/foliopath/backup-6010-ocr-source-classification-20261026-9u0olthv`，37,376文件/1,097,187,042 B，逐文件及独立Linux副本/新旧CLI检查通过。原任务`ocrjob_16133bb86bc37e890f99a833d2840483`保持failed/preprocess_failed终态；有效会话显式missing仅提交一次`ocrjob_4acd7be5b914d57c287eb553f19a39d4`，epoch4/upper8719/原89e3身份，终态succeeded，completed15含failed1、checkpoint8719、error空。此succeeded表示调度结束，不表示每张成功。

最终998 ready中658张未检出文字，340张共1063行；不是正式准确率。剩余8705为新增源损坏，文件`Alcxome_Forest radio_p1_89806011_35562628_2631x1488.jpg`，实际3000×1501/4,389,774 B，标准解码缺`0xff00`；需要用户日后从可靠副本恢复原文件后再显式补齐，本轮不改名/修复/覆盖原件、不循环重试或标成功。原10项其他库问题继续跳过且无OCR写入。

六图7789/7923/7987/8073/8194/8205实际行数2/1/0/2/0/0，均current_source=1；31组旧能力/OCR历史保护全相同，22365语义向量保留；新增原984成功行与原failed job独立hash前后相同。原库人工/抑制起始0，非空保护证据来自合成Docker，不能混淆。

终态后浏览器在当前6010实测`Morinaga`：图片文字模式1项、全部来源50项首屏，均可预览“图片文字”和原文`Morinaga`，没有工程统计或识别行号；融合并非50项都OCR命中。最终布局版本更新后再次验证相同结果。搜索的简短未完整处理警告与1张源失败相符，无新model_unavailable/输入超限/任务级处理错误；旧failed任务继续保留历史。

15:20:50 +08又仅主更新`sha256:b34190013455112760db3fd94c22ba119b3f66e29f8c1641f391c49d4666491f`，包含该分类修复及四卡网格；后端/迁移与7ff逐文件相同。最终完整停写备份`/Volumes/extended/foliopath/backup-6010-ai-feature-grid-final-20261026-uvx2wk9r`，37,376文件/1,097,211,618 B，manifest `bf5fd26d57d1f8dde8b855e94b5b0bdab1f9e74a6c2ddb26114ca0d460f62c63`；新旧CLI/独立Linux副本检查通过。AI仍7739486c，ID/13:03:07启动时间不变；全部五Compose、Linux卷、原件只读、无限内存/交换保留，双healthy/restart0/OOMfalse。所有OCR结果、FTS、任务及人工/删除记录前后摘要相同。后续状态见[布局记录](FIX-2026-10-26-ai-feature-grid.md)，本次后台跟进终止。
