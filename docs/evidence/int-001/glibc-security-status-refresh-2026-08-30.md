# glibc / Debian trixie 状态刷新（2026-08-30）

## 结论

Debian 官方 trixie package 页面在本次复核时仍发布 `libc6 2.41-12+deb13u3`，与 2026-08-29 记录的
受影响 runtime 基座相同；没有出现可直接替换当前 distroless closure 的新 trixie libc6 版本。因此
本次只刷新外部状态，不签 VEX、不接受 ADR-0014、不关闭 `INT-014B/202/203/215/228`。

## 来源与边界

- Debian 官方 [trixie libc6 package](https://packages.debian.org/trixie/libs/libc6) 显示版本
  `2.41-12+deb13u3`，并同时列出 amd64 与 arm64 架构包。
- Debian 官方 [trixie glibc source package](https://packages.debian.org/source/trixie/glibc) 同样显示
  `2.41-12+deb13u3` source revision。

网页状态是时间敏感输入；最终 release scan 必须以构建时的 digest-pinned image、离线漏洞数据库、完整
SBOM 和签署 VEX 为准。本记录不把网页抓取当作最终镜像证据。
