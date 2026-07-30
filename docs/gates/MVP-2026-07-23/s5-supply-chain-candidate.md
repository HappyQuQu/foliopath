# S5-007 候选镜像供应链 Gate

## 结论

**No-Go — 自动化与候选证据已建立，但 `S5-007` 尚未完成。**

候选镜像现在可重复生成 source、npm、image 三份 SPDX 2.3 SBOM，并使用固定 digest 的
Trivy 扫描全部 High/Critical 漏洞。2026-07-30 的 `S5-007G` 本机 linux/arm64 候选已用
固定来源 GLib 2.88.3、上游 `CVE-2026-58016` 补丁和更小 GIO 闭包，把扫描结果从
`1 Critical / 8 High` 降为 `0 Critical / 0 High`。同一实现已在干净候选提交
`5c3b3c73a1ce32a3777097fb687c707ba914ad41` 上完成原生 linux/arm64 重建、完整 smoke、
SPDX、notices、provenance 与 `all` 复扫；原生 linux/amd64 配对运行和最终安全/合规
签署仍缺失，因此 `S5-007` 仍是 Release Candidate 阻断项。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-007A`、`S5-007B`、`S5-007C`、`S5-007D`、`S5-007E`、
  `S5-007F`、`S5-007G`
- 需求/质量：`NFR-COMP-001`、`NFR-SEC-001～002`
- owner：发布负责人拥有候选镜像与 CI 证据；安全负责人拥有漏洞处置；合规负责人拥有
  notices、许可证与源码提供义务签署
- 合同：根 `Dockerfile`、`go.mod`/`go.sum`、`web/package-lock.json`、
  `scripts/generate-sbom.sh`、`scripts/scan-release-image.sh`、
  `scripts/collect-release-notices.sh`
- 风险：R-014、R-017
- 架构影响：基础发行版由 Debian 12 更新到 Debian 13；没有改变单容器部署单元、
  持久化边界、信任边界或模块依赖方向，因此不新增 ADR

## 已建立的控制与证据

- Go toolchain 固定为 1.26.5，修复 1.26.4 的已公布安全问题；`golang.org/x/image`
  更新到 v0.44.0。
- build 基础更新为固定 digest 的 Go 1.26.5 trixie；最终运行层固定为 Debian 13
  distroless，Debian trixie-slim 只作为枚举、组装运行库和包状态元数据的中间 stage。
  `make test-release-image` 已在本机原生 linux/arm64 重新通过完整 SPA、只读根/媒体、
  MVP 图片/视频、Compose/代理、恢复、强杀、满盘与损坏数据库 smoke。
- Syft 固定为 `v1.44.0` 镜像 digest。源码扫描显式排除 Git 元数据、`build/`、
  `node_modules`、生成前端资产、运行数据库和测试输出；本轮清单包含 source 48、
  npm 381、image 53 个包。
- SPDX 生成器移除工具写入的随机 namespace 与墙钟时间，用内容摘要构造 namespace，
  因此同一镜像和源状态连续两次生成的 source/npm/image SPDX 字节完全一致。
- Trivy 固定为已验证的 `v0.70.0` 多架构镜像 digest，不使用可被移动或覆盖的 action
  tag。完整 JSON、摘要与 SHA-256 由 CI artifact 保留。
- 候选运行层包含 `foliopath-ffmpeg` 7.1.5-1 与 `foliopath-libvips` 8.16.1-1；
  两者均从固定 SHA-256 的官方源码构建为可被 SBOM 识别的包。libvips 仅启用
  JPEG、PNG、WebP、EXIF 和内置 GIF，并随镜像保留
  libvips 与内置 libnsgif 许可证。FFmpeg 仅保留 MOV/MP4/Matroska demux、常见视频
  解码、scale、WebP 海报编码和 file/pipe protocol；关闭网络、设备与其余组件，
  构建许可证为 LGPL 2.1+，对应许可证文本随镜像存在。
- `S5-007C` 的最小媒体运行时已进入根 `Dockerfile`。同一生产 Dockerfile 的完整
  `make test-release-image` 在本机原生 linux/arm64 通过；MVP JPEG/PNG/WebP/GIF、
  MP4/MOV/MKV、只读根/媒体、Compose/代理与恢复失败语义均未回退。运行时 loader
  inventory 不含 ImageMagick、HEIF、OpenEXR、PDF、SVG 或 TIFF loader。
- 与 `S5-007A` 基线相比，包级发现由 151 降至 81、唯一漏洞由 85 降至 47、
  Critical 由 15 降至 8、High 由 136 降至 73。LibRaw、OpenEXR、ImageMagick、
  HEIF 和 Matio 相关发现已随未使用 loader 闭包移除。
- `S5-007D` 又以最小 FFmpeg 替换 Debian 通用 FFmpeg 闭包；生产镜像完整 smoke
  通过，大小由约 206 MB 降至 55,103,419 bytes。包级发现进一步降至 49、唯一漏洞
  降至 32、Critical 降至 5、High 降至 44；Mbed TLS、libxml2 与 Debian FFmpeg
  依赖闭包发现已移除。
- `S5-007E` 以固定 loopback、2 秒超时的 `foliopath healthcheck` 替换生产
  `curl` readiness probe；通用 HTTP 客户端只存在于隔离的测试 sidecar。生产镜像
  完整 smoke 继续通过，大小降至 50,342,686 bytes。包级发现降至 35、唯一漏洞降至
  25、Critical 保持 5、High 降至 30；`curl`、`libcurl4t64` 及其 HTTP/TLS/LDAP
  依赖闭包从生产 SBOM 和漏洞报告中移除。
- `S5-007F` 以固定 digest 的 Debian 13 distroless 替换通用 final stage，并保留实际
  运行包的 `status.d` 与许可证元数据以维持 SBOM/扫描可见性。生产镜像不含 shell、
  curl、tar 或包管理器；完整 release smoke 和 1,000 资产快速容量复验通过，大小降至
  27,472,161 bytes。包级发现降至 15、唯一漏洞降至 14、Critical 降至 1、High 降至
  14；Perl 与其基础层闭包已移除。
- Expat 以官方签名发布的 2.8.2 固定源码和 SHA-256 构建为 `foliopath-expat`
  2.8.2-1，libvips 构建及运行均显式链接它。完整 release smoke 在 arm64 和原生
  amd64 继续通过；原先 6 条 Expat High 全部消除，包级发现降至 9、唯一漏洞降至 8、
  Critical 保持 1、High 降至 8。
- `S5-007G` 以官方 GLib 2.88.3 固定源码和两条固定 SHA-256 的上游
  `CVE-2026-58016` 提交构建 `foliopath-glib` 2.88.3-1；独立恶意 XML 回归程序在构建
  中失败关闭。关闭 `libmount` 与 SELinux 集成同时移除 `libblkid1/libmount1`，本机
  linux/arm64 完整 release smoke、SPDX、notices 和 Trivy `all` 策略通过，结果为
  `0 Critical / 0 High`。证据见
  [修复来源 GLib 运行时切片](s5-patched-glib-runtime.md)。
- 双架构均已生成并校验 image/source/npm SPDX；notices 收集器归档 103 个 Debian
  copyright、自建 Expat/libvips/FFmpeg 许可证和实际 `status.d` 元数据，并生成
  `SHA256SUMS`。这建立了可重复证据，不替代合规负责人的 LGPL 签署。
- `scripts/generate-provenance.sh` 生成 in-toto Statement v1 / SLSA provenance v1，
  绑定镜像 digest、架构、干净 Git commit、Dockerfile SHA-256、builder 与 invocation。
  脚本对 dirty tree 失败关闭，并已接入双架构候选 artifact；提交 `5c3b3c7` 的原生
  linux/arm64 已生成真实 statement，但它仍是单架构候选，不伪装为最终 paired
  statement。

CI 供应链矩阵现在对原生 amd64/arm64 均使用 `all`：任何 High/Critical 发现立即失败。
每个平台生成绑定 commit、架构、image digest、SPDX/扫描/notices 摘要和 workflow
run/attempt 的 JSON；聚合 job 使用 `make verify-supply-chain-evidence` 拒绝缺失架构、
跨 run 拼接、非零发现、GLib 版本或被移除包回退。只有所有发现已消除，或按 `S5-009`
对具体 CVE、包、可达性、版本和期限逐项正式接受并同步修改策略，才可通过。`fixed` 与
`report` 只允许本地调查，不是当前合并或发布策略。

提交 `5c3b3c7` 的原生 linux/arm64 候选 digest 为
`sha256:8a88d26b6579afea21e4d3d85a1df7b5d45b5f851466c4afd6067d025516457d`，
大小 `28,726,829` bytes。机器证据绑定相同 commit/digest，确认完整 release smoke
通过、Trivy `all` 为 `0 Critical / 0 High`、`foliopath-glib` 为 `2.88.3-1`、四个
被禁止运行包计数为零；细节和 artifact SHA-256 见
[修复来源 GLib 运行时切片](s5-patched-glib-runtime.md)。

## 当前阻断

先前候选的 GLib `CVE-2026-58010～58016` 和 util-linux `CVE-2026-53615` 已在
`S5-007G` 本机 linux/arm64 候选中通过升级、固定上游补丁和移除间接依赖处置，没有使用
漏洞忽略规则或风险接受。固定 Trivy `all` 策略得到 `0 Critical / 0 High`，ELF 与 SPDX
也确认最终闭包不含 Debian GLib、blkid、mount 或 SELinux 包。

当前阻断已从“已知漏洞未处置”变为“最终配对证据尚未形成”：干净候选提交的原生
linux/arm64 证据已形成，但原生 linux/amd64 尚未对相同提交完成构建、媒体/恢复矩阵、
SPDX/notices、provenance 和 `all` 策略复扫；安全与 LGPL 分发合规负责人也尚未签署。
因此 R-017 保持“缓解中”，不能标为关闭。

GitHub Actions
[run 30551526321](https://github.com/HappyQuQu/foliopath/actions/runs/30551526321)
已请求原生 amd64/arm64 paired 运行，但两个 job 均因账户付款失败或 spending limit 在
runner 分配前被拒绝，steps 为空，paired job 跳过。该外部阻断不计为测试失败或通过；
恢复 GitHub Actions 计费额度，或使用已授权的原生 amd64 runner，才可补齐配对证据。

`S5-007B` 与 Release Candidate 仍要求：

1. 在最终 linux/amd64 与 linux/arm64 digest 上重复 SBOM 和 `all` 策略扫描；
2. 确认两个最终 digest 的 High/Critical 仍为零；若重新出现发现，必须升级、移除或逐项
   完成具体、限时、可复审的正式风险接受；
3. 汇总第三方 notices、许可证文本、必要源码/构建脚本与 provenance，并完成
   LGPL 动态/静态链接分发审阅；
4. 将每个平台 SBOM/provenance 绑定到最终不可变镜像 digest。

在这些条件满足前，不得把 `S5-007`、`S5-009`、Release Candidate 或稳定 MVP 标为完成。
