# FIX：6010自动标签model_unavailable恢复

- 范围：POST-MVP-5 / CR-2026-031、FR-AUTOTAG-001/002、NFR-OPS-001；Owner autotag/remoteml/app。
- 用户授权：修复model_unavailable，保留已完成进度，补齐8k/pixiv剩余，10项源问题暂不修复；不连接server222。
- 决策：[ADR-0030](../adr/0030-bounded-autotag-service-recovery.md)。本记录不代表发行验收。
- 当前结论（21:08复核）：两补齐operation均succeeded/attempt1/error=null，四库新身份8700项均已处理，8690成功/10既有源失败/0未处理。没有新增任务级model_unavailable；详细终态在末尾，早前运行/排队描述均为历史快照。

## 事实与原因边界

19:32快照新身份cc65dcdd下coser6200/2、4k706/1终态；8k636/5后原operation失败，pixiv未处理即失败，
两者error=model_unavailable。新身份已处理7550/8700，未处理1150（8k151、pixiv999）。
19:30原主/AI正常停止再启动，镜像未变，当前healthy、OOMKilled=false；RestartCount0不表示从未重启。
只读诊断13组保护摘要与部署前相同、database-check通过，已完成数据未丢失。

代码确认：标签处理器一次非busy推理错误即终止并阻止该库自动重入；启动恢复早于AI就绪或短暂断连也会触发。
AI恢复后tag identity和ping返回200，但历史operation不复活。历史失败未保存底层网络/推理异常，不能补造具体日志或断言OOM。

## 修复与验证

remoteml严格区分临时服务错误与身份/输出硬错误；autotag按ADR-0030有界等待并复用原claim/checkpoint。
没有改模型、标签身份/词表、公开API或数据库。既有已失败任务保留，通过带修订保护的启用设置解除阻止，
既有missing调度只识别缺当前身份run的图片（保留已记录源失败），不全量重跑、不直接改任务/claim表。

部署前先完成单测/race及合成Docker服务故障回归；再正常停主/完整备份核验，仅更换主镜像，AI保持运行。
真实任务操作与最终逐库数量、22PNG/人工保护复核在实际执行后追加，不提前声称完成。

### 实际执行证据

- Go1.26.5：autotag/remoteml/app定向`go test -count=1`通过；三包恢复/processor/busy定向`-race -count=1`通过。
- `make fmt-check arch-check generate-check lint`通过；首次`CGO_ENABLED=0 make test`中下载集成2项触发本机TMPDIR
  空间保护并失败，没有将其当通过。改`TMPDIR=/Volumes/extended/foliopath`重跑`make test test-integration test-e2e`通过，
  后者为native runtime容器smoke，不宣称完整浏览器矩阵。新增SQLite显式重新启用/只补未处理/保留旧失败与历史operation回归单独count1通过。
- Node24.22.1，固定主候选`sha256:ff2ee7e46a9358f006d74e759f82562267add41d5ef0bdd5366fb0d0dcac7dd7`
  本地native arm64增量构建通过；AI仍`3631b432`。未更换权重、92类policy、tag profile或任何模型身份。
- 真实权重合成Docker `fp-multilingual-42e36efa`通过：独立tag PNG换代、非空人工中文同义优先/删除抑制、AI停止再启动
  同operation恢复、原件hash、语义/人物隔离、双后台并发前台中文搜索、重启及database-check。
  AI peak触及4 GiB测试预算，故补充memory.events断言完整重跑，而非据容器OOMKilled=false推断子进程从未OOM。
- 最终`fp-multilingual-80c22c0c`同一固定主镜像完整重跑通过，主peak158044160 B、AI peak4294897664 B，
  memory.events low/high/max/oom/oom_kill/oom_group_kill均0。测试预算与原部署不限内存选择分开，不关闭默认小模型/长期/跨架构/质量/发行门禁。
  两套合成容器/网络按专属project清理，临时数据/scratch保留；未以真实原件作为测试fixture。

## 6010备份、更新与补齐（2026-10-24 19:51～19:54 +08）

