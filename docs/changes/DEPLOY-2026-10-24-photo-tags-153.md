# DEPLOY：6010的153类标签策略

**状态：2026-10-25 05:54终态复核通过。四库新身份任务全部结束，8690 ready、10已知源failed、0待处理；本次跟进6010-153已确认删除。**
10源文件按用户要求暂不修复，不标成功；153类功能落地不等于标签准确率、容量或发行门禁全部通过。

- 范围：POST-MVP-5 / CR-2026-033、FR-AUTOTAG-004/005；Owner curation/autotag/app/remoteml。
- 用户已授权：先本地Docker验证，通过后完整备份切换，保留人工标签/删除抑制，10项源错误继续暂不修复，全部补齐核对后汇报。
- 依据：ADR-0026稳定类别与独立策略、ADR-0029独立tag PNG profile、ADR-0030有界服务恢复；无新部署单元/API/数据库变化。

## 部署前基线与计划

原6010主ff2ee7e4、AI3631b432；92类/png-v4标签身份cc65dcdd四库终态8690成功/10已知源失败/0未处理。
语义及人物已保留，原始媒体只读；旧任务不复活或改写。153类候选AI52f4a40c沿用相同权重/cache，
仅tag policy/runtime身份更新，搜索640维与legacy人物身份不变，主应用保留现有恢复修复。

先验证候选执行源/153类与模型标签表、严格输出/身份/断连生命周期；合成Docker从92类＋png-v4与非空人工/抑制状态
停主全量复制备份后切换153类，验证新身份/真实新增类别、大PNG、坏源、同任务恢复与语义/人物保留。
然后核对原任务均终态，主应用正常停写，完整备份data与私有配置/三Compose逐文件验证、Linux备份database-check，
仅更换AI镜像与部署JSON tag.identity，tag.image_preprocessing=png-v4、semantic/face完整配置、原件/data挂载及不限内存覆盖保留。
先使新AI健康并精确匹配预期身份，再启原主应用；新身份由既有missing调度每个已启用库自动创建一次任务，不重复全量提交。

已知10项源文件不修复/不恢复/不改名，不标成功、不循环重试；新身份遍历可首次记录这些源失败，不引入任意排除API。
旧标签和人工整理不预先清空，旧run不能计新身份完成；最终逐库核对当前源指纹、新身份ready/failed/stale、22PNG实际结果、保护摘要。
执行结果、备份、任务ID及最终数量在实际发生后追加，不提前宣称完成或准确率/发行通过。

## 本次本地验证（部署前）

- `fp-multilingual-82d756bb`真实模型Docker联验退出0：主ff2ee7e4，旧AI3631b432切换候选52f4a40c；完整合成停写备份、92→153新身份/真实新增类别、大PNG、坏源、非空人工同义标签及删除抑制、重复重跑/重启、AI断连同operation恢复、中文搜索/语义/人物保留、源hash及database-check通过。
- 合成峰值主156667904 B、AI4108713984 B；4 GiB合成预算下memory.events max/oom/oom_kill等均0。不是原6010长期内存或标签准确率证明。
- Python photo policy/tag/scheduler/TCP disconnect/runtime manifest/multilingual合计23项通过；最初测试装载路径误选测试目录导致缺少类别文件，修正为使用镜像原模块后通过，非运行时修复。
- Go curation/sqlite/autotag/remoteml/app/contract/architecture定向测试，`make fmt-check arch-check generate-check`、Node脚本语法及`git diff --check`均执行通过。本轮未重跑全量发行检查。
- 原6010复核：四库8700项中8690 ready/10源failed/0 pending，全局无活动AI operation；13组保护摘要及22365语义向量保持。两容器healthy，未切换前主ff2ee7e4/AI3631b432，不限内存覆盖仍在。

预期153＋png-v4标签指纹为`cf6426488141185f8fe6eafea19708103e0fd47339657ff3c045f3375d32f03d`。
独立按生产身份与输入合同计算，并先反算旧配置得到cc65dcdd一致；部署后仍须核对实际落库身份。

