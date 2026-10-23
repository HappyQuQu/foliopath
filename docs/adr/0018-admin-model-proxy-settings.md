# ADR-0018：管理员模型下载代理设置

- 日期：2026-09-10；状态：Accepted for implementation；用户要求 Docker/界面两种配置。
- 目标：POST-MVP-5 / FR-INT-017/019、NFR-INT-007/009；Owner：aimodel、SQLite、API、AI settings。

在 ADR-0017 上增加仅管理员可读写的持久设置：inherit（默认，读取部署环境变量）、direct、custom。
独立 singleton SQLite 表通过同一 write gate 和乐观 revision 更新；启动不重写已有配置。
只有 custom 保存代理 origin；切换其他模式删除旧地址。API 返回生效模式和来源，部署 URL 仅用于
显示是否已配置，不返回部署地址。无凭据代理 URL 校验仍由 downloadhttp 单一 owner 实现。

已认证管理员的显式代理配置是出站信任输入，允许指定私网代理 endpoint，不是模型文件 URL。
保存不拨号；新文件下载读取最新设置，在途文件保持原连接。代理失败不回退，URL 不进入错误日志。
响应 Cache-Control: no-store；PUT 必须提交当前 revision，冲突 412，避免静默覆盖其他管理员窗口。

连接测试 POST 可使用未保存表单，不能指定探测 URL；只取固定官方 HF tokenizer（798330 bytes +
SHA-256），明确允许其已核验 CDN。每实例只允许一个探测，20 秒超时；取消断开，结果仅 available/
unavailable，不泄漏代理/源响应。测试成功不自动保存、不安装模型，不是推理或所有源可用的证明。

新增 GET/PUT /api/v1/ai/model-proxy 和 POST /api/v1/ai/model-proxy/test；沿用会话/CSRF/限流。
前端消费生成合同，配置只作用于模型下载；不改变入站 trusted proxies、其他出站服务或系统网络。
界面明确 Docker 默认、界面覆盖、直连及生效时机，避免隐藏部署配置优先级。发行 Gate 不变。
