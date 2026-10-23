# FIX-2026-10-23：版本化的大 PNG 语义输入

- 范围：POST-MVP-5 / CR-2026-032 / FR-INT-002/003/009；Owner：semantic/imagevips/app，实施 Codex。
- ADR：[0028](../adr/0028-versioned-png-semantic-input.md)；根因：[13 张 PNG 输入限制](INV-2026-10-23-semantic-png-limits.md)。
- 状态：**代码与本地合成 Docker 联验通过；原 6010 尚未部署本修复，13 张尚未实际重试成功**。
- 用户已明确同意“当前任务结束后再切换”，包括完整备份后重算已启用库的图片语义索引。不是立即打断任务。

## 实现与兼容性

部署 JSON 增加可选 `image_preprocessing: "png-v4"`。缺省保持原 v3；未知值拒绝启动。
仅扩展语义 PNG 到 64 MiB / 40,000,000 像素，公共边长上限不变；最多 1 MiB 头校验后才 native load。
Linux 从安全已打开的 FD 解码，非文件流保留精确读取、取消和连续 100 次空读预算。
复用进程级唯一大图准入，关闭 ImageRef/thread 并回收空闲内存后再放行；不增加并发或重新限制原 6010 内存。
输出仍为首帧/autorotate/sRGB/去 alpha/uchar/Lanczos3/长边 1024/strip PNG，输出最多 8 MiB。

输入接受范围也属于身份合同，故新 profile 创建新语义代次。旧向量不删除、不复制到新代次、不重贴身份；
新代次覆盖从 0 开始。人物与自动标签身份、人工整理、移除抑制均不变。
**自动标签仍使用旧 v3 输入，因此本修复不关闭标签大 PNG 失败。153 类切换也不在本次授权范围。**
没有新 API、migration 或 UI 控件；既有设置/任务/搜索消费者和原型不变，不增加“已成功修复”假状态。

## 固定候选与本地验证

- 原生 Linux arm64 主候选：`sha256:5bcc3fef829d652be9e210905806ad774c1003174084f5287fb36e8caa9e8d9f`。
  `spikes/automatic-tags/Dockerfile.localized-app` 增量构建；Web 阶段沿用缓存，本轮没有前端修改。
- AI 不变：`sha256:3631b43219c74bc4e18df503d53e144e9bc86f80bc9490efa1d1e65d356ea903`。
- native builder：`sha256:41cdae585410a8ad7fa5dc18d8f0940c3825343233ee3973a189fe903bc428b3`。

实际通过：

1. `go test -count=1 ./cmd/... ./internal/... ./tests/contract/... ./tests/architecture/...`，Go 1.26.5 / CGO=0。
   定向 app/semantic/autotag `-race`、上述生产/合同/架构/集成及 `spikes/semantic-png` 的 `go vet` 通过。
2. `make fmt-check arch-check generate-check`、扩展盘 TMPDIR 的 `make test-integration` 与 `make test-e2e` 通过。
   后者为真实应用启动/认证拒绝/只读挂载/两次启停/迁移与原件 sentinel smoke，不是浏览器或发行镜像完整回归。
3. Linux `-tags=libvips` 完整 imagevips 测试：**36 项通过，2 项 progressive JPEG 因未显式提供 fixture 跳过**。
   1 CPU / 1 GiB / 无 swap；cgroup peak **347,058,176 B（约 331 MiB）**，oom/oom_kill=0，exit=0。
   新测试覆盖真实编码 50,344,760 B / 16,773,120 像素 PNG，3 次 FD + buffer 输出相同、原件 hash 不变，
   40 MP 边界、字节/像素越界、有效但超预算的 palette 头、截断/伪造输入、alpha/16 位/灰度、
   共享准入等待取消且不额外读 caller 流、100 次空读，以及 native 引用关闭。
4. `tests/release/multilingual_semantic_smoke.mjs` 的 `FOLIOPATH_TEST_PNG_PROFILE=1` 真实 XLM 模型联验通过。
   独立项目 `fp-multilingual-78bb3c19`，合成数据目录 `/Volumes/extended/foliopath/fp-multilingual-78bb3c19-omT7N6`。
   原 v3 为 2 成功 / 3 失败；切 v4 后新代次先为 0，再达到 3 成功 / 2 坏源失败；旧代次仍 retired 且保留 2 个向量。
   “人物 / 蓝紫色背景 / car / 车”均从实际语义 API 返回包含新大 PNG 的结果；这是功能链路，不是检索准确率。
   人物 3 成功 / 2 坏源失败；旧标签仍 2 成功 / 3 失败，证明未暗改标签输入；人工中文标签保持 manual。
   两类后台任务并行时前台中文查询、重启持久化、源 hash、同容器 database-check 通过；自动重启 0、OOMKilled=false。
   主应用峰值 **260,370,432 B（约 248 MiB）**；AI **4,135,108,608 B（约 3.85 GiB）**，测试预算分别 1/4 GiB。
   该 AI 峰值仍高，**不是默认小模型内存预算、长期或容量门禁通过**。

