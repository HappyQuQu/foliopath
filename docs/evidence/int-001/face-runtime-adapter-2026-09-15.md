# 人脸安全图片适配

CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2。
Owner：imagevips 解码，remoteml 运行时适配，face 质量规则；app 组合仍未完成。
沿用 [face-v1 私有合同](face-service-contract-2026-09-15.md)，无公开 API 或数据库变更。

## 本次实现

- `NewFaceRuntime` 实现现有 `face.Runtime`，接收已打开的 reader，不接受路径。
- 复用 `face.ImageDecoder` / imagevips，将最长边限制为 1600，检查 RGB 长度后重建无元数据 PNG；编码支持上下文取消和 8 MiB 写入限制，调用既有独立身份绑定的 FaceClient。
- 返回 `face.Candidate`，质量计算收归 `face.SizeWeightedDetectionQuality`。旧 faceonnx 也调用同一 owner，公式不变：检测分数 × min(1, 边长较小值 / 64)。这是尺寸启发式，不是校准后的识别概率；不新增阈值，不把检测分数直接当质量。
- 最终启用必须由独立人脸代次绑定模型、输入变换、质量规则和阈值配置；本次不绕过激活、不自动发布现有阈值。

## 本地 Docker 验证

- `remote-ml-image-probe` 构建成功；生成带 race 的 remoteml、face、faceonnx 测试二进制。
- `remote-image-probe.test -test.skip 'Probe$' -test.timeout=2m` 通过，新增两项图片适配检查（像素重建/质量、异常解码/取消/编码上限）。
- `face.test -test.timeout=3m`、`faceonnx.test -test.timeout=3m` 全包通过。
- `TestFaceServiceProbe` 真实离线探针通过（1.22 秒）：直接传输加 JPEG/PNG/GIF/WebP 四格式的 libvips→PNG→buffalo_l 调用。全部为 320×240 合成纯色图片，结果为零人脸；非空框和质量计算只由模拟结果测试覆盖。
- `make arch-check`、本次文件 `git diff --check` 通过；Go 文件已 gofmt。

AI 服务使用 `--network none`、只读根/模型缓存，无宿主端口；Go 容器共享其网络命名空间。未使用真实媒体或 server222。临时测试服务已停止、删除；当前 app/AI candidate 未替换。

## 剩余

`run.go` 尚未组合远程人脸运行时。独立人脸服务代次/阈值准入、worker/入库/聚类与人物 HTTP 装配未完成；真实非空人脸、主应用端到端、质量校准、压力和发行未验收。本轮未运行完整七项检查。
