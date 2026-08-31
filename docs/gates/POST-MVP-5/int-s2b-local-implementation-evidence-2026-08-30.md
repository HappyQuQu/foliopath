# INT-S2B 本地实现证据（2026-08-30）

## 结论

`INT-221～226` 的本地 C/D 状态机、持久队列、HTTP 边界与定向回归已经实现；本记录不改变
[INT-S2B Backend Evidence Ready](int-s2b-backend-evidence-ready.md) 的 **No-Go**。合法代表性 tag/video
质量集、native linux/amd64、最终 4 CPU/4 GiB/100k 联合负载和签署供应链仍属于 `INT-227` 与 Gate Go
的外部/发布输入，不能由合成测试替代。

2026-08-31 后续已完成 `INT-228` Gate 复审并保持 **No-Go**；上述输入继续阻断 `INT-227` 和 Gate Go，
但不再把“是否执行过复审”本身记为未完成。

## C：受控标签建议

- migration 25、28～32 提供 immutable vocabulary、tag embedding cache、Top-5 suggestion、review lineage、
  hashed batch-review ledger、每库 review revision、独立 clear queue、每资产/每库 truthful progress 和
  missing/all durable job；所有 migration 均只追加。
- generation、vocabulary snapshot、asset source fingerprint 同时绑定 suggestion；accepted/dismissed review
  不因 suggestion rebuild 删除，已有人工作为标签或已 review 的 asset/tag 不重新展示。
- batch review 最多 100；accept 调用唯一 curation service。curation 成功后 suggestion 仍保持 pending，直到
  review CAS 成功；hashed request ledger 可在进程中断后重放/补偿，且 retry-safe curation 不重复写 tag。
- 显式 `semantic_tag_asset_progress` 记录零建议资产为 ready，列表覆盖率不再用 suggestion 行数猜测。
- review clear 使用强 ETag、hashed idempotency、lease/retry/cancel；只删除 `ai_tag_reviews`，回归证明
  `asset_tags` 保留。

## D：完整故事板视频语义

- migration 26/27 提供完整 4/10 frame plan、coverage 和独立 job queue；worker 只通过
  `thumbnail.SemanticStoryboardRepository` 与 cache reader 消费已发布 WebP sprite，不调用 FFmpeg、
  不打开原视频。
- 任一 frame 失败都不提交部分 plan；无故事板/缓存丢失进入 degraded 并触发唯一 media worker 重排；
  10→4 的变化由 storyboard fingerprint/transform version 使旧 plan 失效。
- search repository 使用每视频 max score，并以最小 ordinal 作为同分 best-frame；cursor 独立绑定
  generation/catalog/progress，排序为 score DESC、asset ID ASC。
- video job 已在 production composition 中复用 image inference session、全局 background=1、通用 operation
  cancel owner、startup lease recovery 和媒体 cache repair。video search 仍因 ADR-0014/text adapter Gate
  不注册，不能伪装可用。

## 验证面

本轮实际执行并通过：

- `go test ./internal/semantic ./internal/store/sqlite`
- `go test ./internal/app ./internal/api ./internal/aimodel`
- `go test ./tests/architecture/...`
- `make contract-check`
- `make generate-check`
- `make arch-check`
- `make fmt`
- `make lint`
- `make test`
- `make test-race`
- `make test-integration`
- `make test-e2e`
- `make test-libvips`

`make test` 首次执行发现 `run.go` composite literal 因新增较长字段改变 gofmt 对齐，导致旧 architecture
fitness 的精确 composition substring 失败；把新 S2B 字段分为独立 literal group 后，定向 architecture
suite 与完整 `make test` 均已重跑通过。首次失败仍保留在本记录，未被错误写成通过。

`make test-libvips` 随后在 Dockerfile 固定的 production 依赖闭包中重新构建 Web SPA 与
`foliopath`（`CGO_ENABLED=1`、`-tags=libvips`），并成功执行
`go test -count=1 -tags=libvips ./internal/media/imagevips`。该结果证明当前宿主可复现的容器化
libvips 编译/适配器回归；它不是 native linux/amd64 证据，也不替代 `INT-227` 的合法质量集。

完整 race 复跑也以退出码 0 结束；其中 `internal/store/sqlite` 在 race instrumentation 下运行
244.763 秒，S2B durable queue/事务回归包含在该包中。`make test-e2e` 重新构建 arm64 application
smoke 镜像并输出 `application container smoke passed`。这些是本地并发与容器纵向证据，仍不能解释为
最终双架构模型质量、100k 联合负载或已签署发布证据。

`make spike-ai` 通过。随后执行的 `make spike-capacity` **失败**，不得登记为通过：darwin/arm64、
`GOMAXPROCS=4` 的 10,000 目录/100,000 媒体基线中 `searchKeysetP95Us=296212`，超过既有
`s4-search-v1` 的 250,000 µs 预算；1000 层深目录汇总子测试通过。该失败与既有
[10 万媒体搜索 keyset 查询计划维护提案](../../changes/FIX-2026-08-27-search-keyset-query-plan.md)
一致，说明最终联合容量 Gate 仍有可复现的非 AI 搜索债务；不得通过提高预算或只重跑一次来解除。
