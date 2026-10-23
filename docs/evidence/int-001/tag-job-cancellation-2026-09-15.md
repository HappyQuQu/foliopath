# 标签生成任务取消边界

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016、NFR-INT-007。
Owner semantic/SQLite；取消、任务领取及资源通知合同不变。

新增真实 SQLite 测试先在 Docker 复现：准入前取消滞留 cancelling、取消后成功
收尾冲突、取消后失败收尾覆盖为 failed；单纯停机仍 running 正常。

TagJobProcessor 现在复用资源准入取消通知和 Process 退出时最多 5 秒的收尾。
FinishTagJob 在原事务内要求 job/operation 同为 cancelling 时优先收尾 cancelled，
保持原领取版本和终态校验。停机无用户取消请求则继续保留租约恢复。

新增四个子场景使用无已入库图片向量的合成库（零候选），验证任务与操作状态一致、
取消释放租约、停机保留租约；准入测试占满资源额度且调用前 context 已结束。
这不是非空生成中取消，也不是实际 worker 心跳与最终非空提交交错的完整证明。

## 本地 Docker

修复后回归镜像 manifest list：
`sha256:620c74551a764564e48de26e200b3b1993a18808083ca9df544360d4ceaac701`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestTagJob|TestTagSuggestionJob" -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m && /out/resourcecontrol.test -test.count=20 -test.timeout=2m'
make arch-check
```

合成夹具、临时 SQLite、race 二进制，无网络、真实媒体或 server222。
最终 SQLite 3 个顶层测试（新增含 4 子场景）、semantic 全包、20 轮资源控制及架构检查通过。
未更新实际运行镜像，未执行全套七项检查。非空生成/实际取消恢复、跨库审核账本、
模型质量、人物、容量及发行仍待完成。
