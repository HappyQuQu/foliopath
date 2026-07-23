# S0-109：Stage 0 风险复审

- 日期：2026-07-23
- 范围：R-002～R-016
- 结论：全部风险已有明确 Owner、Fallback、状态和最迟阻断 Gate；没有无主风险

本复审只判断风险是否已被识别并分配到可执行 Gate，不把 spike 证据等同于生产或发布就绪。
`缓解中`和`开放`风险会继续阻断其所属 Gate，但不反向阻断 Stage 1 中不依赖该能力的后端基础。

| 风险 | Stage 0 判断 | Owner | 最迟阻断 Gate |
| --- | --- | --- | --- |
| R-002 路径越界 | 缓解中；FS-01 双架构内核边界通过 | 安全负责人 | 首个受保护文件 API Backend Gate；发布挂载由 Release Gate |
| R-003 扫描误清理 | 缓解中；generation 故障保留通过 | 后端负责人 | 扫描 Backend Ready |
| R-004 SQLite 位于网络存储 | 缓解中；本地盘约束、WAL 与离线恢复已有证据 | 运维负责人 | 扫描 Backend Ready；代表性部署由 Release Gate |
| R-005 容量不足 | 缓解中；Stage 0 扫描/索引基线通过 | 技术负责人 | 按 S0-106 分配到扫描、浏览、搜索、UI、Performance/Release Gate |
| R-006 敌意媒体 | 缓解中；合成损坏输入和双架构链路通过 | 媒体处理负责人 | 媒体处理 Backend Ready |
| R-007 格式/浏览器不一致 | 缓解中；服务端矩阵通过，浏览器未验证 | 产品负责人 | 查看器 UI/Integrated Done |
| R-008 双架构差异 | 缓解中；原生双架构 fixture 与 FS-05 镜像通过 | 发布负责人 | 最终双平台 Release Gate |
| R-009 磁盘耗尽 | 开放；仅 FS-05 失败关闭，不代表缓存策略 | 运维负责人 | 缩略图 Backend Ready；最终 Release Gate |
| R-010 未认证网络暴露 | 开放；Stage 1 认证是共享预览前置条件 | 安全负责人 | 认证 Backend Ready；共享预览与 Release Gate |
| R-011 迁移/恢复丢失 | 缓解中；FS-05 离线恢复、重复迁移与故障关闭通过 | 发布负责人 | 运行骨架 Backend Ready；真实版本升级由 Release Gate |
| R-012 Unicode/大小写差异 | 开放 | 后端负责人 | 媒体库 Backend Ready；搜索 Backend Ready |
| R-013 多库队列饥饿 | 开放 | 技术负责人 | 扫描 Backend Ready；Performance Gate |
| R-014 许可证义务 | 缓解中；三类 SBOM 与 GPL 组合已识别 | 合规负责人 | 最终镜像 Release Gate |
| R-015 瀑布流可访问性 | 开放；默认规则网格可作为 fallback | 前端负责人 | 浏览 UI/Integrated Done |
| R-016 CI/生成漂移 | 缓解中；真实 PR、双架构、契约、runtime 与 SBOM jobs 已建立 | QA 负责人 | 每个切片 Gate 与 Release Gate 持续阻断 |

## 不可降级的系统边界

- 原媒体始终只读，不为进度放宽路径失败关闭或 mount 边界。
- 失败、取消、离线扫描不得清理上一次可靠索引。
- 未认证业务 API 默认拒绝；认证完成前不得发布到不可信网络。
- 数据库仅支持本地可靠文件系统；缓存和派生数据可重建，配置数据不可用清缓存代替恢复。
- amd64/arm64、许可证、SBOM、真实升级和代表性设备结果在 Release Gate 再签署。

## 复审触发

每个切片在 `Architecture Ready` 时复核所列风险，在 `Backend Ready`、`Integrated Done` 或
Release Gate 提交自动化证据。若 Owner、Fallback 或最迟 Gate 发生变化，必须先更新本记录、
风险登记册和追踪矩阵，不能通过新增功能绕过。
