# 标签生成非空评分与提交取消交错

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016、NFR-INT-007。
Owner semantic/SQLite。本轮仅增加测试，不修改生产逻辑。

真实 TagJobProcessor、ControlledTagPlanBuilder 和 SQLite，合成图片/词条向量均为 [1,0]，
1 个候选会生成 1 条建议。确定性包装器注入四种顺序：

- 评分后接受取消但 context 未结束：提交拒绝，任务 cancelled，建议 0 条。
- 评分后接受取消并结束 context（模拟心跳通知）：cancelled，建议 0 条。
- 评分后仅结束 context（停机）：任务保持 running、租约保留，建议 0 条。
- 提交建议后、终态事务前接受取消：任务 cancelled、无租约，完成数 1，已提交建议 1 条保留。

所有情况核对 job/operation 状态一致。未提交数据不会被写入，已提交数据不会被取消回滚。
不执行真实文本推理，不能把心跳式 context 注入称为实际 worker 心跳验收。

## 本地 Docker

回归镜像 manifest list：
`sha256:a185139b7a2dcbab3698a7f549ed1fcc08a12bc7087f4a5d95be97165b2ea3a8`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run "TestTagJob|TestTagSuggestionJob" -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m'
make arch-check
```

临时 SQLite、合成向量、race 二进制，无网络、真实媒体或 server222。
最终 SQLite 4 个顶层测试（新增含 4 子场景）、semantic 全包与架构检查通过。
未改生产代码，无需为本次测试重建运行镜像；未执行全套七项检查。
真实文本推理中断、跨库审核账本、质量/浏览器/人物/容量及发行仍待完成。
