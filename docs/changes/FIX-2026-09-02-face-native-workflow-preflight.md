# FIX-2026-09-02：人脸候选原生双架构工作流预检

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-241`、`INT-250`
- 影响不变量：QEMU 不得冒充 native；候选 functional preflight 不得声称 production、quality 或 compliance
  approval

## 问题

YuNet/AuraFace 的 production boundary 已在 Linux/arm64 和模拟 Linux/amd64 手工通过，但远端原生 workflow
只运行仓库、libvips、搜索和容量基线。即使入口被调度，也不会在两个原生 runner 上复现精确 detector、
embedder、ORT 和公开 JPEG 的完整 pipeline。

## 修复

- 新增固定 Docker test stage：按 runner 架构选择 ONNX Runtime 1.28.0 官方 archive，复核 archive SHA、
  VERSION 和 upstream commit 后编译 `libvips onnxruntime` tagged candidate test。
- 新增 Linux-only harness，同时绑定 `uname`、Go arch 和 Docker arch；任何不匹配立即失败，禁止
  `DOCKER_DEFAULT_PLATFORM`/QEMU 式架构替换进入原生证据。
- detector、embedder 和公开 fixture 从固定 revision URL 下载并逐项校验 SHA；运行时网络关闭，rootfs、
  模型和 fixture 只读，CPU/内存/PID/tmpfs 有界。
- artifact 只记录模型/fixture/runtime hash、candidate count 和 0.001 量化单向指纹，不记录图片、crop、路径、
  embedding 或身份标签；三个 approval flag 必须显式为 false。
- paired verifier 以严格 JSON、普通文件边界和同 source commit 校验每个架构的 candidate record，并要求
  两边使用相同 ORT commit、detector/embedder/fixture digest 和 candidate count；数值指纹允许不同且不被
  解释为 `1e-3` parity。最终模型仍必须走 `verify-intelligent-media-native-model-evidence`。
- 同一镜像另在 4 CPU/4 GiB 下运行 100k × 512 合成聚类并生成独立 `face-capacity.json`；paired verifier
  要求两侧 workload、非质量声明与确定性结果指纹一致。该项由后续
  [LSH 桶边界容量修复](FIX-2026-09-01-face-lsh-bucket-boundary-capacity.md)补入。

## 回归证据

- native evidence verifier 单测覆盖 candidate 非法批准声明、跨架构候选数不一致和原有歧义 JSON 拒绝。
- architecture fitness 固定 Linux/架构三元组、断网只读运行、候选非批准 flag、ORT 版本/commit 和 tagged
  build。
- `bash -n`、verifier 单测与 `make arch-check` 通过；远端 workflow 尚未在同一候选 SHA 上真实运行，故本
  记录不解除 native amd64、最终模型、质量、容量或合规 Gate。
