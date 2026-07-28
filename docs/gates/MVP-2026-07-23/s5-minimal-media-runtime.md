# S5-007C 最小媒体运行时切片

## 结论

**Completed — 已降低 libvips 暴露面，但不解除 S5-007 的 No-Go。**

## 治理

- 目标版本 / Stage：`MVP-2026-07-23` / Stage 5
- task：`S5-007C`
- 需求 / 风险：`NFR-COMP-001`、`NFR-SEC-001～002` / R-017
- owner：后端负责人实现；安全负责人审阅漏洞差异；发布负责人保留证据
- 合同：根 `Dockerfile`、MVP 媒体格式合同、`tests/release/image_smoke.sh`、
  `scripts/generate-sbom.sh`、`scripts/scan-release-image.sh`
- 架构判断：不改变媒体格式合同、部署单元、信任边界或模块方向，无需 ADR

## 实现与证据

- libvips 8.16.1 从官方 release tarball 构建，Docker `ADD --checksum` 固定
  SHA-256 `d114d7c132ec5b45f116d654e17bb4af84561e3041183cd4bfd79abfb85cf724`。
- Meson 默认关闭可选能力，仅启用 JPEG、PNG、WebP、EXIF 和内置 GIF；关闭 modules、
  introspection、deprecated、examples、C++、PPM、Analyze 和 Radiance。
- 产物封装为 `foliopath-libvips` 8.16.1-1 Debian 包，SBOM 可识别，并保留 libvips
  与内置 libnsgif 许可证文本。
- 同一生产 Dockerfile 的本机原生 linux/arm64 `make test-release-image` 通过媒体矩阵、
  Compose 安全与恢复 smoke；镜像大小 206,123,724 bytes。
- loader inventory 只保留 MVP 图片 loader，不含 ImageMagick、HEIF、OpenEXR、PDF、
  SVG 或 TIFF loader。
- 固定 Trivy 扫描由 151 个包级发现 / 85 个唯一漏洞 / 15 Critical / 136 High 降为
  81 / 47 / 8 / 73；`fixedAvailable=0`。完整 JSON、摘要、SBOM 与 runtime inventory
  保存在本地 `build/supply-chain-minimal-media-final/`，不提交源码树。

## 剩余边界

本切片只关闭“生产 libvips 闭包可安全收缩且媒体合同不回退”的问题。剩余 8 Critical 与
73 High 未被接受，最终 amd64/arm64 digest 也尚未复扫和签署，因此 `S5-007B`、R-017、
`S5-009` 和 Release Candidate 继续阻断。后续若自建最小 FFmpeg，必须建立新的高风险
切片并重复完整媒体、恢复、SBOM、许可证和漏洞证据。
