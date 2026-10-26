# FIX-2026-10-24：自动标签大PNG与人工保护续验

最终后续：[model_unavailable修复与续补](FIX-2026-10-24-autotag-service-recovery.md)已通过合成Docker并备份更新6010主ff2ee7e4，
仅8k/pixiv的missing续补均结束，21:08核验四库8690成功/10既有源失败/未处理0，22PNG及保护摘要一致；旧失败operation保留。
本文件运行/排队描述为历史快照，不作为当前运行镜像/任务状态。

- 范围：POST-MVP-5 / FR-AUTOTAG-001～003 / CR-2026-031、033；Owner app/imagevips/autotag/curation。
- 决策：[ADR-0029](../adr/0029-independent-tag-png-input.md)。状态：实现/本地Docker通过，四库新身份补齐及22PNG终态验收完成；10项源失败暂不修复，非质量/发行通过。最新结果见上方续补记录。
- 仅本地6010，不连接server222；10项源错误暂不修复，原件只读。

## 当前证据

11:48 +08主应用正常退出（exit0、application.stopped），确认全局无活动分析operation后，完整备份至
`/Volumes/extended/foliopath/backup-6010-tag-png-v4-20261024-INjxqU`（权限700）。
data共37376文件、38404条目、1093361378 B；逐文件SHA256及4份私有配置/Compose对比一致，manifest
`110afd5e25ae776a8a9bcfd8d789dee0b9cc9deeb7aacd58d2763f55c613f8d4`。
备份前原主容器database-check通过；新语义代次8690向量、旧8668向量与15表摘要已记录到备份before.json。
原版恢复后继续开发，备份不包含原始媒体或模型；不从宿主打开活跃SQLite。

备份初次以整个目录只读挂载执行database-check失败：正常停机后没有WAL/SHM，SQLite WAL模式只读打开需要可用sidecar条件。
随后仅从**已脱离运行库的备份**复制数据库至`tag-backup-validation-20261024-4V2HnH`，在独立Linux容器可写验证目录执行
同版database-check通过；原备份仍只读未改。前后DB SHA256一致：
`3d57baae6274ba8809a3af0ab32c864a134cf854a968c82b8c63e15b6b968573`。
恢复的原主容器database-check再次通过，未把首次失败隐去或宣称运行库损坏。

## 逐项定位与实现

原主容器Linux锁域15秒只读快照核对32个旧failed run，均匹配当前源指纹：
coser24项=22张有效大PNG+2596/2715源问题；4k1项=6669；8k7项=六个0 B与7374无效PNG；pixiv0。
22张大PNG同时存在当前png-v4语义向量，目录尺寸/大小命中旧tag16 MP或32 MiB限制。
标签历史只存outcome，没有逐项解码原因码；上述归因基于目录、既有源诊断和实际输入合同，不补造runtime日志。
本轮没有盲目重新提交旧身份任务或改源文件。

实现新增独立`tag.image_preprocessing`配置校验，并将其canonical contract同时绑定tag fingerprint和imagevips实例；
默认仍为原v3哈希，顶层语义profile不能改变标签身份，反之亦然。AI权重/92类策略、阈值、语义和人物身份不变。
没有新迁移、端点或UI；生产组件与稳定Story不变，需求/流程/数据/运维和架构约束已同步。

## 计划验收

独立tag profile/身份配置回归 → 合成native PNG边界 → 真实JoyTag Docker换代与人工/抑制/重启保护 →
停写匹配备份后更新6010主镜像/tag profile → 真实新身份覆盖与22PNG结果 → 更新进度。
既有32标签失败=22大PNG输入限制+10源问题；不能将语义成功自动算作标签成功。

## 本轮实际验证

- Go1.26.5：`CGO_ENABLED=0 go test -count=1` app/autotag/curation/sqlite/imagevips/architecture/contract全部通过；
  app/curation/sqlite的AutomaticTag/Localized/PNGProfile定向`-race -count=1`通过。
- `make fmt-check arch-check generate-check`、`CGO_ENABLED=0 make lint test`通过（包含全包go vet/test）；
  扩展盘TMPDIR下`make test-integration test-e2e`通过。CGO=0不代表原生解码路径，原生证据另列。
