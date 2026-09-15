# 图片回填准入取消收尾

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016 / NFR-INT-007。
Owner semantic，复用 resourcecontrol 既有准入取消通知和 SQLite 图片任务终态 CAS。
不更改扫描状态机、API、预算或部署拓扑。

代码检查确认图片 BackfillProcessor 未实现准入取消通知，且 Process 在 context
取消后直接返回，未执行及时取消收尾。现实现 BackgroundAdmissionCancelled，
使用最多 5 秒清理上下文；Process 的取消退出也复用该方法。
仅数据库已接受的取消能写入 cancelled；停机而无取消请求保持 running 待租约恢复。

新增 SQLite 组合测试覆盖 4 种组合：已占满资源槽/直接进入 processor，分别用户取消/停机。
context 在调用前已取消，验证取消准入后的收尾，不声称本用例证明等待中准确唤醒时序。
断言推理不启动、完成数 0、job 与 operation 状态一致，未释放占用槽也可返回。

## 本地 Docker 验证

镜像 `foliopath-remote-ml-regression:local` 构建成功，manifest list
`sha256:3b2f6ce66fd6ec5fd2aadcea156d163ee9a8f4a78af56c43ac953fae240ae9d7`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "Test(ImageBackfillCancellationBeforeAdmission|StreamBackfillPersistedLifecycle|SemanticBackfillClaimLeaseCancellationAndFinishAreCASBound)$" -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m'
make arch-check
```

使用临时数据库和合成记录，不读取真实媒体，不访问 server222。
最终 SQLite 3 个顶层测试（新增 4 子场景、流式生命周期 2 子场景）通过；
semantic 全包、资源控制 5 项各 20 轮及架构检查通过，Docker 测试二进制带 race。
仅定向回归，未执行完整七项检查或重建实际运行镜像。
图片取消与最后提交/失败并发、真实图片等待期取消、扫描/清理任务取消、人物及发行仍待检查。
