# 语义搜索边界失败关闭

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：已批准 S2A 切片的例行正确性与安全修复
- Requirement：`FR-INT-002`、`FR-INT-005`、`NFR-INT-004`、`NFR-INT-008`
- Owner：`internal/semantic` search service
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 修复

`SearchService` 不再直接信任 `VectorSearchRepository` 的返回页。在创建公开结果或下一页 cursor 前，
capability owner 会验证：

- 结果数不超过请求上限，library/asset ID、score 均有效；
- 每个结果属于当前 snapshot 的 enabled/online member，且 selected-library scope 不串库；
- 资产不重复，顺序严格符合 `score DESC, asset_id ASC`；
- cursor 后续页的每项都严格位于 continuation tuple 之后。

任一约束失败都收敛为内部 `ErrInvalidEmbeddingRecord`，HTTP 只会通过既有统一映射返回脱敏 500；不会
生成可循环、跳项或跨 scope 的 cursor，也不会把内部异常降格成客户端输入错误。

HTTP adapter 还会把 catalog 批量投影逐项绑定回 semantic match 的 asset ID 与 library ID，并要求来源
仍为 online。搜索与投影之间发生删除、重排、跨库 ID 复用或 offline 转换时，分别收敛为
`semantic_cursor_stale` 或 `semantic_not_ready`，不把竞态后的新资产作为旧向量命中返回。

资源边界同步落实既有 OpenAPI cursor 上限：semantic owner 在任何 snapshot、解码、encoder 或 vector
工作前拒绝小于 8 或超过 2,048 bytes 的 cursor；HTTP adapter 在 `url.ParseQuery` 分配前拒绝超过
16 KiB 的完整 raw query。该 transport 上限容纳 512 个四字节 Unicode code point 的最坏百分号编码、
最大合法 cursor 与其余冻结参数。

## 验证

- `go test ./internal/semantic ./internal/store/sqlite ./internal/api`：通过；
- `go test ./internal/api ./internal/semantic ./internal/catalog`：通过；
- `make test`：通过；
- `make fmt`：通过；
- `make arch-check`：通过；
- `git diff --check`：通过。

本修复不注入 production `SemanticSearch`，不完成新的 `INT-xxx` 主任务，也不解除 tokenizer、最终模型、
合法质量集、原生 amd64、容量或供应链阻塞；S2A 仍为 **No-Go**。