- Linux arm64 native图像Docker：36通过、2个progressive JPEG测试因无显式合成fixture跳过；
  1 CPU / 1 GiB RAM+swap上限，exit0/OOM=false。覆盖大PNG文件/内存流、40MP边界、超限/坏源、alpha/16bit、
  取消/无进展、共享准入和引用回收，没有移除安全输入边界。
- 固定真实权重合成Docker `fp-multilingual-406ef5d5`最终exit0：合成alpha旧标签2成功/3失败 →
  仅tag profile换代后3成功/2故意坏源失败，`large.png`实际存在AI标签而非只有ready计数；beta标签仍关闭。
  人工中文同义优先、删除抑制跨换代/全量重跑/主应用重启有效；原件SHA256一致，容器内database-check通过。
  语义/人脸代次及覆盖未变、AI StartedAt未变；前台中文查询与两类后台任务并发通过。
  主峰值169439232 B，AI峰值3710689280 B；4 GiB合成AI预算下memory.events max/oom/oom_kill均0。
  主/AI自动RestartCount=0，显式测试重启不冒称未重启。此结果不关闭默认小模型预算/长期/准确率/发行门禁。
- 主候选固定`sha256:805bdf8063ecb2fd3aa2a3520dd926a3012248235ba3534291dd6cac9d5c252c`，
  `spikes/automatic-tags/Dockerfile.localized-app`增量native构建；AI保持
  `sha256:3631b43219c74bc4e18df503d53e144e9bc86f80bc9490efa1d1e65d356ea903`。
  Web阶段Node24.22.1类型/Vite构建通过，依赖构建报告2个moderate与chunk大小警告；未宣称浏览器全矩阵/前端全量或安全发行通过。
- 首次启动误把构建config digest当image ID导致未启动测试，改由image inspect取得固定ID；
  首轮联验已通过换代人工保护，但旧后续断言仍要求已全部人工整理的样例含AI关联而失败。
  修正为从未整理的大PNG上断言真实AI输出，保留原模式断言，再完整重跑通过；不把初次失败当通过。
  最终合成容器/网络按其专属project清理，合成数据/scratch保留；未触及原库。

## 6010部署与当前进度（2026-10-24 12:17 +08:00）

开发期间原版恢复运行产生启动扫描写入，故部署前重新正常停机（12:16:19 +08，exit0/OOM=false），
确认无活动分析operation，再完成第二份当前时点全备份：
`/Volumes/extended/foliopath/backup-6010-tag-png-v4-20261024-IQVMkt`（700）。
data 37376文件、38404条目、1093373666 B；4配置逐文件核验一致，manifest
`8c90e572cc1e9bc2b1b57781dabdd050c7c03ceec9426680278456529798b311`。
独立Linux验证副本database-check通过，数据库SHA256
`84a4c80da3f1343818f24b02b51b329a78f7277dbeb1d4a86a2704e50a5183ea`；原备份未改。

仅替换主镜像为上述805bdf80，以及私有deployment JSON新增`tag.image_preprocessing="png-v4"`。
去除这两项后其他env行、完整JSON和三份Compose逐字一致；单个`/library:ro`和原data挂载不变，RAM/swap=0覆盖保留。
主12:17:25 +08启动后healthy/restart=0/OOM=false，AI3631b432与StartedAt保持原样未重启；新主database-check通过。
新tag fingerprint为`cc65dcdd3dd9b03697dd07ace62e11900d69a804241968de8b2724dae7d9ec3d`，
旧92类模型/policy/runtime identity JSON均不变，未启用153类。

13组保护摘要（手工标签、抑制、收藏、人物/人脸各表及**全部22365语义向量**）部署前后逐项行数/SHA256一致；
所有语义向量摘要`5feebf36c0c47eada2b99a013d985495c86593f5f89b271272bf2e23ef2d862f`，
活动语义代次仍`aig_Nij9FMsPIw3d9dlzUqAKbkLV`，未提交语义或人脸任务。
真实库当前人工关联/抑制为0，非空保护由上述合成Docker验证，不能以原库空计数冒充保护测试。