## 原6010已安全切换（2026-10-24 21:49 +08）

- 确认无活动AI operation后正常停止主应用，exit0/OOMfalse；完整备份`/Volumes/extended/foliopath/backup-6010-photo-tags153-20261024-m1T7oV`。
  data为1093398242 B、37376文件/38404条目，逐文件SHA256及4份配置核对通过，manifest为`663e7df60925c205a928f07108fd54a90acb6a2ddff9b4279512880054e2f17b`。
  `before.json`保留13组保护摘要，`verification.json`保留校验结果；从停写备份复制出的独立Linux验证副本database-check通过。
- 仅将AI换为`sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626`并更新私有部署JSON的tag.identity。
  主镜像仍`sha256:ff2ee7e46a9358f006d74e759f82562267add41d5ef0bdd5366fb0d0dcac7dd7`；semantic/face完整配置、两个png-v4 profile、其他env行/三Compose均核对不变。
  初次脚本对env引号形式的断言不符，均在任何切换写入前停止；按现有raw JSON/单引号镜像行修正后一次完成切换，无数据库变更。
- 先启动新AI并逐端点核验新tag与原semantic640/face身份，再启动主应用；AI StartedAt21:48:25、主21:49:33。
  本次AI为换策略重建，不沿用此前“未重启AI”的表述。双服务healthy/restart0/OOMfalse，RAM/swap=0。
- 新主容器database-check通过，13组保护摘要与before完全一致，包括22365条全代次语义向量。
  原库人工/抑制起始0，真实非空保护证据来自上述合成Docker，不混淆。
- 实际新标签指纹与预期cf642648一致；旧92类结果在逐图替换前保留，不作为新身份成功数。

| 库 | 自动创建的新operation | 初次新身份快照 |
| --- | --- | --- |
| coser/lib_1 | `autotag_a30b10da19cc7c20ddb2b64cd550da54` | running，12 ready/6190 pending，attempt1/error=null |
| 4k/lib_2 | `autotag_7071da838b4522075ae0b57c18f03081` | queued，707 pending/error=null |
| 8k/lib_3 | `autotag_0d2580dc417b6c5c52be208524d4e762` | queued，792 pending/error=null |
| pixiv/lib_4 | `autotag_b5ae98af8d3b2a1bb8a3ac49ecef0f5e` | queued，999 pending/error=null |

以上为运行快照，**补齐尚未完成**。四库仅由既有missing调度各建一次，未人工重复提交。
后台`6010-153`已由应用确认ACTIVE，每30分钟跟进，正常运行/排队保持安静，终态复核并更新文档后删除。
22PNG必须按此新身份/当前源指纹再验，不能沿用92类时的标签数；10源文件继续暂不修复/不循环重试。
在线审计使用`/tmp/fp-tag-failure-20261024-H6QGrm/audit153.go`，只在原主容器Linux锁域有界只读执行，临时二进制已移除。

21:59复核：coser新身份170 ready/6032 pending，checkpoint172/attempt1/error=null；其余三库仍排队/error=null。
13组保护摘要再次一致，`resumed.json`已存本次备份目录；新AI memory.events max/oom/oom_kill均0。
未重启或重复提交，等待同一批operation终态。那10项源问题为lib_1的2596格式标记不符、2715截断JPEG，
lib_2的6669无效PNG，lib_3的7374无效PNG和7038/7387/7449/7510/7574/7576六个0 B文件；保持原件不变。

22:23后台只读复核：coser新身份618 ready/5584 pending，checkpoint620/attempt1；其余707/792/999仍排队，四个原operation均error=null。
目标PNG中602/603/606/615/619已是新身份ready/当前源，实际AI标签数9/5/4/10/7；其余17张尚为旧身份，不计本轮完成。
13组保护摘要与before一致，双服务仍原StartedAt/healthy/restart0/OOMfalse，memory.events max/oom/oom_kill均0。
主日志出现VipsJpeg损坏数据/读tile警告，未关联到具体资产，不能补造逐图原因；本次当前新身份暂无failed或任务级model_unavailable。
原10条失败仍属旧身份记录，等待新遍历按事实记录；未改原件或重试任务。只读证据保存为备份目录`heartbeat-20261024-2223.json`。

