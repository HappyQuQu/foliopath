# 图片回填：取消优先于后续终态提交

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite；复用既有取消与领取版本校验，不改变 API 或数据结构。

## 复现与修复

先增加真实 SQLite 回归，再在本地 Docker 中运行：两个子场景均失败。
已接受取消后尝试成功收尾返回 semantic job conflict；尝试失败收尾则把任务
改成 failed/model_unavailable。均不是正确的取消终态。

FinishSemanticBackfill 现在在同一终态写事务内读取已持久化的 operation.state，
若为 cancelling，则将成功/失败尝试收尾为 cancelled。取消尚未被接受时仍遵守
原完成数量检查、取消检查；旧领取版本与已结束任务仍拒绝写入。
不通过修改 context 或伪造前端状态规避事务规则。

新增测试验证成功/失败两种收尾尝试、旧领取版本拒写、job/operation 同为取消终态、
租约清除、进度快照不变、再次收尾拒绝。这里测试的是取消先提交、终态事务后执行
的确定性顺序，未覆盖实际模型/处理器最终批次与取消的所有并发交错。

## 本地验证

修复后回归镜像 manifest list：
`sha256:ff914c1f3e4663fb3a367ae91e251d93c00f1e7339b2aa7a6918b463e5509e1b`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestSemanticBackfill|TestImageBackfillCancellationBeforeAdmission" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m'
make arch-check
```

使用合成夹具、临时 SQLite 和 race 测试二进制，无真实媒体或 server222。
修复后 SQLite 10 个顶层测试（新增终态测试含 2 子场景）、semantic 全包及架构检查通过。
未重建实际运行镜像，未执行完整七项检查；清理批次交错、处理器/实际运行竞态矩阵、
人物生产接入、压力容量及发行仍待完成。