四库原开关均开启，既有reconcile各自动创建一次新身份missing，没有手动重复提交：

| 库 | 新标签operation | 初始状态 |
| --- | --- | --- |
| coser/lib_1 | `autotag_62e73b529dc0108c6c0ed00add3c0c3f` | running，attempt1，首快照6成功/checkpoint6 |
| 4k/lib_2 | `autotag_6a4259f0709a713e030526764b1eb747` | queued，attempt0 |
| 8k/lib_3 | `autotag_a94a77918a83e1563379c46c4a03e380` | queued，attempt0 |
| pixiv/lib_4 | `autotag_b12e0cc38ded583df2aae8bf8cff168c` | queued，attempt0 |

四operation任务级error均null；上述仅启动快照，旧8668个标签成功不能算新身份完成。
正式浏览器原会话仍有效，刷新后4库/8719资产存在；coser新标签进度从34增加到46，语义6200/6202、人物6170/6202未变，
4k显示0/707、707待重新识别，而非把旧706成功当新版覆盖。未点击重跑/开启按钮或添加原库测试标签。
22张目标PNG在此快照尚未到checkpoint，仍保留旧failed run；**尚未声称原库22张标签已修复成功或全库打标结束**。

10项源问题仅暂不修复，不改原件，也不伪造成功/删除历史；既有新身份遍历可能再记录源失败，不自动循环重试。
用户要求的“先跳过”不等于新增逐图排除API或修改数据库来把它们标完成。后续只核对以上四operation及新身份真实结果。
已询问是否授权后台跟进至终态；截至本快照尚未收到回复，未自行新建定时任务。

12:23收尾只读核验：coser同一新operation已成功97张，checkpoint97、attempt1、任务error=null；
其余三库仍为原新operation queued/attempt0，尚无新身份失败。22张PNG均尚未到checkpoint，新标签成功0/22，
不以合成测试或语义向量冒充原库结果。13组保护摘要仍与部署前完全一致；
`make fmt-check arch-check generate-check`及`git diff --check`再次通过。未提交推送、未连接server222。

## 后台验收已授权（2026-10-24 13:26 +08:00）

用户明确允许后台跟进至终态；已创建本聊天heartbeat“6010 自动标签新版本验收”（automation ID `6010`），
每30分钟核对上述四个原operation和新标签身份，正常运行/排队保持安静。终态后记录逐库结果、22PNG与保护摘要并删除跟进；
不重复提交、不重启/部署、不修复或伪造那10项源错误，不扩展至153类/其他功能/发行。

13:26原主容器15秒有界只读核对：coser新身份ready=1195、failed=0，原operation running/attempt1、
checkpoint1207、completed_items1195、error=null；其余三库原operation仍queued/attempt0。
原13张PNG均已是新身份ready且匹配当前源指纹，12张已有AI标签（逐图2～12个）；
`ast_698`为ready但0个AI标签，表示现阈值/受控词表未输出候选，不是输入失败，也不伪造标签。
后段9张尚未到checkpoint，仍待验收；不能宣称22张全部已打标。
13组保护摘要仍与部署前一致；双容器healthy/restart=0/OOM=false，未重提或修改原件/人工整理。

## 定时验收快照（非终态）

### 19:32用户进度查询：发现任务级模型不可用阻塞

2026-10-24 19:32 +08在原主容器Linux锁域有界只读核对：四个原operation均已终态，
但8k、pixiv为`failed/model_unavailable`，不能当成全库重算完成：

| 库 | 新身份ready | 新身份源失败 | 尚未按新身份处理 | 原operation终态与checkpoint |
| --- | --- | --- | --- | --- |
| coser | 6200 | 2 | 0 | succeeded，attempt1，6202/6202，checkpoint6221，error=null |
| 4k | 706 | 1 | 0 | succeeded，attempt1，707/707，checkpoint6928，error=null |
| 8k | 636 | 5 | 151 | failed，attempt1，completed=total=641，checkpoint7569，model_unavailable |
| pixiv | 0 | 0 | 999 | failed，attempt1，completed=total=0，checkpoint0，model_unavailable |

