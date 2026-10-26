# 标签审核清理：非空批次取消

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-016 / NFR-INT-007。
Owner semantic/SQLite，沿用清理事务与取消接口，无 API/迁移变化。

合成库插入两条审核记录，正式标签通过 curation 仓储写入。真实清理处理器每批
删除 1 或 1000 条；测试包装器在第一批提交后接受取消，不取消 context。
最初夹具使用过期审核版本，被准入拒绝，修正为查询当前审核版本后重新运行。
随后两个场景复现：部分批次终态 failed/internal_error，最终批次收尾冲突。

FinishTagReviewClear 现在于终态事务内优先处理已持久化 cancelling，保留原领取版本
和已结束状态校验。部分批次剩余 1 条，最终批次剩余 0 条；不恢复已删除审核记录。
验证任务/操作均 cancelled、删除数和完成数准确、无租约、正式标签状态前后相同，
重复失败收尾不能覆盖取消。部分路径仍向调用方返回下一批的冲突错误。

## 本地 Docker

回归镜像 manifest list：
`sha256:40f955803ff3fefcf0d9adaae23c111d8ae12a679bad3f2321719006cd796f5b`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run TestTagReviewClear -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m'
make arch-check
```

临时 SQLite、合成夹具、race 二进制，无网络或真实媒体，不访问 server222。
最终 SQLite 5 个顶层测试（新增批次测试 2 子场景）、semantic 全包和架构检查通过。
新增测试未生成审核幂等请求账本，跨库混合请求账本与取消交错不在本轮证据内。
未更新实际运行镜像、未执行全套七项检查或真实 worker 交错压测。
实际镜像/环境、其余任务取消矩阵、人物生产接入、容量与发行仍待完成。
