# FolioPath 版本范围清单

每个冻结版本使用一份只追加的 scope manifest，固定需求、非目标、验收 ID 和修订号。
产品需求文档可以补充说明，但不能通过直接改写它来悄悄改变已冻结版本。

- [MVP-2026-07-23 scope revision 1](MVP-2026-07-23-scope.md)
- [MVP-2026-07-23 scope revision 2](MVP-2026-07-23-scope-r2.md)：加入经认证的局域网
  HTTP 部署，并替代 revision 1 作为当前范围。
- [MVP-2026-07-23 当前 RC readiness 快照](MVP-2026-07-23-rc-readiness.json)：
  聚合 Stage 5 前置 Gate 与发布阻断风险；不改变冻结 scope。

已冻结的后续版本能力：

- [`POST-MVP-1` scope revision 1](POST-MVP-1-scope.md)：冻结
  [FTR-VID-001 视频故事板悬停预览](../features/video-storyboard-preview.md)；
  VSP-S2、VSP-S3 与 VSP-301 已完成，VSP-302 原生双架构候选复验 Pending。
- [`POST-MVP-1` 发布说明草案](POST-MVP-1-release-notes.md)：只记录候选行为和证据，
  在 VSP-302～304 完成前不得改写为已发布。
- [`POST-MVP-1` readiness 快照](POST-MVP-1-readiness.json)：机器校验 VSP Gate、
  `VSP-AC-001～008`、R-018 与最终 Go/No-Go；当前为 No-Go。

已合入的 manifest 不原地改写。范围变化先创建 Change Record；获批后新增 revision 文件或下一版本
manifest，并在新旧文件中链接替代关系。安全不变量不能通过 scope-budget exception 移除。