新身份合计已处理7550/8700（86.8%），成功7542、源失败8、未处理1150；operation的641/641和0/0是此次终态计数，
不代表8k或pixiv全库完成。8k已记录的5项源失败为7038/7374/7387/7449/7510，仍属用户要求暂不修复的已知源问题。
22张PNG逐图新身份ready、当前源指纹及实际AI标签数不变；13组保护摘要仍与部署前完全一致，容器内database-check通过。

两个原容器StartedAt均变为19:30:32，镜像/容器ID/创建时间未变，当前healthy、RestartCount0、OOMKilled=false、RAM/swap=0。
主日志19:30:23记录`application.stopped`，AI日志正常shutdown后重新startup；现有证据说明发生过停止再启动，
不能据RestartCount0宣称“从未重启”，也不能据此推断OOM或确定操作者。任务级模型不可用与此次服务重启的确切时序/根因尚待排查，
当前容器健康不等于失败operation会自动恢复。本次查询未重提、重启、改任务或claim表；只读临时工具已移除。
本轮更新进度文档并执行`git diff --check`；尚未完成剩余两库，不能关闭功能验收或冒称全部修复。

| 时间（+08） | coser新身份ready/failed；checkpoint；状态 | 其他三库 | PNG与保护 |
| --- | --- | --- | --- |
| 2026-10-24 13:56 | 1744/0；1757；原operation running、attempt1、completed_items1744、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张均匹配当前源指纹的新ready，12张有AI标签、698为0候选；后9张仍未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 14:26 | 2265/0；2280；原operation running、attempt1、completed_items2265、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/源指纹及实际AI标签数不变；后9张未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 14:56 | 2791/2；2809；原operation running、attempt1、completed_items2793、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready与实际标签数不变；后9张未到checkpoint；13组保护摘要与部署前一致；2失败均为既有2596/2715源问题 |
| 2026-10-24 15:26 | 3411/2；3430；原operation running、attempt1、completed_items3413、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready、当前源指纹及实际标签数不变；后9张仍为旧身份、未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 15:56 | 3836/2；3855；原operation running、attempt1、completed_items3838、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/当前源指纹及实际标签数不变；后9张仍待新身份覆盖；13组保护摘要与部署前一致 |
| 2026-10-24 16:26 | 4347/2；4366；原operation running、attempt1、completed_items4349、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/当前源指纹及实际标签数不变；后9张未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 16:56 | 4855/2；4874；原operation running、attempt1、completed_items4857、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/当前源指纹及实际标签数不变；后9张仍待处理；13组保护摘要与部署前一致 |
| 2026-10-24 17:26 | 5380/2；5399；原operation running、attempt1、completed_items5382、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/当前源指纹及实际标签数不变；后9张未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 17:56 | 5904/2；5925；原operation running、attempt1、completed_items5906、error=null | 四库原operation/身份未变；4k/8k/pixiv仍queued、attempt0、error=null | 原13张新ready/当前源指纹及实际标签数不变；后9张未到checkpoint；13组保护摘要与部署前一致 |
| 2026-10-24 18:26 | 6200/2；6221；原operation succeeded、attempt1、completed_items=total_items=6202、error=null | 4k原operation running、attempt1、ready221/failed0、checkpoint6442、completed_items221、error=null；8k/pixiv仍queued/attempt0 | 22张全部新身份ready且源指纹一致，21张有实际AI标签、698为0候选；13组保护摘要与部署前一致 |
| 2026-10-24 18:56 | 6200/2；6221；原operation保持succeeded/attempt1、6202/6202、error=null | 4k原operation succeeded/attempt1、707/707、ready706/failed1、checkpoint6928；8k原operation running/attempt1、ready36/failed0、checkpoint6964、completed_items36；pixiv queued/attempt0；均error=null | 22张新身份ready/当前源指纹及标签数不变；13组保护摘要与部署前一致 |
| 2026-10-24 19:26 | 6200/2；6221；原operation保持succeeded/attempt1、6202/6202、error=null | 4k终态706/1不变；8k原operation running/attempt1、ready575/failed4、checkpoint7507、completed_items579、error=null；pixiv queued/attempt0/error=null | 22张新身份ready/当前源指纹及标签数不变；13组保护摘要与部署前一致 |

