# Stage 0 软件供应链与许可证审查

## 状态

**状态：Passed（Stage 0 可生成性与关键许可证识别范围）**

**验证日期：2026-07-23**

## 可重复入口

```sh
scripts/generate-sbom.sh IMAGE OUTPUT_DIRECTORY
```

入口固定 Syft `v1.44.0` 镜像 digest，并生成三个 SPDX 2.3 文档：

| 文件 | 输入 | 本机包数量 |
| --- | --- | ---: |
| `source.spdx.json` | 仓库目录；Go module 与锁文件 | 40 |
| `npm.spdx.json` | `web/package-lock.json`，npm 原生 SBOM | 37 |
| `image.spdx.json` | FS-05 本机 arm64 image | 339 |

生成物是 CI artifact/evidence，不提交到源码树；每次构建记录 SHA-256，发布时应把平台 SBOM
作为 OCI attestation 随 digest 发布。Stage 0 只证明清单可生成和关键依赖可追踪，不等于
漏洞或法律审查已经永久完成。

[CI run 29990480565](https://github.com/HappyQuQu/foliopath/actions/runs/29990480565)
在 Linux runner 上重新构建 FS-05 镜像、生成并校验三份 SPDX，并记录 codec build 与 Debian
copyright；全部 jobs 已通过，构成 S0-108 的正式证据。

## 关键依赖结论

| 依赖 | 证据 | Stage 0 判断 |
| --- | --- | --- |
| FolioPath Go module | source SBOM、`go.mod`/`go.sum` | 可追踪；发布前继续漏洞与许可证策略检查 |
| Web npm | npm SPDX、lockfile、high-severity audit | 可追踪；当前 CI high-severity audit 通过 |
| govips 2.18.0 | 隔离 module 与 upstream license | MIT；生产整合时必须进入最终 binary/image SBOM |
| libvips42 8.14.1 | Debian copyright、image SBOM | LGPL-2.1；必须随镜像保留 copyright/notice |
| FFmpeg 5.1.9 | `-buildconf`、Debian copyright、image SBOM | 构建启用 GPL、libx264、libx265、libwebp；分发按 GPL 组合处理，不得标成纯 LGPL |
| Debian runtime | image SBOM 与固定 base digest | 包较多且镜像约 206 MB；RC 必须做平台漏洞扫描和 notice/source-offer 审查 |

本轮没有发现要求立即更换核心技术的许可证冲突，但 FFmpeg 的 GPL 构建意味着发布流程必须
提供完整许可证/notice，并由发布/合规负责人确认对应源码提供义务。此结论不是法律意见。

## 剩余 Release Gate

- 对最终 amd64/arm64 digest 生成并附加 SBOM/provenance；
- 执行 OS、Go 与 npm 漏洞扫描，按严重度策略阻断；
- 汇总第三方 notices、许可证文本与必要源码/构建脚本；
- 对最终 LGPL FFmpeg 组合和 govips/libvips 动态链接方式完成发布签署；
- 任何 apt 包、codec、base digest 变化都必须重新生成证据。

## Stage 5 更新

2026-07-28 的生产候选复审见
[S5-007 候选镜像供应链 Gate](gates/MVP-2026-07-23/s5-supply-chain-candidate.md)。
该复审升级了 Go、`x/image` 和 Debian 候选基础，并把确定性生产候选 SPDX、固定
digest Trivy、双架构第三方许可证 notices 与 artifact 接入 CI；S5-007C/D/E/F 又以
固定源码构建、可由 SBOM 识别并保留许可证的
`foliopath-libvips` 8.16.1-1 与 `foliopath-ffmpeg` 7.1.5-2 替换通用媒体闭包。FFmpeg
7.1.5-2 在原视频 probe/poster allowlist 上追加 storyboard 所需的 PNG 编解码、`image2`
demuxer、`setsar` 与 `xstack`，并显式链接 PNG 编解码所需的 zlib；它仍禁用网络和
自动探测，不扩大支持的原媒体格式。
最终 FFmpeg 构建关闭 GPL/x264 和网络，许可证为 LGPL 2.1+；内建 readiness probe
同时替换了生产 curl 闭包；固定 digest 的无 shell distroless final stage 保留实际运行
包的 `status.d` 元数据。候选流程还会生成绑定不可变镜像 digest、干净 Git commit、
Dockerfile digest、目标架构和构建调用的 in-toto/SLSA provenance；生成器对 dirty
worktree fail closed，因此最终 provenance 要等已接受的 Stage 5 改动提交并从该干净
commit 重建后才能签发。发现由
15 Critical / 136 High 降为 1 Critical / 8 High。剩余发现尚未处置，因此不改变
本文 Stage 0 Passed 的历史范围，也不构成 Release Candidate 通过。
