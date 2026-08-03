# FolioPath 版本范围清单

每个冻结版本使用一份只追加的 scope manifest，固定需求、非目标、验收 ID 和修订号。
产品需求文档可以补充说明，但不能通过直接改写它来悄悄改变已冻结版本。

- [MVP-2026-07-23 scope revision 1](MVP-2026-07-23-scope.md)
- [MVP-2026-07-23 scope revision 2](MVP-2026-07-23-scope-r2.md)：加入经认证的局域网
  HTTP 部署，并替代 revision 1。
- [MVP-2026-07-23 scope revision 3](MVP-2026-07-23-scope-r3.md)：加入管理中心四个独立
  功能页与单管理员账户维护，并替代 revision 2。
- [MVP-2026-07-23 scope revision 4](MVP-2026-07-23-scope-r4.md)：冻结
  [FTR-UIF-001 生产前端原型一致性](../features/frontend-prototype-fidelity.md)，加入当前
  目录全量过滤、生产视觉合同和阻断式一致性 Gate，并替代 revision 3 作为当前范围。
- [MVP-2026-07-23 当前 RC readiness 快照](MVP-2026-07-23-rc-readiness.json)：
  聚合 Stage 5 前置 Gate 与发布阻断风险；不改变冻结 scope。
- [MVP-2026-07-23 UIF 当前集成状态](MVP-2026-07-23-uif-integration-status.md)：
  只追加聚合 `UIF-401～408` 实际证据、精确视觉矩阵边界和 `UIF-S4` Go；Stage 5 独立
  发布阻断保持 No-Go，不修改 revision 4 manifest 或历史 Gate。

已冻结的后续版本能力：

- [`POST-MVP-1` scope revision 1](POST-MVP-1-scope.md)：冻结
  [FTR-VID-001 视频故事板悬停预览](../features/video-storyboard-preview.md)；
  VSP-S2、VSP-S3 与 VSP-301 已完成，VSP-302 原生双架构候选复验 Pending。
- [`POST-MVP-1` scope revision 2](POST-MVP-1-scope-r2.md)：继承 revision 1，并通过
  [CR-2026-010](../changes/CR-2026-010-avi-and-size-sort.md)追加 AVI 与文件大小排序，
  替代 revision 1 作为当前范围。
- [`POST-MVP-1` scope revision 3](POST-MVP-1-scope-r3.md)：继承 revision 2，并通过
  [CR-2026-011](../changes/CR-2026-011-directory-media-counts.md)追加选中目录媒体类型数量，
  替代 revision 2 作为当前范围。
- [`POST-MVP-1` scope revision 4](POST-MVP-1-scope-r4.md)：继承 revision 3，并通过
  [CR-2026-012](../changes/CR-2026-012-nas-resource-profiles.md)追加 NAS 资源模式。
- [`POST-MVP-1` scope revision 5](POST-MVP-1-scope-r5.md)：继承 revision 4，并通过
  [CR-2026-014](../changes/CR-2026-014-derived-media-progress.md)追加扫描后缩略图与视频预览
  只读生成进度，替代 revision 4 作为当前范围。
- [`POST-MVP-1` 发布说明草案](POST-MVP-1-release-notes.md)：只记录候选行为和证据，
  在 VSP-302～304 完成前不得改写为已发布。
- [`POST-MVP-1` readiness 快照](POST-MVP-1-readiness.json)：机器校验 VSP Gate、
  `VSP-AC-001～008`、R-018 与最终 Go/No-Go；当前为 No-Go。
- [`POST-MVP-2` scope revision 2](POST-MVP-2-scope-r2.md)：当前冻结；继承
  [revision 1](POST-MVP-2-scope.md)的后端自动发现合同，并把页面消费改为目录导航重取和
  手动刷新
  [FTR-SCN-001 媒体库自动发现](../features/automatic-library-discovery.md)；
  WCH-S0 当前只对 Linux watcher spike 与 ADR 评审 Conditional Go。
- [`POST-MVP-3` scope revision 1](POST-MVP-3-scope.md)：冻结后台任务恢复、结构化日志、
  关于/版本更新与全局消息中心；按失败诊断、发布信息、消费者 UI 和统一任务历史分片交付。

已合入的 manifest 不原地改写。范围变化先创建 Change Record；获批后新增 revision 文件或下一版本
manifest，并在新旧文件中链接替代关系。安全不变量不能通过 scope-budget exception 移除。

## 稳定版本发布记录

稳定标签必须使用 `vMAJOR.MINOR.PATCH`。创建标签前必须新增非空的
`docs/releases/vMAJOR.MINOR.PATCH.md`，使用面向用户的“新增、优化、修复、升级注意事项和已知
限制”描述；Docker Hub workflow 在构建前验证该文件，并在镜像成功发布后用同一文件创建
GitHub Release。构建参数写入相同标签，应用关于页和官方 Release 因而共享版本号与更新记录。
