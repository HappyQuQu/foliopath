# 查询与图片共同代次校验

CR-2026-030 / ADR-0019 / POST-MVP-5，FR-INT-017/019/020、NFR-INT-007/009/010。
Owner：semantic。合同为 independent-ml-generation-contract 的共同准入子片。

`BoundEncoder` 组合 GenerationRuntimeRepository、EncodedImagePreprocessor、EncodedTransport。
文本实现既有 TextEncoder，图片采用新的 stream 方法，不冒用 SigLIP tensor 接口。
查询只允许 active；图片允许 building/ready/active。统一检查代次 ID、空间指纹、维度；
推理后复查，图片准备后还会在发送前复查。错误不返回部分向量，复用向量规范化 owner。

已运行：

- `go test -race ./internal/semantic ./internal/inference/remoteml` 通过。
- `go test -race ./internal/store/sqlite -run 'TestBoundEncoderUsesPersistedGeneration|TestGenerationSpace' -count=1` 通过。
- `make arch-check`、相关包 `go vet`、`git diff --check` 通过。
- SQLite 组合测试实际创建、绑定 building 代次，验证文本拒绝、图片接受；改为 active 后文本接受，
  在模拟推理完成时实际把数据库代次退役，确认结果被丢弃。传输和预处理为合成桩，不是真实 ML 联调。
- 单元覆盖无指纹/错误维度/错误 ID/退役时无工作、处理间变更、错误输出维度和归一化。

限制：构造器验证描述格式，不证明部署身份。前后读取不解决最终写入的 TOCTOU，提交事务
仍须校验代次。未接入 internal/app、旧 backfill processor 或正式 Compose；本轮没有真实 AI
服务联调、全量仓库测试或用户照片测试。只完成可被后续 app 组合的共同校验层。