首轮联验通过主要功能后，在新增峰值采集中调用主镜像不存在的 `sh` 而失败；修正为仅在合成 owner
运行只读 Go 探针后从头重跑通过。未放宽产品断言。两次临时项目的容器/网络已移除，合成数据和 scratch 留存可复核。
探针 `spikes/semantic-png/audit` 只在容器中读取本地 generation 计数或 cgroup，不能从宿主执行以打开活跃 DB。
两只本轮已退出的 native 测试容器也已移除；测试二进制、合成证据目录及候选镜像保留，原 6010 未动。

没有执行无限定 `make test` / `make lint`，其包含另行维护的 native spikes；上面列明实际生产包替代范围。
未重跑完整浏览器矩阵、amd64、质量、大容量、完整发行或原库 13 张验收；不能引用旧证据冒充本候选全部通过。

## 原 6010 只读核对（20:00 +08:00）

主镜像仍 `81887939`，AI `3631b432`；两个 StartedAt 均未变化、healthy、restart=0、OOM=false、RAM/swap=0。
容器内 `database-check` 通过。未修改部署、未重启、未提交任务、未改原件/人工整理。
只读 Linux 锁域快照的临时工具已移除。

| 库 | 当前语义成功 / eligible | 失败 | 原 operation 状态 |
| --- | ---: | ---: | --- |
| coser / lib_1 | 1379 / 6202 | 13 | running，已处理 1392，checkpoint=1404 |
| 4k / lib_2 | 368 / 707 | 0 | queued，保留原断点 |
| 8k / lib_3 | 785 / 792 | 7 | succeeded，但有源错误 |
| pixiv / lib_4 | 999 / 999 | 0 | succeeded |

coser 旧 92 类标签累计 ready=4407、failed=15；原 operation 已处理3538、checkpoint=4439、仍 running。
语义和标签 operation 级 error 均为空不代表逐图无错误。人物此前 41 个源错误与此次 PNG 输入阻挡分开记录。

## 已授权后续操作与停止条件

已建立本聊天每 30 分钟一次的后台跟进 `6010-png`（ACTIVE），按以下前置条件推进，不立即切换。
这类本地跟进需要机器与应用保持运行，届时的实际执行与授权边界以工具结果和当前状态为准。

1. 仅跟进本地 `foliopath-ai-original-review`：等原有排队/运行中分析任务结束，不重复提交、不强制取消。
   持续排队不是阻塞；有新任务级错误才诊断。使用现有登录，失效请用户登录，不重设凭据。
2. 记录逐库终态、原数据/人工关系摘要及部署配置；主应用正常停机并确认退出后，完整备份整个 `/app/data`
   和私有配置到新的独立目录；必须核验备份完整/磁盘余量，再执行切换。
3. 核对本候选不可变镜像仍在，仅改主镜像与 JSON 的 `image_preprocessing`；保留原 identity/face/tag、AI 镜像、
   单一 `/library:ro`、数据目录、base + unlimited-memory + multilingual 三份 Compose 覆盖，不重启 AI。
4. 健康/容器内完整性/人工数据/新空代次通过后，为已启用的四库各提交一次新代次图片 semantic `missing`，
   记录 operation ID 和代次以防重复。**不是原代次 mode=all 或清除。**原图/人物/人工标签/旧向量均保留。
   视频与语义建议也绑定代次；切换前若发现非空的视频/建议结果，应先报告影响，不能擅自扩到视频重算或改写候选。
5. 验证 13 张原 PNG 真实结果、其他新失败逐项分类，并持续记录新代次进度；不得把合成通过或旧代次成功数当原库完成。
   不能仅删除字段就宣称恢复旧活动代次；回退需匹配旧镜像/配置与停写全备份，任何切换后新增人工整理需先保全。
6. 四库新任务到终态且剩余错误明确后，记录终态/残余源恢复要求，通知用户并停止该跟进。
   不连接 server222，不启用 153 类、不清除、不开模型商店、不改原件、不提交推送，不扩展质量/发行验收范围。

## 继续核对（2026-10-23 20:19–20:23 +08:00）

未达到“当前任务结束”的切换前置条件，故本轮没有停机、备份活动数据库、部署或重复提交。

- 20:19 coser 语义成功 **1587/6202**、失败仍 13；同 operation 已处理1600、checkpoint=1612。
  相比20:00成功增加208，没有新失败；4k仍queued 368/707，8k与pixiv终态不变。
- coser旧标签ready=4605、failed=15；同任务已处理3736、checkpoint=4637，比前次增加198。
  后续预检中处理量继续到3776，证明仍推进；不能把后续处理量混作此前成功数。
- 全局active operation仅3个：coser标签running、coser语义running、4k语义queued，任务级error均null。
  四库semantic开关均enabled；没有另外隐含的人物/视频/清理任务。
- 全部和当前代次的semantic video frame均0；旧ai_tag_suggestions及词表向量均0。
  目前未发现额外的视频/建议迁移范围，但正式切换前仍须重查，不能把此快照当未来保证。
- 人工标签关联、收藏、people、人工face anchors、tag suppressions此时均0；只是只读基线，不是执行清理。
  正式备份前重取数据摘要，保留届时新增人工整理。
