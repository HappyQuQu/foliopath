# FIX-2026-09-01：semantic v2 production tokenizer 与文本推理闭环

- 目标版本：`POST-MVP-5` revision 2 / S2A
- 关联 Gate：`INT-S2A Backend Evidence Ready`（**Backend Ready / Release No-Go**）
- 影响任务：`INT-202`、`INT-203`、`INT-207`、`INT-208`、`INT-214`
- 影响不变量：模型来源只经安全文件边界；runtime/query/vector 不外泄；无审核模型时发布失败关闭

## 原因

ADR-0014 已接受 fail-closed 后端实现，但 production 仍缺 semantic package format v2、SentencePiece
FD adapter、ORT text session、generation-bound text owner 和搜索 route composition。已有隔离 spike 能证明
方案可行，却不能让正式后端消费 exact `spiece.model` 与 text graph。

## 实现

- `internal/aimodel` 成为 semantic format v2 的唯一 parser owner：严格要求 portable ONNX、顶层许可、
  `image_encoder`、`text_encoder`、`sentencepiece_model` 三个唯一角色及四个精确 contract ID。v1 仍可被
  catalog 读取以解释历史记录，但 activation 在打开 runtime 文件前明确拒绝。
- `internal/inference/sentencepiece` 通过 Linux `/proc/self/fd/<fd>` 加载调用方已安全打开的 model，固定
  63 pieces + EOS/pad 到 64 的合同；默认、非 Linux 或未带 build tag 时稳定失败关闭。
- production ORT adapter 新增 `[1,64] int64 → [1,768] float32` text session。每次执行使用独立
  RunOptions，context 取消触发 terminate；输出必须有限，session close 与当前执行串行。
- activation smoke 同时打开 SentencePiece 和 text graph，验证固定 `red armor portrait` token IDs、ABI
  与有限非零 768 维输出后才允许 generation commit。
- application text owner 每次复核 active generation、available model、reviewed manifest 与 source；最多
  一个 resident session，单次 30 秒 hard timeout、5 分钟 idle unload，切代或非取消故障关闭旧 session。
  tokenizer role 在打开文件前先完成 exactly-one 校验，避免 hostile manifest 导致半开资源。
- production app 注册 semantic image/text search 与 video semantic search service，但继续构造空 reviewed
  catalog。无审核模型、缺 native build tag 或来源失效时只返回稳定 unavailable，不影响核心浏览。

## 证据边界

默认 Go 回归覆盖 v1/v2 confusion、角色/contract/path/license 边界、不可用 runtime、session reuse/switch/
fault/close、搜索 composition 与 activation 前置拒绝。官方 Microsoft ORT 1.28.0 Linux/arm64 archive（SHA-256
`e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb`）与发行版 SentencePiece header/library
已完成 `sentencepiece onnxruntime` tagged production 包联合编译链接。

这些证据关闭上述五个后端实现任务，但不批准任何候选模型字节或 release catalog entry，也不替代最终
合法质量集、原生 Linux/amd64 运行、100k 联合容量、完整 SBOM/VEX/notices/provenance 与负责人签署。
按后续 [CR-2026-022](CR-2026-022-s2-backend-release-gate-separation.md)，这些最终 release 输入归入 S4；
`INT-209/210/215` 以 S2 production-boundary 故障与本地容量证据收口，INT-S2A 现为 **Backend Ready /
Release No-Go**。
