# S5-007D 最小 FFmpeg 运行时切片

## 结论

**Completed — 已移除通用 FFmpeg 闭包，但不解除 S5-007 的 No-Go。**

## 治理

- 目标版本 / Stage：`MVP-2026-07-23` / Stage 5
- task：`S5-007D`
- 需求 / 风险：`NFR-COMP-001`、`NFR-SEC-001～002` / R-017
- owner：后端负责人实现；安全负责人审阅漏洞差异；发布负责人保留证据
- 合同：根 `Dockerfile`、MP4/MOV/MKV 媒体合同、
  `tests/fixtures/media-matrix/`、`tests/release/image_media_smoke.sh`
- 架构判断：不改变媒体格式合同、部署单元、信任边界或模块方向，无需 ADR

## 实现与证据

- FFmpeg n7.1.5 从官方 GitHub tag archive 构建，Docker `ADD --checksum` 固定
  SHA-256 `e3963a50831c985933e1a625ed566ec4c7adb5c012c34fa9f84438e1d61bdacc`。
- `--disable-everything --disable-network` 后只启用 MOV/MP4 与 Matroska demux、
  常见视频解码器、scale、libwebp 海报编码、image2pipe 和 file/pipe protocol；
  不包含网络、采集设备、转码编码器或 GPL/x264。
- 产物封装为 `foliopath-ffmpeg` 7.1.5-1 Debian 包，SBOM 可识别；构建声明
  LGPL 2.1+，相关许可证文本随镜像存在。
- 媒体 smoke 改用隔离的固定基础生成器制作合成夹具，候选运行时只验证真实产品路径：
  libvips 图片读取、MP4/MOV/MKV 与 FFV1 元数据、首帧缩放、WebP 海报和损坏 MP4 拒绝。
- 同一生产 Dockerfile 的本机原生 linux/arm64 媒体、Compose 安全和恢复 smoke 全部
  通过；镜像大小 55,103,419 bytes。
- 固定 Trivy 扫描由 S5-007C 的 81 / 47 / 8 Critical / 73 High 降为
  49 个包级发现 / 32 个唯一漏洞 / 5 Critical / 44 High，`fixedAvailable=0`。
  完整报告和 SBOM 保存在本地 `build/supply-chain-minimal-runtime-final/`。

## 剩余边界

剩余 Critical 为 `perl-base` 4 条和 GLib 1 条；High 主要来自 curl、GLib、expat 与
基础系统包。它们尚未接受，最终 amd64/arm64 digest 也尚未复扫和签署，因此
`S5-007B`、R-017、`S5-009` 和 Release Candidate 继续阻断。
