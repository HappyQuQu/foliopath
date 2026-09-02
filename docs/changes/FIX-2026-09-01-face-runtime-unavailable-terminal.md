# FIX-2026-09-01：人脸 runtime 不可用终止语义

- Slice：POST-MVP-5 revision 2 / S2C（`INT-241`、`INT-250`）
- Gate：[INT-S2C 人脸隐私与模型准入](../gates/POST-MVP-5/int-s2c-privacy-ready.md)
- 不变式：模型/runtime 不可用时失败关闭，不推进虚假覆盖率，不发布部分聚类

## 问题

人脸分析器在任务处理中返回 `ErrRuntimeUnavailable` 时，旧 worker 会把它计为单个资产的普通失败并继续
扫描。若该错误持续，任务可能遍历全库后以 degraded success 结束并尝试重建 cluster；这会把代次级模型
故障误表示为媒体级失败。

## 变更

- `JobProcessor` 将 `ErrRuntimeUnavailable` 识别为整项任务的稳定 `model_unavailable` 终态；
- 在任何 checkpoint、完成/失败/陈旧计数或 observation 写入前终止；
- 不进入 cluster rebuild，库设置由 canonical `FinishFaceJob` 转为 `awaiting_model`；
- worker context 已取消时不写业务终态，保留 durable claim 供既有 lease recovery 处理；
- 普通单媒体解码/分析错误仍按失败项统计，不扩大模型错误分类。

## 验证

- `go test -count=1 ./internal/face ./internal/store/sqlite -run 'FaceJob|Processor'`；
- runtime-unavailable 回归证明 analyzer 只调用一次，job/operation 为 `model_unavailable`；
- progress 的 completed/failed/stale/checkpoint 全部保持为零；
- 没有创建 cluster build，settings 保持 enabled 并转为 `awaiting_model`。

该修复关闭 runtime 故障分类子项，不提供最终审核 detector/embedder、真实质量、双架构、联合容量或签署；
S2C Release Gate 仍为 No-Go。