19:26的8k新身份4项失败逐项为`ast_7038/7374/7387/7449`，均匹配当前源指纹；分别为既有3个0 B文件和37340 B无效PNG，
按授权暂不修复，未出现新程序错误。coser/4k终态保持，8k仍正常推进、pixiv排队，未宣称全库完成。
原主容器Linux锁域15秒有界只读审计及13组保护摘要核对通过，临时工具已移除；双容器healthy/restart0/OOMfalse，
镜像/StartedAt和RAM/swap=0不变。仅更新文档并执行`git diff --check`，未部署、重启、重提、复制在线DB或改动原件/人工整理。

18:56四库继续由原operation接续，coser已终态，4k本次新增终态；4k唯一新身份失败`ast_6669`匹配当前源指纹，
为已知246125 B无效PNG，按用户要求暂不修复；不是新程序错误。8k正在处理、pixiv仍排队，尚未全库终态。
原主容器Linux锁域15秒有界只读审计及13组保护摘要核对通过，临时工具已移除；双容器healthy/restart0/OOMfalse，
镜像/StartedAt和RAM/swap=0均未变。仅更新文档并执行`git diff --check`，未部署、重启、重提、复制在线DB或修改原件/人工整理。

## 22张大PNG原库标签验收（2026-10-24 18:26 +08）

以下22张逐图均为新标签身份`cc65dcdd3dd9b03697dd07ace62e11900d69a804241968de8b2724dae7d9ec3d`、
`outcome=ready`、`current_source=1`；不是以旧run或语义向量计数推定。实际AI标签关联数如下：

| lib_1资产ID | AI标签数 | lib_1资产ID | AI标签数 |
| --- | --- | --- | --- |
| 602 | 7 | 698 | 0 |
| 603 | 4 | 701 | 12 |
| 606 | 3 | 6176 | 8 |
| 615 | 6 | 6182 | 12 |
| 619 | 5 | 6183 | 10 |
| 621 | 4 | 6184 | 2 |
| 625 | 7 | 6185 | 8 |
| 626 | 9 | 6186 | 6 |
| 627 | 5 | 6189 | 2 |
| 629 | 2 | 6190 | 8 |
| 694 | 4 | 6191 | 13 |

22项旧输入限制已在当前源指纹的新身份上全部成功处理；其中21张实际输出AI标签，698仍为阈值/受控词表下0候选，
不伪造标签或当成输入失败。coser原operation处理6202/6202并终态succeeded，资产成功6200、失败2，
失败精确为已知2596格式标记不符、2715截断JPEG；任务succeeded不代表每张源文件成功。
4k已由原operation自动接续运行，8k/pixiv仍排队，不重复提交或宣称四库全部完成。

18:26原主容器Linux锁域15秒有界只读审计及13组保护摘要与部署前比对通过，临时工具已移除；
全部22365语义向量、人物及人工/抑制摘要未变。双容器healthy/restart0/OOMfalse，固定镜像、StartedAt、RAM/swap=0不变。
本轮仅更新进度文档并执行`git diff --check`，未部署、重启、重提任务、复制在线DB或修改原件/人工整理；后台验收继续至四库终态。

17:56较17:26新增524个新身份ready；2项failed仍为匹配当前源指纹的既有2596/2715源问题，无新增程序错误。
原主容器Linux锁域15秒有界只读审计及13组保护摘要比对通过，临时工具已移除；双容器healthy/restart0/OOMfalse，
固定镜像/StartedAt、RAM/swap=0不变。后9张PNG尚未到checkpoint，四库尚未全部终态，继续安静跟进。
本轮仅更新文档并执行`git diff --check`，未部署、重启、重提任务、复制在线DB或修改原件/人工整理。

