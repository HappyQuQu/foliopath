# FolioPath 验收证据

本目录保存由 Gate 或集成状态引用的可复核证据。证据不是产品或架构规范；发生冲突时，
以冻结 scope、OpenAPI/migration、已接受 ADR 和当前 Gate 为准。

## MVP 发布浏览器证据

- [S5-006B：Chrome 200% 缩放与核心流程](s5-006b/README.md)
- [S5-006C：Firefox 200%/400% 缩放与核心流程](s5-006c/README.md)

## 前端原型一致性

- `uif-301/`：管理中心基础比较
- `uif-312/`：浏览工具栏与目录比较
- `uif-315/`：查看器桌面与移动比较
- `uif-316/`：异步、离线和状态矩阵比较
- [UIF-317：主题、语言和断点矩阵](uif-317/README.md)
- [UIF-401：逐页同状态比较](uif-401/README.md)
- [UIF-402：Linux 视觉回归基线](uif-402/README.md)
- [UIF-403：真实后端纵向链](uif-403/README.md)
- [UIF-404：浏览器与可访问性矩阵](uif-404/README.md)
- [UIF-405：100k/10k 容量](uif-405/README.md)
- [UIF-406：完整仓库验证](uif-406/README.md)
- [UIF-407：文档收敛](uif-407/README.md)
- [UIF-408：最终集成复验](uif-408/README.md)

同一图片内容可能被不同状态或验收 ID 引用。删除、重命名或去重前必须先检查对应 Gate、
readiness manifest 和视觉清单。
