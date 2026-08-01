# FolioPath 架构决策记录

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