- 原主/AI镜像、StartedAt、RAM/swap=0、3份Compose覆盖及`/library:ro`不变；healthy、restart=0、OOM=false。
  同容器database-check通过；主应用最近25分钟完整日志未出现WARN/ERROR/failed/panic匹配。
- 候选`5bcc3fef`仍存在、arm64；扩展盘约79GiB可用。尚未创建本次切换备份，不能把空间检查当备份成功。
- 两个临时只读诊断程序均只在主容器Linux锁域运行（query_only、15秒单快照），运行后从data移除。
  原媒体未打开或改写；没有宿主SQLite访问。既有`6010-png`跟进仍ACTIVE，每30分钟一次，未创建重复跟进。

本轮仅运行状态/切换前检查与文档更新，`git diff --check`通过；没有新的代码修改或重跑全套测试。
此前36项native及真实模型合成Docker结果仍属于候选验证，不把它们冒称本轮原库13张已通过。

## 后台跟进（2026-10-23 20:49 +08:00）

- 原 v3 代次仍为 `aig_j3AvNPmoeq-giuIydLEti_VK`。coser 同一语义 operation 仍 running，
  成功1907/6202、失败13、已处理1920、checkpoint=1934；4k 同一 operation 仍 queued 368/707。
  8k 保持785成功/7失败，pixiv保持999成功/0失败，两者原 operation 已终态。
- coser 原92类标签累计ready=4923、failed=15；同一任务running、已处理4054、checkpoint=4955。
  相比20:19语义成功增加320、标签成功增加318，失败数未增加；任务级error均null。
- 主/AI镜像、StartedAt、RAM/swap=0不变，均healthy、restart=0、OOM=false。
  本轮未重复执行database-check，最近通过证据仍为20:23；未新增程序修复或重跑测试。
- 查询仅在原主容器Linux锁域执行有界只读快照，临时诊断二进制已移除。
  未停机、未复制活动数据库、未切换配置/镜像、未提交新任务，也未访问原始媒体或修改人工整理。

原任务尚未全部结束，继续保持 `6010-png` 跟进；新png-v4代次和13张原PNG验收仍未开始。

## 后台跟进（2026-10-23 21:16 +08:00）

同一原v3代次与operation继续推进：coser语义成功2179/6202、失败13、已处理2192、
checkpoint=2207，仍running；4k仍queued 368/707，8k/pixiv终态与逐图计数不变。
coser旧标签累计ready=5201、failed=15；原任务已处理4332、checkpoint=5233，仍running。
相比20:49语义成功增加272、标签成功增加278，没有增加失败或出现任务级error。

双容器仍healthy、restart=0、OOM=false，原镜像/StartedAt和RAM/swap=0不变。
本轮只做原主容器内有界只读进度查询和Docker状态检查；诊断二进制已移除。
未重复执行完整性检查或功能回归，未停机、备份活动数据库、部署、重算或修改原件/人工整理。
切换条件仍未满足，保持后台跟进，不将原v3的新增成功数计作png-v4验收。

## 后续只读轮询

以下均为原v3代次、原operation，非新png-v4结果；未满足全部任务终态条件时不切换、不重复提交。

| 时间（+08:00） | coser语义成功/失败；已处理；checkpoint | coser旧标签ready/failed；本任务已处理；checkpoint | 其他库及运行状态 |
| --- | --- | --- | --- |
| 2026-10-23 21:47 | 2483/13；2496；2511，running | 5511/15；4642；5543，running | 4k queued 368/707；8k、pixiv计数与终态不变；任务级error均null |
| 2026-10-23 22:18 | 2801/15；2816；2833，running | 5825/15；4956；5858，running | 4k queued 368/707；8k、pixiv计数与终态不变；任务级error均null |

21:47 主/AI均healthy、restart=0、OOM=false，镜像/StartedAt及RAM/swap=0不变。
有界只读探针仅在原主容器Linux锁域执行，已移除临时二进制；没有原件访问、人工整理变更、
数据库备份或部署。失败数未增长；本轮未重跑database-check/功能回归，文档`git diff --check`通过。

22:18 coser语义相较21:47多出2张失败：`ast_2596`（21:56已告知用户）及本轮新遇到的`ast_2715`。
两者均匹配当前源指纹的既有人物失败，区别于原13张PNG限制。此前[逐图源诊断](FIX-2026-10-23-face-source-diagnosis.md)
已实测：2596的`002.jpg`内容为PNG但标记为JPEG；2715的`076.jpg`是截断JPEG，native与标准解码均报提前结束。
本轮没有重新解码原件，也不把历史诊断伪写成本轮语义runtime错误码；这些已知源问题需单独恢复，
不能纳入png-v4的13张验收或宣称提高内存即可修复。

22:18双容器healthy、restart=0、OOM=false，镜像/StartedAt、RAM/swap=0不变；
同主容器`database-check`复查通过。有界只读探针已移除，未停机/备份/部署/重提，未修改原件或人工数据。
切换条件仍未满足；本轮无代码变化或功能回归，文档`git diff --check`通过。