主应用19:51:36正常停止，exit0/OOMfalse；当时四个旧operation均终态，没有活动标签任务。
完整停写备份`/Volumes/extended/foliopath/backup-6010-tag-recovery-20261024-awycAw`（700）：
data37376文件/38404条目/1093398242 B、私有ai.env和三份Compose逐文件核验；manifest
`476877e563ac4ee11b77d1de20d86d4877b265d4de420cd914e1d60f34fadc24`。
备份DB SHA256 `9465e81289eb23a006e8cfe6e3a12925f196e669b2ec694b6dd0ebf96d2245ac`；
Linux独立验证副本`tag-recovery-validation-20261024-hAg9EC`的database-check通过，原备份保持不动，未复制在线DB。
before/after/resumed/verification.json记录真实结果与保护摘要，无凭据内容进入仓库。

仅替换ai.env的FOLIOPATH_IMAGE为上述ff2ee7e4，其他env逐字和三份Compose不变；主19:52:33启动后healthy、database-check通过。
AI镜像及StartedAt仍19:30:32，这次部署未重启AI。主/AI RAM/swap=0；同一只读/library和原data挂载保留。
部署前后所有run逐项未变、13组保护摘要完全一致（含22365语义向量）；tag身份仍
`cc65dcdd3dd9b03697dd07ace62e11900d69a804241968de8b2724dae7d9ec3d`，未激活新代次或153类。

通过原6010有效管理员会话，仅对8k/pixiv的自动标签分别关闭后重新开启；带修订保护的设置清除阻止，由原missing调度各建一次：

| 库 | 补齐operation | 19:54快照 |
| --- | --- | --- |
| 8k/lib_3 | `autotag_fb76d5fda162cfa81fdda0b1d3bd866a` | running/attempt1，checkpoint7577，completed8，error=null；累计ready642/failed7 |
| pixiv/lib_4 | `autotag_79f02ccb7186b1404737f9faac66d21b` | queued/attempt0，checkpoint0，completed0，error=null |

不是点击all/重新识别，也没有直接修改任务、claim、run或旧终态；coser/4k未动。
8k新增2项failed为先前未处理的7574/7576两张0 B源文件；用户要求跳过表示不修复/不伪造成功/不循环重试，
不是删除失败记录。旧model_unavailable两operation保留历史；新任务是missing续补，不把8个新处理说成全库完成。
22张PNG逐图当前身份ready/源指纹及实际标签数保持（21张有标签、698无候选），13组保护摘要再次一致。
补齐剩余仍运行/排队；后台继续跟进，终态后记录真实总量并停止跟进。

应用工具已确认更新既有ID6010 heartbeat为ACTIVE，沿用每30分钟及同聊天，跟进上述两个新missing operation，
不再拿旧终态或旧主镜像作为当前部署依据；没有另建重复定时任务。

## 后台验收快照（非终态）

| 时间（+08） | 8k原补齐operation | pixiv原补齐operation | 保留核验 |
| --- | --- | --- | --- |
| 2026-10-24 19:57 | running/attempt1，completed86，checkpoint7655，error=null；累计ready720/failed7、未处理65 | queued/attempt0，completed0，checkpoint0，error=null；待补999 | 22PNG均当前身份ready/current_source=1，21张有标签、698为0；13组保护摘要与本次备份一致 |
| 2026-10-24 20:28 | succeeded/attempt1，completed=total=151，checkpoint7720，error=null；累计ready785/failed7、未处理0 | running/attempt1，completed468，checkpoint8188，error=null；ready468/failed0、未处理531 | 22PNG当前身份ready/源指纹与标签结果不变；13组保护摘要与本次备份一致 |
| 2026-10-24 20:29 | succeeded/attempt1，151/151、checkpoint7720、error=null；785/7不变 | running/attempt1，completed494，checkpoint8214，error=null；ready494/failed0、未处理505 | 22PNG当前身份ready/current_source=1，13组保护摘要与本次备份一致 |

20:29例行核对未出现新的model_unavailable；8k终态已在用户进度查询中说明，pixiv继续正常推进。
双容器healthy/restart0/OOMfalse、StartedAt未变。原容器Linux锁域有界只读审计、保护核对通过，临时工具已移除；
仅文档和`git diff --check`，未重启、部署或重提任务，继续安静跟进pixiv终态。

20:28用户进度查询：8k剩余151项已全部处理结束（新增149成功、2项已知空源失败），全库785成功/7源失败；
pixiv已成功468/999（46.8%），还在处理，不能宣称四库全部结束。当前新任务没有model_unavailable，
旧两个failed/model_unavailable operation仍按要求保留历史，不冒充新错误或清除记录。
主/AI镜像及StartedAt均不变、healthy/restart0/OOMfalse；13组保护摘要核对通过，10项源失败仍均匹配当前源指纹。
原容器Linux锁域有界只读审计通过，临时工具已移除；本轮仅更新文档与`git diff --check`，未重启、部署或重复提交。

