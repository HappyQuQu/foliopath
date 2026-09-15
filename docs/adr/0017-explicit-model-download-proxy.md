# ADR-0017：显式模型下载代理

- 日期：2026-09-10；状态：Accepted for implementation
- 授权：用户要求模型下载支持其指定 HTTP 代理。
- 目标：POST-MVP-5 / FR-INT-017/019、NFR-INT-007/009；Owner：app 配置 / downloadhttp。

新增部署环境变量 `FOLIOPATH_MODEL_DOWNLOAD_PROXY`，仅影响模型下载。空值/未设置仍直连，
不自动继承 HTTP_PROXY、HTTPS_PROXY、ALL_PROXY、NO_PROXY，不设代理失败后的直连回退。
接受无认证 HTTP/HTTPS 代理 origin（可含非标准端口、显式私网代理），拒绝凭据、路径、query、fragment。
该设置不进入浏览器 API；启动错误不打印 URL。代理不可用仅使下载失败，不自动重启或改变模型状态。

这是 ADR-0016 出站信任边界的显式扩展：HTTPS 通过 CONNECT，目标域名由受信任代理解析，
因此代理模式不能宣称客户端已验证并锚定目标 DNS 地址。运营方须信任并配置代理的目标地址策略，
拒绝私网/metadata 目标和 DNS rebinding。客户端仍拒绝目标中的私有/保留 IP 字面量及本地域名，
保留固定 catalog、HTTPS 证书校验、重定向主机清单、大小/hash、超时/取消和原子安装。
唯一允许连接的私网出口是显式配置的代理 endpoint；不是允许浏览器提供任意 URL 或代理。

不采用自动继承系统代理（隐式改变所有出口）、禁用证书验证、hosts 固定 CDN 地址、DNS-over-HTTPS
内置服务或未经批准的模型镜像。部署未配置代理时仍保留现有每次解析并锚定 public IP 的边界。

验证：配置校验、真实本地 CONNECT+TLS 合成文件、拒绝目标/重定向/证书错误、代理失败不回退、
取消、现有直连回归，以及用户代理的实测。网络验证不替代真实模型启用或发布准入。
