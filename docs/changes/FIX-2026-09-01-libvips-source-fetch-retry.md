# 固定 libvips 源包获取抗瞬断

- 日期：2026-09-01
- Slice：MVP maintenance / POST-MVP-5 native evidence prerequisite
- Gate：[S5 最小媒体运行时](../gates/MVP-2026-07-23/s5-minimal-media-runtime.md)、
  [S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)
- Owner：release image / supply-chain
- 影响不变量：构建只消费固定官方来源和固定 SHA-256；下载失败不得产生可用构建产物。

## 问题

`make test-libvips` 两次在 BuildKit remote `ADD` 获取固定 libvips 8.16.1 tarball 时以
`unexpected EOF` 终止。BuildKit 的 checksum 能拒绝损坏内容，但该入口没有有界重试，短暂传输中断会让
本地和原生双架构 evidence workflow 在进入编译/测试前失败。

## 修复

libvips build stage 使用 curl 的 30 秒连接超时、300 秒单次总时限、最多五次请求、渐增有界退避和
`--continue-at -` 断点续传获取同一官方 URL。下载后必须由 `sha256sum --check --strict` 匹配原 SHA-256
`d114d7c132ec5b45f116d654e17bb4af84561e3041183cd4bfd79abfb85cf724`，之后才允许解包。curl 与 CA
证书只存在于 build stage，不进入运行镜像。

该变更不修改 libvips 版本、源码字节、编译选项、运行闭包、SBOM component 或许可证。重试耗尽、HTTP
失败、超时、截断或 hash 不符继续失败关闭。

## 验证

- 架构 fitness 固定重试参数和精确源包 hash；
- `make arch-check` 通过；
- `make test-libvips` 通过：第一次请求在 300 秒时下载至
  `14,381,561 / 29,544,884` 字节后超时，第二次请求从断点继续并完成；
- `/tmp/vips.tar.xz: OK` 证明最终字节精确匹配固定 SHA-256；libvips 8.16.1 随后完成编译、安装，
  `go test -count=1 -tags=libvips ./internal/media/imagevips` 通过。