17:26较16:56新增525个新身份ready，失败仍仅匹配当前源指纹的2596/2715既有源问题；无新增程序错误。
原主容器Linux锁域15秒有界只读审计、13组保护摘要核对通过，临时工具已移除；双容器healthy/restart0/OOMfalse，
固定镜像/StartedAt和RAM/swap=0均未变。四库尚未全部终态，余9张PNG仍待真实新标签结果。
本轮仅更新文档并执行`git diff --check`，未部署、重启、重提任务、复制在线DB或修改原件/人工整理。

16:56新身份较16:26增加508个ready，失败仍仅既有2596/2715源问题，无新增程序错误；四库未全部终态。
原主容器Linux锁域15秒有界只读审计及13组保护摘要核对通过，临时工具已移除；
双容器healthy/restart0/OOMfalse，固定镜像/StartedAt和RAM/swap=0未变。
本轮仅更新进度文档并执行`git diff --check`，未部署、重启、重提任务、复制在线DB或改动原件/人工整理。

16:26新身份较15:56增加511个ready；2项failed仍为2596/2715既有源问题，无新增程序错误。
原容器Linux锁域15秒有界只读核对及13组保护摘要比对通过，临时工具已移除；双容器healthy/restart0/OOMfalse，
固定镜像/StartedAt、RAM/swap=0保持不变。未部署、重启、重提、复制在线DB或修改原件/人工整理。
本轮仅更新进度文档并执行`git diff --check`，继续安静跟进四库终态与余9张PNG真实标签结果。

15:56新身份较15:26增加425个ready，2项失败仍精确为匹配当前源指纹的既有2596/2715源问题；另17项旧身份失败不计入新失败。
原容器Linux锁域15秒有界只读审计及13组保护摘要核对通过，临时可执行文件已移除。
双容器healthy/restart0/OOMfalse，固定镜像、StartedAt和RAM/swap=0未变；未发现新程序错误。
本轮仅更新文档并执行`git diff --check`，未重启、部署、重提、复制在线DB或修改原件/人工整理；四库尚未终态。

15:26新身份较14:56增加620个ready，失败仍仅为已知的`ast_2596`与`ast_2715`源问题；其余17项failed仍属旧身份。
原主容器Linux锁域内15秒有界只读审计、13组保护摘要核对通过，临时工具已移除。
双容器固定镜像/StartedAt、RAM/swap=0及healthy/restart0/OOMfalse未变，无新程序错误。
本轮仅更新进度文档并执行`git diff --check`，未部署、重启、重提任务、复制在线DB或改动原件/人工整理；继续跟进至终态。

13:56双容器固定镜像、StartedAt、RAM/swap=0均未变，healthy/restart=0/OOM=false。
失败列表中的19项仍是尚未覆盖的旧身份记录（后9张PNG+10源问题），不是新身份19个失败。
本轮原主容器Linux锁域15秒只读审计通过、临时可执行文件已移除；未重启/备份/部署/重提/改原件或人工整理。
仅更新文档并执行`git diff --check`，未重复database-check或代码回归；正常推进，后台继续安静跟进。

14:26同样的原容器Linux锁域15秒只读审计通过，临时可执行文件已移除；新身份较13:56增加521个ready，
failed仍0。双容器固定镜像/StartedAt及RAM/swap=0未变，healthy/restart=0/OOM=false。
19项旧failed记录仍未到当前checkpoint，不能误算为新失败；无新程序错误或用户操作要求。
本轮仅进度文档与`git diff --check`，未重启、部署、复制在线DB、重提任务或改动原件/人工整理。

14:56新身份记录2项failed；只读诊断工具的失败投影补充model_identity列后，在原容器Linux锁域重新核对，
精确为`lib_1/ast_2596`与`ast_2715`，均匹配当前源指纹，属于已明确暂不修复的格式不符/截断JPEG。
另17项failed仍属旧身份（后9张PNG+其他库8项源问题），不把它们算入当前新失败。
这不是新程序错误，不重试、改原件或清零状态；四个operation任务级error仍null。
两次15秒有界只读审计及保护摘要核对通过，临时工具已移除；固定镜像/StartedAt、双healthy/restart0/OOMfalse未变。
本轮仅更新临时只读投影与文档并执行`git diff --check`，未部署、重启或运行新功能回归，后台继续安静跟进。
