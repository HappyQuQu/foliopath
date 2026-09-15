# 视频移除覆盖率（2026-09-15）

CR-2026-030 / POST-MVP-5 / FR-INT-006/007/015/016，Owner SQLite。
OpenAPI 语义搜索说明同步更新，生成客户端重新生成，无端点或字段变更。

成功扫描确认移除视频时，在资产/帧级联删除前复用 completeCurrentVideoPlan，
按剩余视频核对 eligible 和当前完整计划。ready 不高于旧计数，避免全量重建时
把保留的旧帧提前计入完成。失败扫描不执行此步骤。

现有数据只有 degraded/failed/stale 汇总，缺少逐资产结果归属。因此删除后不猜测
哪种状态被移除：这些旧汇总归零，对应未能确认的剩余项回到 pending 差额。
历史任务结果不变；新的缺失项回填重新分类。这是保守重算，不是新增精确失败账本。

## 已执行

- 本地 remote-ml-regression Docker 镜像重新构建成功。
- SQLite race：TestScanInvalidates / TestVideo / TestFailedScanPreserves 全部 PASS。
- 新案例覆盖混合降级/失败、失败扫描保留、成功扫描后 pending、全部删除及重复空扫描；
  另验证保留有效 4 帧计划，以及 ready=0 的全量重建不提前获得完成计数。
- Docker OpenAPI 合同测试 PASS，使用只读绑定的最新 api 目录；客户端生成与生成检查通过。
- make arch-check 和 git diff --check 通过。

本轮没有重新运行真实主进程/AI 联调或浏览器 E2E，没有验证十万级性能或完整七项检查。
未访问 server222 或真实媒体；测试仅操作容器临时 SQLite 夹具。

后续：真实主进程删除→pending→回填闭环、UI 展示、并发全量回填及容量验证。
