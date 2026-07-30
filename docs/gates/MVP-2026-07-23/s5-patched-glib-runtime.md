# S5-007G 修复来源 GLib 运行时切片

## 结论

**Completed locally — 当前 linux/arm64 候选的 High/Critical 扫描为 0，但不解除
`S5-007`、`S5-009` 或 Release Candidate 的 No-Go。**

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-007G`
- 需求 / 风险：`NFR-COMP-001`、`NFR-SEC-001～002` / R-014、R-017
- owner：发布负责人拥有可重复构建与镜像证据；安全负责人拥有最终双架构复扫；
  合规负责人拥有许可证与源码提供义务签署
- 合同：根 `Dockerfile`、`tests/release/glib_cve_2026_58016.c`、
  `scripts/collect-release-notices.sh`、`scripts/generate-supply-chain-evidence.sh`、
  `tests/release/supplychain_evidence`
- 架构影响：不改变单容器、持久化、信任或模块边界；只收窄既有原生运行闭包

## 实现

- GLib 改为从 GNOME 官方 2.88.3 tarball 构建，固定 SHA-256
  `ab24d24e698dfa1e408b7bcdb508f4aafc906185a8b8ce72fdf79bbbdc9b383b`。
- 2.88.3 已包含 `CVE-2026-58010～58015` 的稳定修复；`CVE-2026-58016` 使用上游已合并
  提交 `656ad4582cb1d7a7fa8bafe3ce8aec6aa3c17da0` 与
  `c9da977c178fbfc0e4caf99f9fdf5dc433d6fcc2`，并固定完整补丁 SHA-256。
- 由于稳定分支测试文件上下文不同，构建从固定的上游完整修复补丁中只提取
  `gio/gdbusintrospection.c` 运行时 hunk；独立 C 回归程序对四种恶意嵌套 XML 输入验证
  `G_MARKUP_ERROR_INVALID_CONTENT`，失败会直接中断镜像构建。
- 自建 `foliopath-glib` 2.88.3-1 明确关闭 `libmount` 与 SELinux 集成。最终 `libgio`
  ELF 只依赖自建 GLib/GObject/GModule、zlib 与 libc，不再带入 `libmount1`、
  `libblkid1` 或 `libselinux1`，因此同时移除 `CVE-2026-53615` 的两个包级发现。
- GLib 许可证文本和包状态元数据进入最终镜像及 notices；Syft 能识别
  `foliopath-glib` 2.88.3-1。

## 2026-07-30 本机 linux/arm64 证据

- 候选镜像：`foliopath:s5-glib-audit`
- image ID：`sha256:e65e38dd106af1afb29df1a98379403350696d2d6c83613aba9df6e184b09559`
- 平台 / 大小：`linux/arm64` / `28,726,703` bytes
- `make test-release-image`：使用该已构建镜像通过 SPA、只读根/媒体、MVP 媒体矩阵、
  Compose、代理、恢复和优雅停机完整 smoke
- Trivy `v0.70.0` 固定 digest、`HIGH,CRITICAL`、`all` 策略：`0 Critical / 0 High`
- Syft `v1.44.0` 固定 digest：image SPDX 仅识别 `foliopath-glib 2.88.3-1`，不含
  GLib Debian 包、mount、blkid 或 SELinux 包
- notices 收集与 SHA256 清单通过

上述 image ID 来自 dirty worktree 的实现验证，只能证明切片可行，不能作为最终发布
provenance 或不可变发布 digest。

CI 已增加原生 amd64/arm64 供应链矩阵和失败关闭的成对证据合同。每个平台 artifact
绑定 source commit、实际架构、image digest、SPDX/扫描/notices SHA-256、`all` 策略零
发现、GLib 版本、被移除包计数及 workflow run/attempt；聚合 job 拒绝缺失、跨 run 拼接
或两架构 source/npm SPDX 不一致。接线完成不等于 runner 已成功，实际结果仍由下述 Gate
持有。

## 剩余 Gate

`S5-007B` 仍必须从合并后的干净提交：

1. 分别在原生 linux/amd64 与 linux/arm64 重建最终候选；
2. 对两个最终 digest 生成 SPDX、notices、provenance，并执行 `all` 策略复扫；
3. 复跑相同媒体、Compose、恢复与运行时依赖矩阵；
4. 由安全负责人审阅漏洞结果，由合规负责人完成 GLib/libvips/FFmpeg 动态链接、
   notices 与必要源码/构建脚本的分发签署；
5. 把全部证据绑定到最终不可变镜像 digest。

这些条件完成前，R-017 只能标为“缓解中”，不能关闭；`S5-007`、`S5-009` 和稳定 MVP
继续保持 No-Go。

## 上游依据

- [GNOME GLib 2.88 发布目录](https://download.gnome.org/sources/glib/2.88/)
- [GLib 2.88.1 修复说明](https://download.gnome.org/sources/glib/2.88/glib-2.88.1.news)
- [Debian GLib 漏洞状态](https://security-tracker.debian.org/tracker/source-package/glib2.0)
- [Debian CVE-2026-58016](https://security-tracker.debian.org/tracker/CVE-2026-58016)
- [Debian CVE-2026-53615](https://security-tracker.debian.org/tracker/CVE-2026-53615)
