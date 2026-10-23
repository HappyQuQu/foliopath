# 图片处理器终态边界组合回归

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite，本轮增加测试，无生产代码或合同变化。

在真实 BackfillProcessor 与 SQLite 之间，仅终态队列方法增加测试包装：先通过
正式取消方法接受取消，再调用真实 FinishSemanticBackfill。编码器和媒体来源为合成桩，
目录候选、分页、批次提交、任务和向量持久化使用真实 SQLite。

两种场景：所有图片按每页 1 项提交后取消，以及编码器不可用后、失败收尾前取消。
断言 job/operation 均 cancelled、无租约、错误码 cancelled；成功处理路径的全部
已提交向量和完成数量保留，编码失败路径为 0。后者仍向调用方返回原编码错误，
但持久化用户可见终态为 cancelled，不把两者混为一谈。

## 验证

本地 Docker 回归镜像 manifest list：
`sha256:054db25af0983147accb35b3808c02c39a890a2cceaf7d8c06b0aef3d3ae64e3`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestImageProcessorAcceptedCancelAtTerminalBoundary|TestSemanticBackfillAcceptedCancelWinsTerminalTransaction|TestSemanticClear" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m'
make arch-check
```

合成夹具、临时 SQLite、race 二进制，无真实媒体或 server222。
SQLite 9 个顶层测试（两个图片终态测试各含 2 子场景）、semantic 全包和架构检查通过。
这是确定性终态边界注入，不是随机并发压测或真实模型端到端验收。
未更新实际运行镜像，未运行完整七项检查。

## 清理检查剩余项

检查发现 FinishSemanticClear 尚未像图片终态事务一样优先处理 cancelling：
成功尝试会冲突，失败尝试仍允许写 failed。现有清理测试通过不代表这一交错已覆盖。
下一步补删除批次后、最终收尾前取消的专门复现与修复；本轮未修改清理实现。
人物生产接入、压力容量及发行仍待完成。
