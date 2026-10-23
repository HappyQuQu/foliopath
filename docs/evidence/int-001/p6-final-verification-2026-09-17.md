# P6 总体验证与发布状态复核（2026-09-17）

## 结论

P6 的三个核对工作包 **G01/G03/G04 已执行完毕**。这表示全量检查、当前供应链材料收集和发布状态判断均已有可复核结果；不表示发行门禁通过。

最终结论为 **Release No-Go**：工程回归通过，但当前候选仍有可修复 High/Critical 漏洞、仅有本地 arm64 镜像、没有可验证的干净源码 provenance，`buffalo_l` 没有商业/再分发授权，而且 B01～B04、E01～E04 的原完整质量与容量门禁仍按用户决定延期。没有代签安全、合规或发布批准。

## 候选边界

- 工作分支：`aifeature`；基准提交：`7d2c05f415467a11f1b8a0d07e6d7731a78335fd`。
- 工作树包含本轮尚未提交的功能和文档改动，因此不能把基准提交当作候选源码身份。
- 主应用：`foliopath-app:p5-delivery`，linux/arm64，镜像 ID `sha256:5629c6119b72c02876734b2ba46d673a4821f223ff3916860d815c8f0c6fac99`。
- AI：`foliopath-inference:p5-delivery`，linux/arm64，镜像 ID `sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689`。
- 本次没有构建或声称存在最终 amd64 配对镜像；没有更新 server222、推送或发布。

## G01：七项全量检查

最终源码状态逐项执行：

| 检查 | 结果 | 说明 |
| --- | --- | --- |
| `make fmt` | 通过 | Go 源码格式化完成 |
| `make arch-check` | 通过 | architecture 测试通过 |
| `make generate-check` | 通过 | SQLC 与 OpenAPI 客户端生成一致 |
| `make lint` | 通过 | 宿主机首次因缺少 `pkg-config/libvips` 失败；随后在本地 Linux/libvips builder 中原命令通过，包括 architecture、contract 和 `go vet ./...` |
| `make test` | 通过 | 本地 Docker、禁外网；全部 Go package 通过 |
| `make test-integration` | 通过 | 本地 Docker、禁外网 |
| `make test-e2e` | 通过 | 首次暴露 `internal/webassets/dist/.gitkeep` 被构建过程删掉导致 embed 目录为空；恢复受版本控制占位文件后，真实镜像构建、两次启动、health、鉴权、只读媒体和重启持久化通过 |

`git diff --check` 通过。宿主机 `make lint` 的缺失工具不记代码通过；结论来自容器内实际完整命令。

## G03：当前供应链复核

### SBOM 与 notices

使用仓库固定的 Syft `v1.44.0` 生成 SPDX 2.3：

| 产物 | 包数 | SHA-256 |
| --- | ---: | --- |
| 应用镜像 SBOM | 50 | `d9e22ce46c56e16b6043408e3b0f87199a49ee681d92052659a9c9823d37801f` |
| AI 镜像 SBOM | 269 | `0f4db784072f0cd73f1ed2a1b55256953b9e347705dbbb7eeb4bc99d148fd401` |
| 源码 SBOM | — | `3b8587cb8b602ca5cb1886c2b9ab8bd5fe7cbf5583815b0a6387789636d8d105` |
| npm SBOM | — | `51ddc38ff69d8e4d3dce4770695750ab4536fd73ed4fc365c1de58a9304b1117` |

应用镜像 notices 收集脚本通过，`SHA256SUMS` 摘要为 `a9b275fed53d54cb61a1e326b92858c6c60f8d8b210527b8096eb762720704e8`。AI 镜像中枚举到 176 个 Debian/Python license、copyright 或 notice 文件，但并非 269 个 SBOM component 的完整合并 notices，因此不能写成 notices 完整。

SBOM、扫描原始文件保存在被 Git 忽略的本地 `build/p6-2026-09-17/`，不作为源码提交；本记录固定其摘要和结论。

### 漏洞扫描

使用仓库固定的 Trivy `0.70.0` 和 2026-09-17 数据库扫描 High/Critical：

| 镜像 | Critical | High | 有修复版本 | 结论 |
| --- | ---: | ---: | ---: | --- |
| 主应用 | 0 | 11 | 11 | No-Go |
| AI | 10 | 99 | 30 | No-Go |

应用发现包括 Go stdlib 1.26.5、`golang.org/x/crypto`、`golang.org/x/image` 和 `libssl3t64` 的可修复项。AI 发现包括 Mesa Critical 可修复项、若干无发行版修复的 Critical，以及 ONNX、Pillow、Starlette、urllib3 等可修复 High。没有编写 VEX 来压低或忽略这些结果。

### 模型来源、哈希和许可

- CLIP 缓存来源 revision：`immich-app/ViT-B-32__openai@a857c8de2c07bbcfa6646adfcf31b798845afa1e`；text ONNX `b80cf0af751533a6712d92247f0ddc0c95208748bc59a1a27f33e67be6864e3b`，visual ONNX `33a3df41ceef21acdf371af00f6dd0456ec1f9eba24d03a7720f9c3734e40859`。该模型仓库只说明是 OpenCLIP 导出，没有给权重级 SPDX license；OpenCLIP 代码的 MIT 许可不能自动替代权重再分发授权。因此保留发布 hold。[模型页](https://huggingface.co/immich-app/ViT-B-32__openai)、[OpenCLIP LICENSE](https://github.com/mlfoundations/open_clip/blob/main/LICENSE)。
- 人脸缓存来源 revision：`immich-app/buffalo_l@d09715916a0778919a770c343533641e250b8699`；detector `5838f7fe053675b1c7a08b633df49e7af5495cee0493c7dcf6697200b85b5b91`，recognizer `6afbf406aa229a439abbca7436cc42be254d4e3af6200d8b7ae4c1fec0c18f2f`。仓库标记 InsightFace license；官方条款明确公共预训练模型（包括 `buffalo_l`）默认仅限非商业研究，商业使用需另行授权。当前缓存没有 `MODEL.LICENSE` 或独立商业/再分发授权，因此不得随正式发行物分发。[模型页](https://huggingface.co/immich-app/buffalo_l)、[InsightFace license](https://github.com/deepinsight/insightface#license)。
- 当前 Compose 只读挂载 operator-owned cache；P5 功能测试不等于模型发布授权。

### Provenance

仓库的 provenance 入口按设计拒绝脏工作树，实际退出码为 2，信息为 `provenance requires a clean source tree`。因此本次没有生成或伪造可验证 provenance，也没有签名、VEX 或外部审批。

## G04：最终发布范围核对

| 范围 | 状态 |
| --- | --- |
| P1～P3 功能闭环 | 已完成并有既有证据 |
| P4 用户确认的小样本工程范围 | 已通过 |
| P5 本地双容器交付 | 5/5 通过 |
| P6 检查与状态核对 | 3/3 已执行完毕 |
| B01～B04 完整模型质量 | 延期，未验证通过 |
| E01～E04 完整容量、双架构、长期稳定性 | 延期，未验证通过 |
| 供应链发行门禁 | 未通过；见漏洞、notices、许可和 provenance 缺口 |
| 发布批准 | **No-Go**；未代签、未发布 |

因此，P6 可以关闭为“审计完成并给出 No-Go 结论”，但不能写成“产品已正式发布就绪”。原 34 个工作包现为 26 个关闭、8 个延期未关闭。