19:57新任务正常推进，10个当前身份failed均为已明确暂不修复的既有源问题，不把旧model_unavailable历史operation当新错误。
主ff2ee7e4/AI3631b432、StartedAt、RAM/swap=0保持部署后值，双healthy/restart0/OOMfalse。
原主容器Linux锁域15秒有界只读审计、保护核对通过，临时工具已移除；本轮仅文档与`git diff --check`，
未部署、重启、重提、复制在线DB或改原件/人工整理。两个新补齐任务尚未全部终态，继续安静跟进。

## 最终验收（2026-10-24 21:08 +08）

8k补齐operation `autotag_fb76d5fda162cfa81fdda0b1d3bd866a`于20:01:51结束，succeeded/attempt1、completed=total=151、checkpoint7720、error=null；
pixiv补齐operation `autotag_79f02ccb7186b1404737f9faac66d21b`于20:57:34结束，succeeded/attempt1、completed=total=999、checkpoint8719、error=null。
本次1150项续补真实新增1148个ready、2个已知空源failed；原已完成7550项未清除，旧两条model_unavailable终态历史保留。

终态只读审计新增有界current_coverage投影，将asset与run按library/asset联结，逐项绑定**当前源指纹和当前标签身份**，不是只汇总历史run：

| 媒体库 | 可处理图片/动图 | 当前成功 | 当前源失败 | 未处理/旧身份/过期源/stale |
| --- | --- | --- | --- | --- |
| coser/lib_1 | 6202 | 6200 | 2 | 0 |
| 4k/lib_2 | 707 | 706 | 1 | 0 |
| 8k/lib_3 | 792 | 785 | 7 | 0 |
| pixiv/lib_4 | 999 | 999 | 0 | 0 |
| 合计 | 8700 | 8690 | 10 | 0 |

两个新任务没有model_unavailable或其他任务级错误；全局活动AI operation列表为空。
主容器19:52:33以来有界日志检查未找到WARN/ERROR/panic/fatal，AI同期有界日志无新增报错；不据此补造历史失败的底层原因，
也不宣称未来不会失败或所有服务瞬时探测均从未波动。当前双healthy/restart0/OOMfalse，镜像/StartedAt/RAM/swap=0未变；
AI memory.events low/high/max/oom/oom_kill/oom_group_kill均0。容器内`/app/foliopath database-check`通过。

22张PNG逐项仍为cc65dcdd完整新身份、ready、current_source=1，实际AI标签数与18:26验收表逐项一致（21张有标签，698为0候选）。
与本次备份before.json的13组人工/抑制/收藏/人物/全部22365语义向量行数和SHA256逐项相同；语义向量摘要仍
`5feebf36c0c47eada2b99a013d985495c86593f5f89b271272bf2e23ef2d862f`。
真实人工/抑制起始0不冒充非空保护测试，非空证据来自前述合成Docker。终态完整只读结果已保存备份目录的`terminal-audit.json`。

剩余10项均匹配当前源指纹并按用户要求暂不修复或再次重试：

- lib_1/2596：内容格式与JPEG标记不符，需在应用外纠正源格式/标记；FolioPath未改名或改写。
- lib_1/2715：截断JPEG，需完整有效原图。
- lib_2/6669、lib_3/7374：非空无效PNG，需有效源替换/外部纠正；现有证据不能细断具体损坏字节。
- lib_3/7038、7387、7449、7510、7574、7576：6个0 B空文件，需恢复实际内容。

“本轮补齐结束”表示所有8700项有当前身份/当前源终态记录，并非把这10项标成功、修复源文件或关闭AI全功能/质量/容量/发行门禁。
本轮未新增部署/重启/提交/清理，不连接server222，不复制在线DB；仅在原主Linux锁域内15秒有界mode=ro/query_only审计，临时工具已移除。
按约停止并删除6010定时跟进，删除工具结果在成功后另记；最终文档执行`git diff --check`，未重复宣称新的代码回归。

应用工具已返回`automationId=6010`、`mode=delete`、`deleteStatus=deleted`，本次heartbeat确认删除，不再继续周期轮询。
