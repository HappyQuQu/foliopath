# FIX-2026-09-01：人脸检测与嵌入原生 session 边界

- Slice：POST-MVP-5 revision 2 / S2C（`INT-241`、`INT-250`）
- Gate：[INT-S2C 人脸隐私与模型准入](../gates/POST-MVP-5/int-s2c-privacy-ready.md)
- 影响不变量：模型文件只通过 kernel handle 打开；runtime 不可用时失败关闭；production composition
  在 Release Gate 前保持不可达

## 变更

`internal/inference/onnx` 新增独立的人脸 detector 与 embedding session，`internal/inference/faceonnx` 与
`internal/media/imagevips` 形成仍未注册的 production adapter：

- 分别只接受 manifest 中唯一的 `face_detector` / `face_embedder` role，并继续要求 `/proc/self/fd/<n>`
  runtime path 与声明大小一致；
- detector 固定 YuNet `input float32[1,3,640,640]` 及 12 个 stride 8/16/32 输出，按冻结阈值解码 box、
  五点 landmark 与 score，并以有界 Top-K/NMS 收敛输出；
- libvips 只从已打开的流解码，校验声明格式/大小/像素上限，自动方向校正并把长边限制为 1600；adapter
  以 BGR CHW 零填充 detector 输入，把 box/landmark 映射回原图，执行五点 similarity alignment，再按
  `(RGB-127.5)/127.5` 生成 AuraFace tensor；
- 固定 ONNX Runtime `1.28.0`，严格校验 AuraFace 候选图的
  `data float32[-1,3,112,112] -> 1333 float32[1,512]` 合同；
- 每次执行创建独立 run options，context 取消会终止当前推理；关闭 session 会释放 ORT session、environment
  和模型文件句柄；
- 非 Linux、无 CGO 或未启用 `onnxruntime` build tag 时继续返回稳定的 runtime unavailable；
- 没有把候选登记进 production model catalog，也没有注册 face route、worker 或自动模型获取。

该边界覆盖审核候选的 decode、detector、五点对齐、确定性 candidate quality 投影与 embedding，但不包括
正式模型准入、face generation/model-package activation owner 或 production composition，因此不能单独完成
`INT-241`。

## 验证

- `gofmt -w internal/inference/onnx/*.go`
- `go test -count=1 ./internal/inference/onnx`
- Linux/arm64 Docker 中以冻结的 ONNX Runtime `1.28.0` C API 编译
  `go test -c -trimpath -tags onnxruntime ./internal/inference/onnx`
- 对只读挂载、SHA-256 `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4`
  的 YuNet 与 SHA-256
  `a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60` 的
  `glintr100.onnx` 执行 `TestNativeFaceDetectorCandidate` / `TestNativeFaceEmbeddingCandidate`；detector
  完成 12-output graph execution，embedder 得到有限、非零的 512 维输出
- 在同一 Linux/arm64 原生依赖闭包中，以精确 SHA-256
  `ab8413ad9bb4f53068f4fb63c6747e5989991dd02241c923d5595b614ecf2bf6` 的 OpenCV Zoo 公开 JPEG 运行完整
  `libvips -> YuNet -> alignment -> AuraFace`，至少产生一个有限、非零的 512 维 candidate embedding
- `make fmt arch-check generate-check lint test test-integration test-e2e`
- `go test -race -count=1 ./internal/face ./internal/api ./internal/store/sqlite ./internal/inference/... ./tests/integration`
  （全部通过；SQLite race 段耗时约 373 秒）
- `make spike-ai`（包含 format v3 hostile-manifest 与本地功能冒烟安全合同）

原生记录见
[AuraFace production-boundary Linux/arm64 smoke](../evidence/int-001/auraface-production-boundary-linux-arm64-2026-09-01.md)。
S2C Release Gate 仍为 **No-Go**。
