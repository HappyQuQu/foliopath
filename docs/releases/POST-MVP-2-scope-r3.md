# POST-MVP-2 Scope Revision 3：删除缓存一致性与有界页面刷新

- 日期：2026-08-03
- 状态：Frozen
- Change Record：[CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md)
- Requirement：`FR-SCN-010～014`、`FR-MED-002`、`NFR-REL-002`、`NFR-PERF-003`
- Owner：`internal/scanner`、`internal/thumbnail`、Web catalog-state consumer
- 既有 Gate：`WCH-S0～S4`、`S3-005`

Revision 3 保留 revision 2 的安全 watcher、定向校准、显式刷新和完整扫描兜底，并接受以下
用户可见调整：

1. 可靠定向校准与成功完整 generation 删除索引前，都必须先在同一事务登记对应的 ready
   派生缓存删除；事务提交后主动唤醒缓存清理器。
2. 媒体库管理页显示 `active/degraded/unsupported/disabled` 自动发现状态、净化错误原因和
   最近自动同步时间，并提供明确刷新按钮。
3. 已认证用户停留在媒体库、浏览或搜索页面且页面可见时，客户端每 5 秒对既有
   `GET /api/v1/catalog/state` 发起一次带 ETag 的条件请求。`304` 不重取 catalog；revision
   变化时只保留当前 active infinite query 的第一页并重新获取当前范围。页面隐藏时停止。
4. 启动、手动和默认计划完整扫描继续是正确性基线；watcher、revision 检查和缓存删除队列
   都不能获得修改原媒体的能力。

本 revision 不增加 WebSocket、SSE、新部署单元或全树轮询，也不承诺 SMB/NFS/FUSE 近实时。
历史版本可能遗留的无数据库引用缓存审计不在本次修复内；在建立有界磁盘遍历、宽限期和恢复
证据前，不允许以一次无界目录扫描实现。

## 验收增量

- watcher 删除文件/目录与完整扫描 stale cleanup 都产生幂等 `cache_deletions`。
- failed/cancelled/offline/部分不可读扫描不登记删除，也不清理可靠索引或缓存。
- 缓存清理失败保留 durable 删除记录并可重试；缓存文件已不存在按幂等成功处理。
- 条件检查只在相关可见页面运行，revision 不变不刷新浏览/搜索结果；变化只重建第一页 cursor
  链，并保持 URL、排序、筛选、布局和有效焦点。
- 自动发现状态与错误在中英文、键盘和移动断点下可读，不只依赖颜色。
