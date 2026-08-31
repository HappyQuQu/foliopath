# glibc / Debian trixie 状态刷新（2026-08-31）

## 结论

Debian 官方 trixie package 页面本次复核仍发布 `libc6 2.41-12+deb13u3`；没有出现可用于重建当前
双架构 distroless runtime closure 的新 trixie libc6 revision。因此供应链 blocker 不变，本记录不签
VEX、不接受 ADR-0014，也不关闭 `INT-014B/202/203/215/228`。

## 来源与边界

- Debian 官方 [trixie libc6 package](https://packages.debian.org/trixie/libs/libc6) 仍显示
  `2.41-12+deb13u3`。
- Debian 官方 [trixie amd64 libc6-dev package](https://packages.debian.org/trixie/amd64/libc6-dev) 也仍绑定
  `2.41-12+deb13u3` source revision。

网页状态只用于判断是否值得重建复扫。最终 release 判断仍必须来自 digest-pinned 双架构镜像、完整
SBOM、构建时漏洞数据库及签署 VEX/provenance；网页抓取不能替代这些证据。
