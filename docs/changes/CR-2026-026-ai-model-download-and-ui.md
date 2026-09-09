# CR-2026-026：模型下载与智能功能界面优化

- 日期：2026-09-08
- 授权：用户要求“模型在界面下载、优化智能功能界面”
- 目标：POST-MVP-5 模型获取扩展；不扩大 MVP
- 阶段：先合同与后端，再 UI 消费者
- Owner：模型管理 / API / 前端 / 部署
- 需求：FR-INT-015/017/019/020；NFR-INT-007/009/010；INT-301/302/305
- 决策：[ADR-0016](../adr/0016-admin-model-download-catalog.md)

下载列表、按 ID 下载、真实字节进度、取消、重试与服务器保存的最近任务；所有获取通过现有
安装队列和 managed store。界面增加概览与设置引导，优先显示功能状态、模型和媒体库处理，
将文件路径说明与清理操作收进次级区域；复用当前 tokens/primitives，中英文与窄屏同步。

合同：`api/openapi.yaml`、`internal/aimodel/download.go`、现有 model install / operation 合同。
配置清单来自部署管理员，不宣称项目默认已经运营下载源。runtime 缺失、来源未配置、已下载但未
激活分别显示。实现后补充真实验证记录；历史 S4 未验证事项不转为通过。

## 范围与兼容性

用户明确请求此项 POST-MVP 扩展，预算独立于历史 S0～S4 主任务，不挪入冻结 MVP，也不恢复用户已
关闭的 S4 取证工作。增加两个 v1 endpoint 与下载 DTO，不删除/重命名原 endpoint，不改变媒体、
认证、离线导入或 activation 的 HTTP 请求结构。managed install 的进度从未知总量提升为已接收字节；
其余 operation 保留 item 语义。迁移 36 仅追加最近任务查询索引。OpenAPI 锁在复核加法差异后更新。

下载目录是部署管理员的信任输入，未建立项目托管源运营；当前默认镜像仍无推理组件、人脸 composition
不变。功能代码和合成下载测试不证明真实模型可启用。本轮未部署到 `server222`，不声称线上已更新。

## 2026-09-08 本地验证

- 通过：`make fmt`、`make fmt-check`、`make arch-check`、`make lint`（含完整 API contract/vet）、
  `make test`、`make test-integration`。
- 通过：`GOPROXY=https://goproxy.cn make generate-check`；默认 proxy.golang.org 首次下载 sqlc 返回
  EOF，重试固定相同版本后 SQL/OpenAPI 两项生成检查均通过，没有手工编辑生成文件。
- 通过：`go test -race ./internal/aimodel/... ./internal/files/... ./internal/api/... ./tests/integration/...`。
- 通过：`npm ci`（0 vulnerabilities）、API 生成与一致性、TypeScript、前端架构、OpenAPI lint、
  Vitest **50 files / 187 tests**、Vite production build、Storybook build。
- 视觉核对：在应用内浏览器查看 1440px 中文下载页/整理页/概览、390px 中文与英文深色下载页；
  对照现有 AI 原型保留管理壳、卡片层级与 token，修正窄屏模型行和按钮，页签键盘路径由组件测试覆盖。
  工作台全部使用合成元数据/测试图片；没有真实模型下载、分析或用户媒体的视觉验收声明。
- 未通过环境准入：`make test-e2e` 因本机 Docker daemon 未运行而未执行容器 smoke；没有用其他机器
  或历史结果代替。浏览器自动化全矩阵、native 推理和生产部署未运行。
- 保留警告：OpenAPI 两个既有 health endpoint 缺少 4xx 声明；Vite/Storybook 提示大 chunk。

未修改、提交或推送用户原有的媒体任务 lease 修复。未上传原始媒体、未发布模型或本次代码。

## 2026-09-09 部署跟进

用户授权重新完整构建并更新 server222，现已完成，保留原有 lease 修复。版本与新旧前端哈希、
备份/迁移、合成候选验证及 E2E 网络失败见
[部署一致性修复](FIX-2026-09-09-aifeature-deployment-integrity.md)。
本节更新的是私有部署事实，不改变以上 9 月 8 日本地验证记录或 S4 Release No-Go；模型源和
推理组件仍缺失，不能把新版界面部署解释为真实模型可用。
