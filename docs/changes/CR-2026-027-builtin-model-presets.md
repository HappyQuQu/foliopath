# CR-2026-027：内置模型预设展示

- 日期：2026-09-09；用户授权：优化项目自带模型列表。
- 目标：POST-MVP-5 模型获取扩展；Owner：aimodel / API / AI settings。
- 关联：FR-INT-017/019/020；CR-2026-026；ADR-0016。
- 合同：`api/openapi.yaml` 的 AIModelDownloadPage 新增可选 presets；旧消费者兼容。

后端内置有界的预设元数据，浏览器不从 HF 动态发现模型、不维护另一份生产清单。
预设与可安装下载目录严格分离：SigLIP 1 等待项目兼容包，SigLIP 2、buffalo_l 等待适配。
名称出现不是适配、下载或激活完成；不提供虚构大小、hash、下载按钮或 ready 标识。
所有实际下载仍遵循 ADR-0016 的固定部署清单、精确 hash、安装队列和激活检查。
默认预设现在只提供选型信息，不改变模型授权或 release Gate，不扩大 MVP 或增加 OCR 能力。

界面优先显示有来源的真实下载项；预设候选单列并显示阻塞原因。配置说明移至默认折叠的高级选项。
没有可下载包是项目待交付事项，不再暗示用户必须先手工配置才能获得默认模型。

后续：核验并发布兼容包后，才可加入实际下载目录。当前不是默认一键下载完整交付。

## 本轮验证与限制

- 通过：`go test ./internal/aimodel ./internal/api ./tests/contract`、`make arch-check`、
  前端生成一致性、类型检查、前端架构检查、相关 Vitest 15 项、Storybook build（既有大 chunk 警告）。
- 通过：`npm ci` 后重跑生成一致性、类型检查及上述 15 项测试；`make fmt-check`、`git diff --check`。
  npm 报告仍有 2 项 moderate 依赖漏洞，未使用 force 或绕过 peer 校验。
- 浏览器：启动 loopback 6007 工作台，查看中文模型页及截图，确认三项候选、无候选下载按钮、
  高级配置默认折叠。这是 Storybook 元数据夹具，不是生产 API 或真实模型下载证据。
- 未通过：`go test ./tests/integration -run TestAIModelDownload -count=1` 返回 insufficient_space。
  本机可用约 21 GiB，低于 managed store 保留磁盘容量 10% 的门槛；未降低安全余量或删除用户数据。
- 未执行全套 Go/浏览器 E2E、真实模型下载/推理和 server222 部署。历史 S4 状态不变。
