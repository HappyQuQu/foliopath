# 视频源变化与新增候选（2026-09-15）

CR-2026-030 / POST-MVP-5 / FR-INT-006/007/015/016，Owner SQLite。
复用扫描原事务和 completeCurrentVideoPlan 合同，不新增 API/迁移。

新增视频时增加已有视频进度的 eligible；重复 upsert 不重复增加。
源指纹变化时，仅针对曾有该源帧的代次，复用候选查询的完整当前计划谓词，
核对当前有效视频数量，将失效 ready 差额转为 stale，推进 revision。
不按 4/10 帧分别扣减，不删除旧帧，也不改变图片设置状态。
保留旧计划本身并不能证明它仍计入 ready，因此没有直接按“存在旧帧”扣减。

本地 Docker SQLite race 命令：

```sh
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
 '/out/sqlite.test -test.run "TestScanInvalidates|TestVideo" -test.v -test.timeout=3m'
```

新测试从一个完整 4 帧计划开始，经真实扫描仓储更新源，确认 1 ready→1 stale，
重复扫描/继续改变指纹不重复扣减；新增视频增加 eligible，失败扫描保留计数及旧帧。
既有完整计划、部分计划排除、任务提交/恢复/降级测试纳入同轮。
首轮通过后改为复用完整计划谓词，重新构建并复跑全部 PASS / exit 0。
make arch-check 与 git diff --check 通过；未运行完整七项仓库检查。

边界：视频删除尚未实现精确计数；尤其降级/失败只有汇总，不能猜测被删资产的状态。
本轮未重跑真实 AI 主进程/浏览器，未验证大规模扫描性能、故事板缓存淘汰或完整并发矩阵。
核对有效计划涉及库内查询，仍须容量验收；未宣称 100k 容量通过。
无真实媒体或 server222 访问。