22:53后台复核：coser新身份1184 ready/5018 pending，checkpoint1196/attempt1/error=null；余三库仍按原ID排队/error=null。
前段13张PNG已全部新身份ready/当前源，实际AI标签数按ID602/603/606/615/619/621/625/626/627/629/694/698/701依次9/5/4/10/7/5/10/14/8/3/5/0/13。
698在新策略下仍为0候选，但输入处理成功；后段9张PNG仍旧身份，尚未计入本轮验收。
当前新身份failed=0，13组保护摘要与before一致，双服务healthy/StartedAt未变/restart0/OOMfalse，memory.events max/oom/oom_kill均0。
证据保存为备份目录`heartbeat-20261024-2253.json`；没有重启、重复提交或源文件改动，继续等待终态。

23:23后台复核：coser新身份1727 ready/4475 pending，checkpoint1740/attempt1，四原operation均error=null；其余三库仍排队。
前13PNG保持新身份成功，后9张尚旧身份；当前新身份无failed。13组保护摘要一致，双healthy/StartedAt未变/restart0/OOMfalse、memory.events无max/oom事件。
只读快照已存`heartbeat-20261024-2323.json`，继续等待原任务，不重复提交或部署。

23:53后台复核：coser新身份2237 ready/3965 pending，checkpoint2252/attempt1，余三库仍排队，四原operation error=null。
前13PNG保持新身份成功、后9张尚未处理；当前新身份failed=0。13组保护摘要一致，双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志未见新的任务/服务错误，快照已存`heartbeat-20261024-2353.json`；继续同一批任务，无重复提交或部署。

2026-10-25 00:19用户查询时实测：新身份共处理2703/8700（31.1%），其中2701 ready、2源failed；coser剩3499，4k707/8k792/pixiv999仍排队。
coser原operation completed2703/checkpoint2719/attempt1/error=null，其余三个原operation error=null。新failed仅2596/2715两条已知源问题，当前源指纹一致，不属于新程序错误，不重试或改原件。
前13PNG保持新身份成功，后9张尚旧身份；13组保护摘要与before一致。双healthy/StartedAt未变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0，容器内database-check通过。
最近35分钟有界日志未见model_unavailable或任务/服务ERROR；有JPEG警告，不把未保存逐图码解释成新失败。证据存`latest-user-status.json`，继续原任务至终态。

10-25 00:23后台复核：coser新身份2785 ready/2已知源failed/3415 pending，completed2787/checkpoint2803/attempt1；其余三库仍排队，四原operation error=null。
失败仍仅2596/2715，无额外新身份失败；前13PNG成功、后9张未处理。13组保护摘要一致，双healthy/StartedAt不变/restart0/OOMfalse、AI memory.events max/oom/oom_kill均0。
证据存`heartbeat-20261025-0023.json`，未重复提交、重启或改原件，继续等待终态。

10-25 00:53后台复核：coser新身份3370 ready/2已知源failed/2830 pending，completed3372/checkpoint3389/attempt1；余三库仍原ID排队，四operation error=null。
新failed仍仅2596/2715，前13PNG保持新身份成功、后9张未处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志未见任务/服务ERROR或model_unavailable。快照存`heartbeat-20261025-0053.json`；继续原批次，无重复提交或部署。

10-25 01:23后台复核：coser新身份3888 ready/2已知源failed/2312 pending，completed3890/checkpoint3907/attempt1；其余三库原ID排队，四operation error=null。
新失败仍仅2596/2715，前13PNG新身份成功、后9张待处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志无任务/服务ERROR或model_unavailable，快照存`heartbeat-20261025-0123.json`；未重复提交或部署，继续等待终态。

10-25 01:53后台复核：coser新身份4431 ready/2已知源failed/1769 pending，completed4433/checkpoint4450/attempt1；余三库原ID排队，四operation error=null。
失败仍仅2596/2715，前13PNG新身份成功、后9张待处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志无任务/服务ERROR或model_unavailable，快照存`heartbeat-20261025-0153.json`；继续同一批任务，未重启或重复提交。

