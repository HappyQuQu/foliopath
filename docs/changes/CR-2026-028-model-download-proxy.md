# CR-2026-028：模型下载显式代理

- 日期：2026-09-10；授权：用户要求支持 `http://192.168.2.231:7890`。
- 目标：POST-MVP-5 模型获取扩展；FR-INT-017/019、NFR-INT-007/009。
- Owner：app 部署配置 / downloadhttp；合同：[ADR-0017](../adr/0017-explicit-model-download-proxy.md)。
- 不改变浏览器 API、MVP、数据库或安装状态机；不授权模型公开发布。

新增 `FOLIOPATH_MODEL_DOWNLOAD_PROXY` 环境变量，显式 HTTP/HTTPS CONNECT；未配置仍直连。
配置校验拒绝凭据/路径/query/fragment/非法端口，不输出 URL。代理模式 DNS 信任边界由 ADR 明示，
保留目标字面量/本地域名、redirect allowlist、TLS、hash、大小、取消与原子安装。

## 验证

- 通过 `go test ./internal/aimodel/downloadhttp ./internal/app -count=1`。
- 通过 `go test -race ./internal/aimodel/downloadhttp -count=1`、`make arch-check`、`make fmt-check`。
- 通过 `go test ./tests/integration -run TestAIModelDownload -count=1`。
- 合成 CONNECT/TLS 测试覆盖完整下载、目标私网及本地域名、私网重定向、无效证书、取消和非代理
  endpoint 拨号拒绝；默认下载器不继承 HTTPS_PROXY。显式配置和重复/非法配置测试通过。
- server222 初次代理 HEAD 握手失败；随后 curl 经代理获取 GitHub 和固定官方 tokenizer 返回 200，
  TLS verification 为 0。不是永不失败或所有大文件来源均可用的保证。
- 项目 Go 下载器初次 opt-in 验证被 HF → us.aws.cdn.hf.co 的非白名单跳转拒绝；只为这个固定
  tokenizer probe 加入精确 CDN 主机后重跑通过，没有扩大生产下载清单或放开通配跳转。
- 交叉编译 Linux/amd64 的 `modelnetwork` test binary 在 server222 实跑
  `TestExplicitProxyPinnedTokenizer` 通过（1.55s）：798330 bytes，SHA-256
  `1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`。未注册/启用模型。
  显式设置 `FOLIOPATH_TEST_MODEL_PROXY` 才运行该网络 probe，默认测试不访问网络。

当前为源码实现与 server222 独立验证，**尚未构建/替换正式镜像或改 Compose**。无需也未修改 DNS/hosts、
关闭证书验证、读取用户媒体或公开上传模型。固定兼容包托管源、正式推理镜像、完整搜图链路仍待完成。
