# CR-2026-029：模型下载代理界面设置

- 授权：用户要求 Docker 环境变量或界面配置代理，并继续优化。
- 目标：POST-MVP-5 / FR-INT-017/019、NFR-INT-007/009，模型获取扩展；不扩大冻结 MVP。
- Owner：aimodel（模式和优先级）、downloadhttp（地址校验和固定探测）、SQLite（持久化）、API、AI settings。
- 合同：[ADR-0018](../adr/0018-admin-model-proxy-settings.md)、OpenAPI model-proxy 三个新增操作、迁移 00037。
- 兼容性：只新增端点和 singleton 表；默认 inherit 保留已有部署代理行为；现有端点不变。
- 发行影响：需要含迁移和前后端的新镜像；不会使模型或 AI Release Gate 自动通过。

界面提供跟随 Docker、直连、自定义模式；仅管理员可操作。配置持久化，revision 冲突拒绝覆盖。
自定义地址不允许凭据，其他模式清除旧地址；Docker 地址不返回浏览器，只显示是否配置。
测试未保存表单只下载固定、带 hash 的参考文件，单实例一个探测、20 秒超时，不保存、不安装。
保存后下一文件连接使用新配置，在途文件不切换；代理失败不回退直连。界面中英文一致。

验证记录：SQLite 集成覆盖默认继承、界面覆盖、直连、重启、revision 冲突和未保存测试；
API 回归覆盖缺字段/null/非法 revision/任意探测目标拒绝和 412/409 映射；
UI 回归覆盖未保存测试、修改清除旧结果、保存请求和直连清除地址。
最终执行通过：make fmt / fmt-check、generate / generate-check、lint（含 arch-check、
contract-check、go vet）、make test；集成测试及 make test-e2e 的应用容器 smoke。
代理相关 aimodel/downloadhttp/API race 测试通过，认证/CSRF 拦截与探测取消释放并发槽有回归。
前端 npm ci、generate:check、check:types、全量 51 文件 / 191 测试、build:storybook 通过。
OpenAPI lint 通过（原有两个 health 4xx 警告）；npm ci 仍报告两个 moderate 审计问题，未强制升级。
浏览器验证本地 Storybook：代理卡片、三种模式、自定义地址输入和按钮状态；不是服务端保存验收。
本记录不表示已部署 server222 或完成真实模型一键安装；未运行全量 Playwright 发布矩阵。