10-25 02:23后台复核：coser新身份4964 ready/2已知源failed/1236 pending，completed4966/checkpoint4983/attempt1；其余三库原ID排队，四operation error=null。
失败仍仅2596/2715，前13PNG新身份成功、后9张待处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志无任务/服务ERROR或model_unavailable，快照存`heartbeat-20261025-0223.json`；继续原任务，无重复提交或部署。

10-25 02:53后台复核：coser新身份5514 ready/2已知源failed/686 pending，completed5516/checkpoint5533/attempt1；其余三库原ID排队，四operation error=null。
失败仍仅2596/2715，前13PNG新身份成功、后9张待处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志无任务/服务ERROR或model_unavailable，快照存`heartbeat-20261025-0253.json`；继续原批次，未重复提交或部署。

10-25 03:23后台复核：coser新身份6060 ready/2已知源failed/140 pending，completed6062/checkpoint6081/attempt1；其余三库仍按原ID排队，四operation error=null。
失败仍仅2596/2715，前13PNG新身份成功、后9张尚未处理；13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0。
最近30分钟有界日志无任务/服务ERROR或model_unavailable，有VIPS警告但没有新增标签failed，不补造逐图原因。快照存`heartbeat-20261025-0323.json`；继续原任务，无重复提交或部署。

10-25 03:53后台复核：coser原新任务succeeded/6202项/attempt1/error=null，新身份6200 ready/2已知源failed/0 pending。
4k原任务running，397 ready/310 pending/checkpoint6618/attempt1/error=null；8k792/pixiv999仍原ID排队/error=null。四库合计6599/8700项已处理（6597 ready/2源failed），尚未全部结束。
22PNG均为cf642648新身份ready/当前源指纹；21张有实际AI标签，698为0候选且非输入失败。
后9张ID6176/6182/6183/6184/6185/6186/6189/6190/6191的实际AI标签数依次18/20/17/5/13/11/3/17/19；前13张结果见22:53记录。
13组保护摘要与before一致，双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0，最近30分钟有界日志无新任务/服务错误。
快照存`heartbeat-20261025-0353.json`，未重启或重复提交，按用户要求等待全部终态后再汇报。

10-25 04:23后台复核：coser6200 ready/2已知源failed、4k706 ready/1已知源failed，两个原任务均succeeded/attempt1/error=null。
8k原任务running，250 ready/1已知空源failed/541 pending，completed251/checkpoint7179/attempt1/error=null；pixiv999仍原ID排队/error=null。
当前新身份失败仅2596/2715/6669/7038（均授权暂跳过的源问题），四库合计已处理7160/8700，7156 ready/4源failed，无新程序错误。
22PNG保持全部新身份ready/当前源，13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0，最近30分钟有界日志无任务/服务ERROR或model_unavailable。
证据存`heartbeat-20261025-0423.json`，继续原8k/pixiv任务，未重启、重复提交或改源。

10-25 04:53后台复核：coser6200/2、4k706/1、8k785/7（ready/源failed）三个原任务均succeeded/attempt1/error=null，当前源/新身份pending均0。
pixiv原任务running，26 ready/973 pending，completed26/checkpoint7746/attempt1/error=null；四库合计7727/8700项已处理，7717 ready/10已知源failed。
10条新身份失败与授权暂跳过清单完全一致，源指纹均当前；未发现额外输入/程序失败或model_unavailable，不循环重试那10张。
22PNG保持全部新身份ready/当前源，13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0；最近30分钟有界日志无新任务/服务错误。
快照存`heartbeat-20261025-0453.json`，继续原pixiv任务，等待全部终态后统一复核/汇报。

