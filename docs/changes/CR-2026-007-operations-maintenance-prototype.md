# CR-2026-007：任务中心与系统维护交互原型

## 状态

- 状态：Proposal
- 变更等级：C2
- 目标版本：Prototype / Unscheduled
- 阶段：交互原型；不进入生产 import graph
- 提出日期：2026-07-30
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/jobs`、`internal/scanner`、`internal/thumbnail`、
  `internal/settings`、`internal/files` 与 `web` 对应 feature

## 用户问题与价值

管理员需要看见后台任务是否等待、运行、失败或完成，并能在不影响原始媒体的前提下取消、
重试、补齐派生缓存、检查完整性和备份 FolioPath 应用数据。现有 MVP 原型只显示每库扫描与
缓存配额，无法表达任务历史、恢复语义和系统健康闭环。

## 原型范围

- 在既有“扫描与缓存”独立页增加运行概览、任务筛选、补齐缺失缓存、全部重建确认和任务详情。
- 新增“系统维护”独立页，原型化媒体根目录、应用数据空间、完整性和版本健康摘要。
- 原型化缺失文件、未跟踪文件、派生缓存完整性检查。
- 原型化应用数据库与设置备份计划、保留数量、立即备份和脱敏诊断包导出。
- 所有状态存储于浏览器本地，仅用于行为验证；不连接生产 API。

## 明确不包含

- 不备份、移动、修改或删除原始媒体。
- 不增加上传、视频转码、AI 搜索、人脸、OCR、回忆、回收站或多用户能力。
- 不开放每种任务的任意并发数；生产资源策略仍由 `internal/jobs` 的有界全局队列拥有。
- 不把原型中的模拟任务结果、版本号、容量或计时视为后端合同。

## 架构与安全约束

- 生产实现仍须后端优先：先接受 OpenAPI、任务状态、错误语义、备份格式和恢复合同，再接 UI。
- 完整性检查只生成报告；修复只能调用 scanner、thumbnail、jobs 等 canonical owner。
- 离线、部分不可读、失败和取消必须保留最后可靠索引，不能解释为空库。
- 诊断导出必须排除凭据、会话秘密、主机路径和原始媒体高基数元数据。
- 应用数据备份只覆盖 `/app/data` 中不可重建状态；恢复需要独立的版本校验和演练 Gate。

## Gate 与结论

- 原型：Go，用于产品流程、响应式和可访问性验证。
- 生产：Blocked。任务中心后续由
  [FTR-OPS-001](../features/task-center.md)与
  [CR-2026-008](CR-2026-008-task-center.md)按 `POST-MVP-3` scope proposal 管理；
  系统维护、完整性、应用备份和诊断包仍未排期，需各自范围、FR/NFR、OpenAPI、
  数据/恢复合同和后端证据。
- 回滚：删除 `prototypes/apple-redesign/10-settings-maintenance.html`、
  `11-task-detail.html` 及对应原型状态即可；不影响生产代码和媒体数据。
