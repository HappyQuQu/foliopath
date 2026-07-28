# S5-007 候选镜像供应链 Gate

## 结论

**No-Go — 自动化与候选证据已建立，但 `S5-007` 尚未完成。**

候选镜像现在可重复生成 source、npm、image 三份 SPDX 2.3 SBOM，并使用固定 digest 的
Trivy 扫描全部 High/Critical 漏洞。2026-07-28 本机 linux/arm64 生产 Dockerfile
复扫得到 9 条包级发现、8 个唯一漏洞，其中 1 条 Critical、8 条 High；linux/amd64 与
linux/arm64 结果一致，本次数据库中没有可用修复版本。它们没有被忽略或接受，仍是
Release Candidate 阻断项。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-007A`、`S5-007B`、`S5-007C`、`S5-007D`、`S5-007E`、`S5-007F`
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
- 双架构均已生成并校验 image/source/npm SPDX；notices 收集器归档 103 个 Debian
  copyright、自建 Expat/libvips/FFmpeg 许可证和实际 `status.d` 元数据，并生成
  `SHA256SUMS`。这建立了可重复证据，不替代合规负责人的 LGPL 签署。
- `scripts/generate-provenance.sh` 生成 in-toto Statement v1 / SLSA provenance v1，
  绑定镜像 digest、架构、干净 Git commit、Dockerfile SHA-256、builder 与 invocation。
  脚本对 dirty tree 失败关闭，并已接入双架构候选 artifact；当前工作树尚未形成最终
  commit，因此这里只完成 provenance 入口，不伪造最终 statement。

CI 的持续检查策略是 `fixed`：任何已有上游修复版本的 High/Critical 发现立即失败，
同时完整报告所有未修复发现。Release Candidate 的最终策略是 `all`；只有所有发现已
消除，或按 `S5-009` 对具体 CVE、包、可达性、版本和期限逐项正式接受，才可通过。
`report` 只允许本地调查，不是合并或发布策略。

## 当前阻断

剩余发现来自最小运行闭包中的 GLib、blkid 和 mount 原生库。
不能用“暂无修复版本”替代处置。

唯一 Critical 是 `libglib2.0-0t64` 的 `CVE-2026-58016`；GLib 是最小 libvips 的
核心依赖。Debian trixie、forky 和 sid 当前仍标记该 CVE 未修复；Debian 只指向
GLib 2.89.0 开发线的合并请求。`S5-007C/D/E/F` 已证明裁剪未使用的媒体、探针和通用基础层闭包可显著降低
暴露面，但不能关闭 R-017；下一步是跟踪上游修复，并对无法消除的具体 CVE 作正式风险决定。

对最终最小闭包的动态符号检查发现：应用只直接导入 8 个 GLib 符号，libvips 导入
265 个，但应用和 libvips 都不直接导入下表对应的受影响入口。`libblkid1/libmount1`
只由 GLib/GIO 间接带入；FolioPath 不接受块设备或分区表输入，并在
`internal/files` 通过 kernel-anchored 边界拒绝 mount crossing。该结果降低可达性，但
不能证明 GLib 内部不会经其他入口间接到达，因此不会自动改成 accepted：

| CVE | 包 / 机制 | 当前可达性判断 | 候选处置草案 |
| --- | --- | --- | --- |
| CVE-2026-58016 | GLib GDBus introspection XML | 产品不使用 D-Bus，应用/libvips 无受影响符号导入 | Critical；只允许安全负责人逐项决定，最晚 2026-08-11 复审 |
| CVE-2026-58015 | GLib GDBus SHA1 auth / path traversal | 产品不使用 D-Bus auth，应用/libvips 无受影响符号导入 | 等待 Debian stable 修复；最晚 2026-08-11 复审 |
| CVE-2026-58010 | GLib GVariant serialiser | 无受影响符号直接导入，仍可能存在库内间接路径 | 不宣称不可达；等待上游修复或安全负责人接受 |
| CVE-2026-58011 | GLib invalid GDateTime | 无受影响符号直接导入，媒体元数据仍可能间接使用日期 | 不宣称不可达；等待上游修复或安全负责人接受 |
| CVE-2026-58012 | GLib regex replace | 无受影响符号直接导入，文件名/元数据为不可信输入 | 不宣称不可达；等待上游修复或安全负责人接受 |
| CVE-2026-58013 | GLib GIOChannel line read | 无受影响符号直接导入，仍保留本地文件输入 | 不宣称不可达；等待上游修复或安全负责人接受 |
| CVE-2026-58014 | GLib keyfile locale list | modules/loaders 已禁用且无受影响符号直接导入 | 等待上游修复；最晚 2026-08-11 复审 |
| CVE-2026-53615 | util-linux DOS partition parser（两个包级发现） | 产品不接受块设备/分区表，两个库均为 GIO 间接依赖 | 等待 Debian stable 修复；最晚 2026-08-11 复审 |

任何临时接受都必须只覆盖本候选的固定双架构 digest，记录安全负责人姓名、决定日期、
到期日和撤销条件；上游出现稳定修复、可达性假设变化或媒体矩阵出现异常时立即撤销。
当前这些字段尚无授权签署人，因此上表仍是处置草案，不是风险接受。

`S5-007B` 与 Release Candidate 仍要求：

1. 在最终 linux/amd64 与 linux/arm64 digest 上重复 SBOM 和 `all` 策略扫描；
2. 升级、移除或以更小且经过媒体矩阵验证的运行时闭包处置全部发现；
3. 对无法消除的每个发现完成具体、限时、可复审的正式风险接受；
4. 汇总第三方 notices、许可证文本、必要源码/构建脚本与 provenance，并完成
   LGPL 动态/静态链接分发审阅；
5. 将每个平台 SBOM/provenance 绑定到最终不可变镜像 digest。

在这些条件满足前，不得把 `S5-007`、`S5-009`、Release Candidate 或稳定 MVP 标为完成。