10-25 05:23后台复核：前三库原任务保持succeeded/attempt1/error=null，pixiv原任务running，593 ready/406 pending，checkpoint8313/attempt1/error=null。
当前新身份共处理8294/8700（8284 ready/10已知源failed），失败精确对应暂跳过清单且源指纹当前，无新增程序/任务错误。
22PNG保持全部新身份ready/当前源，13组保护摘要一致。双healthy/StartedAt不变/restart0/OOMfalse，AI memory.events max/oom/oom_kill均0，最近30分钟有界日志无新任务/服务错误。
快照存`heartbeat-20261025-0523.json`；继续原pixiv任务，未重启或重复提交。

## 最终验收（2026-10-25 05:54 +08）

仅核对当前源指纹和新身份`cf6426488141185f8fe6eafea19708103e0fd47339657ff3c045f3375d32f03d`，没有把旧92类结果混入：

| 库 | 终态时间（+08） | operation | ready | 源failed | pending |
| --- | --- | --- | ---: | ---: | ---: |
| coser/lib_1 | 03:31:28 | autotag_a30b10da19cc7c20ddb2b64cd550da54 | 6200 | 2 | 0 |
| 4k/lib_2 | 04:09:49 | autotag_7071da838b4522075ae0b57c18f03081 | 706 | 1 | 0 |
| 8k/lib_3 | 04:51:43 | autotag_0d2580dc417b6c5c52be208524d4e762 | 785 | 7 | 0 |
| pixiv/lib_4 | 05:44:37 | autotag_b5ae98af8d3b2a1bb8a3ac49ecef0f5e | 999 | 0 | 0 |
| 合计 | 全部结束 | 四个原operation均succeeded/attempt1/error=null | 8690 | 10 | 0 |

全局无活动AI operation；8700项最新run全部新身份/当前源，旧身份或stale未处理为0。
任务succeeded表示遍历结束，不把10条源失败伪装成功。旧model_unavailable历史终态保留，本轮四原任务无此错误。

22PNG再次逐张核对全部新身份ready/current_source=1；21张有实际AI标签，698为0候选但输入处理成功。
实际计数完整保存在`terminal-audit.json`，13张前段及9张后段计数分别见22:53、03:53快照；终态未发生变化。

剩余10条新身份failed的ID与授权暂跳过清单完全匹配，源指纹均当前：

- lib_1/2596格式标记不符，需外部纠正有效源；lib_1/2715截断JPEG，需恢复完整文件。
- lib_2/6669、lib_3/7374为无效PNG，需恢复可解码源。
- lib_3/7038、7387、7449、7510、7574、7576共6个0 B文件，需恢复非空有效源。

本轮没有恢复、改名、修改原件或循环重试；源问题继续暂留。日志曾有VIPS源解码警告，不补造没有持久化的逐图错误码。
最终最近30分钟有界日志无新的任务/服务ERROR或model_unavailable，未出现额外failed。

终态保护/运行核验：

- 13组人工/抑制/收藏/人物/语义表行数与SHA256均与停写备份before一致；全部22365语义向量SHA256仍为`5feebf36c0c47eada2b99a013d985495c86593f5f89b271272bf2e23ef2d862f`。
- 原库人工/抑制起始0；非空保护由真实模型合成Docker证明，未向原库添加测试整理。
- 原主容器内`/app/foliopath database-check`再次通过。双服务healthy/StartedAt仍21:48:25与21:49:33/restart0/OOMfalse，RAM/swap=0；新AI memory.events max/oom/oom_kill均0。
- 重新核对三个身份端点：tag匹配153策略、语义640与face匹配切换前配置；两个png-v4 profile、三Compose字节、单/library:ro及AI无媒体/DB挂载均符合原约束。
- 最终只读审计使用`audit153-final.go`增加4条固定operation终态时间查询，仍15秒单只读事务/有界查询，临时二进制执行后移除；没有宿主SQLite访问或运行中数据库复制。
- 完整证据保存在本次备份目录`terminal-audit.json`、`terminal-verification.json`及原before/verification；后台应用返回`6010-153 deleteStatus=deleted`，本次跟进结束。

本轮只完成153类策略部署与四库功能补齐，质量校准、容量/长期预算、跨架构及完整发行检查仍按原清单保留。
