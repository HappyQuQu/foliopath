# 语义清理任务：准入取消与停机区分

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite，复用已有资源准入通知；API、删除批次和并发额度不变。

ClearProcessor 实现取消通知并在 Process 取消退出时复用最多 5 秒清理。
FinishSemanticClear 原先允许 running 直接写 cancelled；现要求先处于 cancelling，
否则返回冲突，防止把仅停机而无取消请求的任务改为 cancelled。

新增组合测试：占满资源槽，已取消 context 进入资源包装器；
用户取消后 job/operation 为 cancelled，设置为 disabled/degraded，原 1 条向量保留。
无用户请求的停机则仍 running、设置 clearing，保留租约恢复机制和向量。
取消前尚未执行删除批次；不声称取消能够恢复此前已经清理的派生数据。

## 本地 Docker

初次构建发现取消检查补丁落入相似查询片段、测试调用错用仓储方法名，
已修正后重新构建。最终回归镜像 manifest list：
`sha256:87980b4b146240bd4e374af2f25187e02533eafaa72cc7c6d37a301cae246cf8`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run TestSemanticClear -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m'
make arch-check
```

仅临时 SQLite 和合成向量，无真实媒体、server222 或服务端用户数据删除。
最终 SQLite 7 个顶层测试（新增测试含 2 子场景）、semantic 全包、资源控制 20 轮
和架构检查通过；Docker 测试二进制带 race。
未重建实际运行镜像，未执行全套七项检查。
扫描状态机已检查：RunClaimedFullScan 在入口取消时直接返回，运行中 abort 使用
scanner 自己的 CancelFullScan 规则；没有直接套用 AI 的 CAS 或修改扫描实现。
扫描等待期取消、清理实际混合任务/批次边界交错、人物和发行仍未完成。
