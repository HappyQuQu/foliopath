# FolioPath 架构决策记录

[ADR-0028：显式版本化的大 PNG 语义输入](0028-versioned-png-semantic-input.md)（实现授权，合成验证与原库切换分开验收）。

[ADR-0026：稳定标签类别与独立扩充策略](0026-localized-tag-policy.md)（Accepted for implementation；功能证据见 [CR-2026-033（本地化图片标签）](../changes/CR-2026-033-localized-photo-tags.md)，不关闭质量/发行门禁）。

[ADR-0025：多语言语义模型与能力身份隔离](0025-multilingual-semantic-model.md)（方向接受，实现/本地联验；发行与质量另计）。

[ADR-0024：大图 native 工作内存共享准入](0024-shared-large-image-native-admission.md)（Accepted for implementation；固定合成回归单列，不等于容量/发行通过）。

[ADR-0023：SQLite 在线维护共享同一内核锁域](0023-same-kernel-sqlite-maintenance.md)（Accepted；本地复现/维护路径验收通过，不等于长期或发行通过）。

[ADR-0022：人物扫描有界等待与脱敏失败诊断](0022-bounded-face-scan-wait-and-diagnostics.md)（Accepted；本地工程验证完成，质量/容量/发行未关闭）。

[ADR-0021：有界模型进程与自动标签原子提交](0021-bounded-inference-and-automatic-tags.md)（Accepted for implementation；验证进行中）。

[ADR-0020：FolioPath 专用 AI 镜像](0020-ai-runtime-delivery-proposal.md)：用户已选择方案 A，方向接受、实现与发行验证进行中。

[ADR-0019：独立 AI 服务](0019-independent-machine-learning-service.md)：用户已接受方向，生产集成进行中。

[ADR-0018：管理员模型代理设置](0018-admin-model-proxy-settings.md)（Accepted for implementation）。

ADR 记录已经改变或将改变系统结构的决定。它回答“为什么这样设计”，不替代
[系统架构](../architecture.md)、产品需求、OpenAPI、迁移或测试规范。

## 状态与规则

- `提议`：可评审，尚不能作为实现依据。
- `已接受`：当前实现必须遵守。
- `已弃用`：仍保留历史背景，但不应继续用于新实现。
- `已被替代`：由一份更新 ADR 取代，并互相链接。

已接受 ADR 不重写其历史决定。方向变化时复制[模板](template.md)，使用下一个连续编号，
在新旧记录中标明替代关系。仅修复链接或明显笔误不算重写。

以下变化必须先有 ADR：部署单元、核心技术、信任边界、持久化模型、模块依赖方向、
API 兼容策略、关键事务或任务一致性，以及共享前端架构。

## 当前记录

合并历史中 `ADR-0026` 存在两个独立决定：[原生双架构构建与已验证 digest 发布](0026-native-release-image-promotion.md)
和[稳定标签类别与独立扩充策略](0026-localized-tag-policy.md)。两者保留原编号、文件和历史证据；
引用时必须带主题与完整文件链接，不得仅凭编号推断为同一决定。

- [ADR-0027：计划停机交回分析任务](0027-planned-analysis-job-handoff.md)
- [ADR-0001：Go、React 与 SQLite 模块化单体](0001-go-react-sqlite.md)
- [ADR-0002：单一允许根目录与多媒体库](0002-library-path-model.md)
- [ADR-0003：扫描 generation 一致性](0003-scan-consistency.md)
- [ADR-0004：MVP 媒体库根路径不可变](0004-library-root-immutable.md)
- [ADR-0005：稳定版内建单管理员认证](0005-built-in-single-admin-auth.md)
- [ADR-0006：契约驱动、切片内后端优先交付](0006-contract-driven-backend-first.md)
- [ADR-0007：单一共享前端设计系统](0007-shared-frontend-system.md)
- [ADR-0008：统一应用组合根并分离纯路径策略](0008-composition-root-and-path-policy.md)
- [ADR-0009：Linux `openat2` 与单一媒体根挂载](0009-linux-openat2-single-media-root.md)
- [ADR-0010：经认证的局域网 HTTP 与可选外部 TLS](0010-authenticated-lan-http.md)
- [ADR-0011：Linux 文件事件只触发锚定的定向校准](0011-linux-inotify-hints-and-anchored-reconciliation.md)
- [ADR-0012：默认 root 运行与免初始化 bind 数据目录](0012-root-runtime-bind-data.md)
- [ADR-0026：原生双架构构建与已验证 digest 发布](0026-native-release-image-promotion.md)（发布工具决定；与同号标签策略按主题和文件路径区分）
- [ADR-0013：本地 AI 运行时与可重建向量索引](0013-local-ai-runtime-and-derived-vector-index.md)（A+B 已接受）
- [ADR-0014：SigLIP SentencePiece tokenizer runtime 与模型包 v2](0014-siglip-sentencepiece-tokenizer-runtime.md)（已接受 fail-closed 后端实现；发行链接、模型激活与发布仍等待质量/双架构/供应链 Gate）
- [ADR-0015：人脸组合模型包与独立 generation 激活](0015-face-model-package-and-generation-activation.md)（已接受 fail-closed 后端实现；模型激活与发布仍等待 S2C 质量/隐私/双架构准入）
- [ADR-0016：部署管理员模型下载目录](0016-admin-model-download-catalog.md)（已接受下载与 UI 扩展；不提供默认下载源、不解除 AI Release No-Go）
- [ADR-0017：显式模型下载代理](0017-explicit-model-download-proxy.md)（部署配置；保留 HTTPS/hash 校验，代理承担目标 DNS/IP 策略）
