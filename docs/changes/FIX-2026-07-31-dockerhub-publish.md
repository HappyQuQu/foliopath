# FIX-2026-07-31：Docker Hub 双架构自动发布

- 类型：已批准 MVP Stage 5 发布切片内的交付自动化
- 关联质量：`NFR-COMP-001`、`NFR-OPS-001`
- 目标版本与阶段：MVP / Stage 5 Release
- Owner：`.github/workflows/dockerhub.yml`
- 合同：根 `Dockerfile`、`compose.yaml`、Docker Hub OCI manifest
- 关联 Gate：S5-001B/002、S5-007A、S5-008、S5-009A
- 不变量：发布不修改原媒体；凭据只来自 GitHub Secrets；预发布不得覆盖稳定 `latest`

## 行为

推送到 `main`、推送 `vMAJOR.MINOR.PATCH` Git tag 或手动运行时，workflow 使用
Buildx/QEMU 构建生产 Dockerfile，并向 Docker Hub 推送一个包含 `linux/amd64` 和
`linux/arm64` 的 OCI image index。PR 不触发发布。

`main` 推送更新 `latest` 和 Git SHA 标签；`vMAJOR.MINOR.PATCH` 标签额外发布完整版本
与 `major.minor`。管理员也可以手动运行 workflow，指定 `edge` 等 Docker tag。

发布目标固定为 `evanqu/foliopath`，认证使用 secrets `DOCKERHUB_USERNAME` 和
`DOCKERHUB_TOKEN`。构建生成 SBOM 与 provenance；可选
`DOCKERHUB_DESCRIPTION_TOKEN` 用于同步根 README。

## 证据

- workflow 语法和 action 版本在修改后使用本地 `actionlint` 检查。
- 发布结果必须用 `docker buildx imagetools inspect <image>:<tag>` 确认同时包含
  `linux/amd64` 与 `linux/arm64`。
- 当前 Release Candidate No-Go 状态保持不变；自动化存在不表示稳定版已经发布。
