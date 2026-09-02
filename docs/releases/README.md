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
- [`POST-MVP-1` scope revision 8](POST-MVP-1-scope-r8.md)：继承 revision 7，并通过
  [CR-2026-018](../changes/CR-2026-018-explicit-resource-limits.md)将三档资源模式简化为
  受后端硬上限约束的两个直接并发数设置，替代 revision 7 作为当前范围。
- [`POST-MVP-1` scope revision 9](POST-MVP-1-scope-r9.md)：继承 revision 8，并通过
  [CR-2026-019](../changes/CR-2026-019-video-preview-default-mute.md)增加与自动播放独立的
  视频预览默认静音偏好，替代 revision 8 作为当前范围。
- [`POST-MVP-1` 发布说明草案](POST-MVP-1-release-notes.md)：只记录候选行为和证据，
  在 VSP-302～304 完成前不得改写为已发布。
- [`POST-MVP-1` readiness 快照](POST-MVP-1-readiness.json)：机器校验 VSP Gate、
  `VSP-AC-001～008`、R-018 与最终 Go/No-Go；当前为 No-Go。
- [`POST-MVP-2` scope revision 3](POST-MVP-2-scope-r3.md)：当前冻结；继承
  [revision 2](POST-MVP-2-scope-r2.md)的后端自动发现合同，并补充完整扫描缓存删除、媒体库
  状态/刷新以及可见相关页面的 5 秒 ETag 条件检查
  [FTR-SCN-001 媒体库自动发现](../features/automatic-library-discovery.md)；
  WCH-S0 当前只对 Linux watcher spike 与 ADR 评审 Conditional Go。
- [`POST-MVP-3` scope revision 1](POST-MVP-3-scope.md)：冻结后台任务恢复、结构化日志、
  关于/版本更新与全局消息中心；按失败诊断、发布信息、消费者 UI 和统一任务历史分片交付。
- [`POST-MVP-4` scope revision 1](POST-MVP-4-scope.md)：冻结
  [FTR-CUR-001 收藏与手动标签](../features/favorites-and-tags.md)；CUR-S3 UI 已 Go，CUR-S4
  当前为 Conditional Go；消费者 UI 已完成，Integrated Done 仍受发布证据阻断。
- [`POST-MVP-5` scope revision 1](POST-MVP-5-scope.md)：冻结模型基础 + 图片语义搜索 A+B、
  `/models:ro` 离线基线和 16 工程周停损；C 标签、D 视频、E 人脸不在 revision 1。INT-S0 对 A+B
  为 Go；[INT-S1 Contract Ready](../gates/POST-MVP-5/int-s1-contract-ready.md)已 Go并曾授权 S2 后端；
  [INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)现为 Backend Ready /
  Release No-Go；S4 Go 前不授权发行 UI。
- [`POST-MVP-5` scope revision 2](POST-MVP-5-scope-r2.md)：替代 revision 1，并按产品用户 2026-08-29
  的明确决定纳入 C 受控标签建议、D 视频代表帧语义搜索与 E 匿名人脸聚类/人物库；接受 32 工程周
  scope-budget exception。S2A/S2B/S2C 已 Backend Ready；外部 Release Gate 与安全门槛不降低。
- [`POST-MVP-5` scope manifest proposal draft 1](POST-MVP-5-scope-proposal.md)：已被 revision 1 替代，
  保留 A～E 完整提议、替代项和收口历史。
- [2026-08-13 综合更新说明草案](2026-08-13-integrated-update-notes.md)：汇总收藏与标签、
  通知/处理结果体验、媒体容错和 4K 大文件故事板适配；只描述已实现候选行为，不声明稳定发布。

已合入的 manifest 不原地改写。范围变化先创建 Change Record；获批后新增 revision 文件或下一版本
manifest，并在新旧文件中链接替代关系。安全不变量不能通过 scope-budget exception 移除。

## 稳定版本发布记录

稳定标签必须使用 `vMAJOR.MINOR.PATCH`，并由 Release Please 的 Release PR 自动创建和
squash merge。
根 [`CHANGELOG.md`](../../CHANGELOG.md) 是面向用户的累计更新记录；自动生成内容按
`✨ 新功能`、`🚀 改进`、`🐛 修复` 与 `⚠️ 注意事项` 分类。提交标题负责提供用户可理解的
中文描述，纯技术提交默认隐藏。合并 Release PR 后，workflow 使用同一版本和日志创建 GitHub
Release，并为 Docker 镜像追加 `MAJOR.MINOR.PATCH`、`MAJOR.MINOR`、`latest` 与 `sha-*`
标签。Release PR 不等待人工批准，因此提交者必须在推送 `main` 前确认措辞与适用发布 Gate；
版本准备和 Docker 发布不包含任何实际实例部署。
