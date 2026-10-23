# CR-2026-032：按 Immich 配置接入多语言语义搜索模型

日期：2026-10-23；Owner inference/remoteml/app/semantic；POST-MVP-5。
范围：FR-INT-002/009/017/019、NFR-INT-007；决策 ADR-0025。
状态：模型与并发修复的真实权重本地 Docker 联验通过，6010 已切换；四库新索引运行/排队，尚未全量完成。

18:21 最新只读快照：pixiv 999/999 成功；8k 785/792 成功、7 既有源失败；coser 416/6202
运行中，4k 368/707 排队。原 operation/身份不变，未为验证反复重启或提交任务。
本轮生产搜索模式提示候选、实际验证与部署边界见[续验记录](FIX-2026-10-23-search-mode-guidance.md)，
不以新 UI/合成数据通过替代中文检索质量或全库索引完成。

## 要求与合同

用户截图及提供的 Immich JSON 均指定 `XLM-Roberta-Large-Vit-B-16Plus`，
本地却仍固定 ViT-B-32__openai。本次只读查看 server222 的旧缓存，仅有 English 模型，
没有在远程测试或修改服务。配置选择不等于那份旧缓存目录已经存在对应文件。
本地 coser 中文“车”和英文“car”均无结果、覆盖 0/0；不是单纯中文问题，须建立当前代次索引。

固定来源 `immich-app/XLM-Roberta-Large-Vit-B-16Plus`，revision
`f8857d09bd30188c40214543d2181f732e64eab4`，179 文件/2,729,566,266 字节。
逐个 LFS SHA256 或 Git blob OID 验证后生成部署 SHA256 清单；权重不进入 Git/镜像或公开发布。
主应用仅接受 English 512 或 XLM 640 的明确组合，不按维度猜模型；不重标旧 embedding。
新语义覆盖层共用原 scheduler，face/tag 仍执行原字节 worker 和身份；私有快照遵循 ADR-0025，
新模式单独 disk scratch，不改变旧缓存约束、26 秒推理期限或默认小模型资源验收。
公开 API/生产页面形状不变，模型仍从部署提供；没有增加假模型选择器，稳定 Story 的旧模型只是合成场景。
OCR、标签语言、Immich 其他配置不在本次切换范围。

## 实际执行证据

- PASS：fmt-check、arch-check、generate-check；全部 internal/cmd 生产 Go 包及 contract；remoteml race、remoteml/app vet。
- PASS：本地 Docker Python 29 项 multilingual/service/scheduler/face/tag。初次无效 tag fixture 已修正后重跑，未删断言。
- 首次 Compose 脚本在创建服务前因 mem_limit JSON 字符串断言失败；数值比较修正后完整重跑。
- PASS：真实权重 `fp-multilingual-0998dcac`，主应用 1 CPU/1 GiB，AI 3 CPU/4 GiB、无额外 swap，
  专用 disk scratch。2 张合成图片成功/1 个损坏负例，实际 640 维索引、幂等与库隔离通过。
  “人物”排序 people.png 在前，“蓝紫色背景”排序 blue.png 在前；“车”与“car”均可查询向量结果。
  后两词仅验证查询链路，合成集中没有真实汽车，不冒称汽车召回或全库中文质量验收。
- PASS：相同测试中的真实 face/tag、人工中文标签保留、显式主应用重启后索引和标签保留、源 hash、容器内 database-check。
  显式重启前无自动重启；AI memory.peak=2,879,848,448 B，不拿这项当默认 2304 MiB 预算通过。
- 原 AI 与新 AI 的 legacy image_identity 输出完全相同（be1d1baa… / e6d671f9…），没有给 face/tag 另造代次。

第一轮受测主镜像 `sha256:75cb1c9ea9940b0941feb2a5e8ee2d6326ad94ea1e0d5b105f183222792e3be9`；
AI `sha256:3631b43219c74bc4e18df503d53e144e9bc86f80bc9490efa1d1e65d356ea903`。
新语义 runtime=81ec3de5…，cache=858e52d6…；完整摘要由镜像 `python -m multilingual_contract` 独立读取。
主应用采用固定本地 builder 的增量构建，保留原前端与 native runtime，复现 Dockerfile 位于 multilingual spike；
不是最新根 Dockerfile 全构建、双架构或发行批准。没有运行完整 lint/integration/E2E/容量/质量矩阵。
合成数据：`/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-multilingual-0998dcac-fYhgXz`。

## 6010 切换与并发补充

切换前自动标签原任务仍在运行（13:45 已处理 827），不能丢失断点或重新建同一个任务。
第一阶段停止主应用确认 exited 后，完整备份至
`/Volumes/extended/foliopath/backup-6010-multilingual-20261023-QF5La9`，复制结束后才更新。
仅对 detached immutable 副本检查 quick_check=ok、外键无行；8,719 媒体、10,623 条历代人脸结果、20,788 条标签关系保留，
人工标签/people/删除抑制均为 0；原标签任务 running、attempt=1、checkpoint=896。
更新为第一轮固定主/AI 镜像与独立复制缓存，保留原 face/tag JSON，叠加 unlimited-memory 与 multilingual 覆盖；
两服务 healthy/restart=0、RAM/swap=0，主容器内 database-check passed。

