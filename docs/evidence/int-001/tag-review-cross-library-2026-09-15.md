# 跨库审核混合账本：清理与取消隔离

CR-2026-030 / POST-MVP-5 / FR-INT-004、FR-INT-016、NFR-INT-007。
Owner semantic/SQLite/curation。本轮增加回归，无生产代码变化。

临时 SQLite 合成 A/B 库各 1 条审核记录，混合幂等请求包含 A+B 两项，另有 B 独立请求。
B 的正式标签通过 curation 仓储写入。清理 A 一批后通过正式取消方法接受取消，
再执行成功终态尝试，要求最终 cancelled。

验证 A 审核为 0、B 审核仍 1；混合请求及两项子账本全部移除，避免部分结果重放；
B 独立请求及子项各保留 1，B curation 前后完整一致；终态后再次删除被拒绝。
这沿用既有清理契约：跨库混合请求整体失效，不是删除另一库的审核或正式标签。

## 本地 Docker

回归镜像 manifest list：
`sha256:390c27b83061f8506d05aae7fad1b9dcbcda76379d0b61018a09bd25db3478ba`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local '/out/sqlite.test -test.run TestTagReviewClear -test.v -test.timeout=2m && /out/semantic.test -test.timeout=3m'
make arch-check
```

SQLite 6 个顶层测试、semantic 全包及架构检查通过。race 二进制、无网络、临时数据库，
不读取或修改真实媒体，不访问 server222。
混合审核请求账本使用合成 SQL 夹具，不声称正式 HTTP 跨库批量审核重放已验收；
仅检查持久化行与 curation 隔离，不模拟删除期间并发重放。
无生产变化，未重建运行镜像，未执行完整七项检查。
正式跨库重放/浏览器、模型质量、人物、容量与发行仍待完成。
