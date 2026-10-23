# 语义清理批次提交后的取消

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite。终态决策仍归 SQLite 原事务，无 API、迁移或部署变化。

## 复现与修复

先由真实图片处理器将合成向量写入临时 SQLite，再运行真实 ClearProcessor。
测试包装器仅在首个删除批次提交后，通过正式取消方法接受取消，不取消 context，
以覆盖心跳尚未通知处理器的间隔。

修复前 Docker 两个子场景均失败：批大小 1 时任务被覆盖为 failed/internal_error，
批大小 1000（本夹具全部 3 项）时成功收尾返回 semantic clear conflict。
FinishSemanticClear 现于终态事务中将已持久化 cancelling 优先收尾为 cancelled，
不恢复已经删除的派生向量，也不绕过领取版本与终态校验。

测试要求：部分删除后剩余 2 条；最后一批后剩余 0 条；两种任务均 cancelled、
完成数量准确、租约清除，设置 disabled/degraded，操作记录同步取消，重复失败收尾
不能覆盖终态。部分批次路径仍返回下一批的原冲突错误，但用户可见终态是取消。
媒体为合成来源，本轮不读取或删除用户原始媒体。

## 本地 Docker

修复后回归镜像 manifest list：
`sha256:434bbee56dfee1a879c598f132bc79b58063f2a400889f6340480a3aac017c9e`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestSemanticClear|TestImageProcessorAcceptedCancelAtTerminalBoundary|TestSemanticBackfillAcceptedCancelWinsTerminalTransaction" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m'
make arch-check
```

测试二进制启用 race，容器无网络，仅临时 SQLite 和合成夹具。
修复后 SQLite 10 个顶层测试、semantic 全包及架构检查通过。
本轮未更新实际运行镜像，未执行全套七项检查或随机压力测试。
实际运行镜像回归、更多任务类型取消矩阵、人物生产接入、容量与发行仍待完成。
