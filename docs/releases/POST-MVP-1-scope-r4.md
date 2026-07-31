# POST-MVP-1 Scope Manifest — Revision 4

Revision 4 完整继承 [revision 3](POST-MVP-1-scope-r3.md)及其全部安全、非目标和验收约束，
并通过 [CR-2026-012](../changes/CR-2026-012-nas-resource-profiles.md)追加：

- `FR-SET-001`：管理员可在 NAS 友好、均衡、性能三个资源模式间选择；
- `NFR-PERF-004`：完整扫描、定向校准和媒体派生共享实例级后台预算，内容读取具有独立上限；
- 降低限制不取消正在运行的工作，持久任务、可靠索引和原媒体只读语义不变；
- 用户设置不得突破既有 worker、libvips 或内容读取硬上限。

- 版本：`POST-MVP-1`
- Scope revision：`4`
- 状态：`Scope Frozen`
- 冻结日期：2026-07-31
- 产品负责人：产品用户
- Scope-budget exception：N/A；独立后续版本
