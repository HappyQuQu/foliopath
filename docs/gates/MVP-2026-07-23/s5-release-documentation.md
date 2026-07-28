# S5-008 发布文档 Gate

## 结论

**Go — `S5-008` 发布候选文档校对完成。**

README、候选 Compose、环境参数、部署/权限、离线备份恢复、升级回滚、支持格式和已知限制
现在与当前 Stage 5 实现及 Gate 一致。文档明确说明没有公开稳定镜像，候选不得当作稳定版
部署；这项 Go 只关闭文档一致性任务，不改变 S5-001～007、S5-009～010 的结论。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-008`
- 需求/质量：`FR-DEP-001～002`、`FR-MED-004～007`、`NFR-SAFE-001`、
  `NFR-SEC-001～002`、`NFR-COMP-001`、`NFR-OPS-001`
- owner：发布负责人拥有候选运行说明；后端负责人拥有数据/恢复和媒体格式事实；
  QA/安全负责人拥有未满足 Gate 与限制披露
- 合同：根 `Dockerfile`、`compose.yaml`、`.env.example`、`internal/media/formats.go`、
  `README.md`、`docs/deployment.md`
- 自动化：`tests/architecture/release_docs_test.go`、`make release-docs-check`
- 风险：R-002、R-004、R-007、R-009、R-011、R-014、R-016、R-017
- 架构影响：没有改变部署单元、路径、持久化、信任或媒体格式边界，不需要 ADR

## 校对结果

| 领域 | 固定后的说明 |
| --- | --- |
| 发布状态 | Stage 0～4 已 Integrated Done；根镜像/Compose 仍是 Stage 5 candidate，没有稳定拉取引用 |
| 候选启动 | 当前源码可构建明确本地 tag 进行评估；正式部署必须等待版本或 digest |
| 网络 | Compose 只发布宿主回环端口；非回环应用监听必须配置精确单跳 TLS proxy CIDR |
| 存储 | 唯一 `/library:ro` 与可写 `/app/data`；媒体根后代无 mount，活动 SQLite 不放 SMB/NFS |
| 权限 | UID/GID 65532、只读根、`cap_drop: ALL`、`no-new-privileges`、受限 `/tmp` |
| 备份 | 停止应用后备份完整 SQLite family；cache/tmp 可省略；禁止运行时单独复制 DB |
| 恢复 | 解包到新的空数据目录，校正运行时映射权限并验证后再切换，不覆盖唯一活动副本 |
| 升级/回滚 | migration 只向前；回滚配对旧镜像与升级前数据备份；原生双架构不同不可变候选演练已通过 |
| 图片 | `.jpg`/`.jpeg`/`.png`/`.webp`/`.gif` |
| 视频 | `.mp4`/`.mov`/`.mkv`；原文件 Range，不转码，播放取决于浏览器 codec |
| 范围外格式 | SVG、HEIC/HEIF、AVIF、RAW |
| 候选限制 | 集中披露双架构、NAS/物理设备、Safari 真机、升级和供应链等剩余阻断 |

README 中“尚无 React 产品前端、认证或媒体链路”的旧状态已经移除；路线图能力状态改为
Stage 1～4 已完成、Stage 5 候选加固进行中。部署文档不再把已固定的
`foliopath.db`/`cache`/`tmp` 称为未来目标。

## 防漂移检查

`make release-docs-check` 现在由 `make arch-check` 自动包含，并验证：

- README 与部署文档同时保留 candidate/非稳定发布警告；
- README、部署说明与 `internal/media/formats.go` 的八个扩展名一致；
- 不转码及 SVG/HEIC/HEIF/AVIF/RAW 范围外说明存在；
- Compose 继续要求显式镜像、回环端口、只读媒体、只读根、最小权限与固定数据路径；
- `.env.example` 保留完整的 Compose 参数清单；
- Dockerfile 继续声明 candidate、非 root entrypoint；
- 已删除的陈旧实现状态不能重新出现。

## 验证

2026-07-28 本机执行：

- `make release-docs-check`：通过；
- 使用完整候选环境值执行 `docker compose --file compose.yaml config --quiet`：通过；
- `make fmt-check arch-check generate-check lint test test-race test-integration`：通过；
- `make web-check`：25 个测试文件、75 个测试与 Storybook build 通过；
- 受影响 Markdown 相对链接检查：通过；
- `git diff --check`：通过。

## 不被本 Gate 关闭的阻断

- 原生 amd64/arm64 候选结果与最终镜像签署；
- 首个稳定版之后新增 migration 时，对真实前一稳定 digest 复跑升级和配对回滚；
- 代表性 NAS、浏览器/物理设备容量与兼容签署；
- 1 Critical / 8 High 供应链发现及最终风险决定/provenance；
- S5-009 Release Candidate 与 S5-010 稳定 MVP Gate。
