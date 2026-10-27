# CR-2026-053：可自助部署的双容器 AI

2026-10-27 用户授权实现。目标 POST-MVP-5 / P6；FR-INT-017/019/020、NFR-INT-007/009/010；Owner 部署 / app / inference。决策见 ADR-0039。

验收：一份 Compose、无必填 .env；只改路径和端口；固定版本模型自动获取、缓存校验、失败重试；主应用独立启动；保留旧离线配置与逐库 opt-in；主/AI配对标签按双原生证据推广。不得使用真实媒体进行测试，或将工程默认阈值称为准确率证明。

合同：FOLIOPATH_AI_PROFILE=standard-v1 选择镜像内置预期身份，与手工 FOLIOPATH_AI_DEPLOYMENT 互斥。AI 的准备状态只含固定状态、已完成/总文件数及字节数，不返回路径、URL、异常或身份。既有媒体、作业、模型可用性 API 不改变；新增安装行为不自动激活库。

证据与未完成门在本文件末尾维护。

## 本地实现与证据

- 新增简化Compose、固定源自动安装器、standard-v1生成profile及独立主/AI预期身份；保留旧离线入口及所有frozen runtime源文件。
- 后端新增鉴权只读准备状态合同；后端测试通过后接入生产服务设置及合成Storybook状态。
- 双原生发布流程分别构建main/AI各一次，同digest验证后发布配对普通/`-ai`标签，保留attestations；AI推广必须包含真实空缓存准备、取消恢复、离线重启与合成推理证据。
- OpenAPI新增进度接口，同时补齐已有OCR latestJob nullable object类型，使现有lint规则通过；不改变其业务行为。
- 实际ARM64镜像完成186文件／3,459,762,979字节公开源下载与校验、离线重启和真实四类推理：文字/图片640维、2张合成人脸、3个标签、4行OCR、空白OCR。cgroup OOM=0。
- 中英文×明暗×390/768/1280/1920×3种准备状态，共48次浏览器检查；无横向溢出或严重/关键axe问题。
- 验证清单及边界见[机器记录](../evidence/release/2026-10-27-managed-ai-local.json)。Go完整面、集成、runtime E2E、生成/架构/lint、Python安装/发布测试、525项前端测试、Storybook构建通过。OpenAPI lint仅保留既有两个health端点4xx警告。

尚未执行AMD64或双原生CI、公开镜像推广、正式标注质量和24小时容量验收；当前是可验证的开发实现，不把本地tag当成已发布镜像。未更新任何生产实例。