此次先同时启动两服务，原 coser 标签任务在 AI 尚未就绪时恢复而终止，留下 model_unavailable 的真实失败记录
（已尝试 884，871 成功/13 源输入失败）。没有把它改为成功、删除历史或回滚数据库。
经正式界面将自动标签关闭/重新启用以解除既有失败暂停，自动 missing 只接续未有运行记录的图片，
不全量重做、不清掉 871 张成功或重试 13 张未恢复输入；之后数量持续递增到 >1,000。
已将安全更新顺序修正为 AI healthy 后才启动主应用，无需改核心依赖/凭据。

依次从正式界面提交 4k 707、8k 792、pixiv 999、coser 6202 的新空间 missing，共 8,700 张；
不是此前人脸 missing 的 8,493，也不是已有 512 维结果被重标。原四库人物终态保留。

现库一边索引/标签、一边查询时又复现 internal_error（req_Snh1…，GET 500/0ms）：
两种后台处理占满 Go transport 的两个槽，查询立即返回 ErrBusy，旧 API 路径将其当内部错误。
先只让文本等待的候选虽然查询通过，但并发 all 测试中后台图片还会立即占槽失败，
在隔离合成目录的 Linux 只读统计确认为 completed=0/failed=3；未将此轮完整测试标通过或上线。
最终所有语义 image/text 均在原 30 秒总期限内等待同一两个 transport 槽；API 一次只准入一个前台查询、
后台 worker 仍有界，不增加 goroutine/模型并发，face/tag 原忙碌退让策略不改。
Go race 验证第三个请求超时不占槽、取消与释放后继续，没有通过删除容量断言规避问题。

最终 Docker `fp-multilingual-e50bee99` 从头完整复验通过：原顺序矩阵 + 两后台能力运行期间中文查询 +
同代次 all 后仍 2 成功/1 个损坏负例 + 显式重启后索引/查询/人工标签保留。
主镜像 `sha256:1806724d1f019bec88cc4092f9630e8223849034aff3ed709c54e0f51edfea83`；AI 仍 3631b432…。
AI memory.peak=4,294,967,296 B，memory.events oom=0/oom_kill=0，自动重启 0；
虽然通过所测功能，已触及 4 GiB 上限，不宣称余量/长期/基础预算或一般容量通过。
最终合成数据留于 `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-multilingual-e50bee99-3RX1RU`。
fmt/arch/generate、全生产 Go 与 contract、remoteml race/vet 重新通过，未跑完整浏览器/发行矩阵。

只为上述已复现并发错误再次计划更新主应用；AI 保持 healthy 不重启。
先停主应用并完成完整备份：`/Volumes/extended/foliopath/backup-6010-semantic-admission-20261023-3niCXu`，
detached quick_check=ok、外键无行；8,719 媒体、10,623 人脸结果、21,785 标签关系，
coser 标签 checkpoint=1318/attempt=2、4k 语义 checkpoint=6589/attempt=2，另三库 queued/attempt=0。
再单次仅替换主应用为最终镜像，无新迁移、无新任务重提，不因当前 attempt 接近恢复上限而额外重启。
这些是计划更新，不能从新容器 restart=0 抹去历史中断或声称原 operation 从未失败。

## 更新后现库只读复核（14:53–14:54 +08:00）

新主 1806724d…、AI 3631b432… 均 healthy、当前容器自动重启 0、RAM/swap=0；
再次容器内 database-check passed。未修改原件、凭据/人工整理，没有迁移、全量清理或再次创建已提交任务。
正式页面在两个后台能力继续工作时查询 4k 库中文“车”，返回 50 个排序结果（分页大小，不是相关图片总数），
覆盖 368/707、失败 0。首行可见汽车画面及“道路,汽车,绿色树林…”图片，不再是 0/0 或 internal_error。
这是手工运行复核，不拿真实图片作自动化评测；检索尚未覆盖全库，也不把每个返回项都称为语义相关。
截图在仓库外 `/Volumes/extended/foliopath/model-experiments/multilingual-evidence-20261023/6010-chinese-car.png`。

| 库 | 原新语义 operation | 此时真实状态 |
| --- | --- | --- |
| 4k | aio_cc93bd76daf70e00683bf2371a72c40e | queued，断点/已处理 368/707 保留，任务错误为空 |
| 8k | aio_7655b053f59d69b5f542a1ca35b3a5ab | running，已处理 80/792，任务错误为空；此前覆盖快照 64，时间点不同 |
| pixiv | aio_479074a1d4d99ad663dd86f2087b1d8d | queued，0/999，任务错误为空 |
| coser | aio_08d77ae32263d0ada16a64c6117051e6 | queued，0/6202，任务错误为空 |

标签新 missing `autotag_0e6072a5243a9a0c7ed8947e2fce74ed` 继续运行，已处理 505；
原 871 张成功保留、之后持续递增，既有 13 个输入失败仍记录，没有宣称完成/清零。
旧模型 retired 索引仍保留；当前新空间为 aig_j3AvNPmoeq-giuIydLEti_VK，不能混入旧索引搜索。
后续不要为更新文档反复重启这些近恢复预算的在途任务；本轮没有创建新的定时跟进。

收尾核验项目归属后，删除 0998dcac/faf8421b/e50bee99 的 6 个临时容器、6 个网络与 3 个
可重建 scratch 卷；合成数据、固定镜像、真实权重、原实例 scratch 和两份完整备份均保留。
文档 diff 检查通过；内存/中文质量/全量索引/其他发行门禁仍按真实未完成状态保留。
