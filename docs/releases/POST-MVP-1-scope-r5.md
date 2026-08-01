# POST-MVP-1 Scope Manifest — Revision 5

Revision 5 完整继承 [revision 4](POST-MVP-1-scope-r4.md)及其全部安全、非目标和验收约束，
并通过 [CR-2026-014](../changes/CR-2026-014-derived-media-progress.md)追加：

- `FR-MED-013`：媒体库状态页必须把完整扫描与 thumbnail/poster、storyboard 派生进度分开；
- 只读聚合公开 queued/running/succeeded/failed，不暴露逐资产内部队列或文件路径；
- 扫描仍在发现媒体或 storyboard eligibility 尚未稳定时，不显示伪造百分比；
- 该切片不包含 `POST-MVP-3` 通用任务中心的历史、取消、重试、补齐或重建 operation。

- 版本：`POST-MVP-1`
- Scope revision：`5`
- 状态：`Scope Frozen`
- 冻结日期：2026-08-01
- 产品负责人：产品用户
- Scope-budget exception：N/A；既有派生队列的只读可见性
