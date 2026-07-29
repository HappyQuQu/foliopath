# POST-MVP-2 Scope Manifest — Revision 2

## 冻结记录

- 版本：`POST-MVP-2`
- 产品显示标识：`Post-MVP/2`
- Scope revision：`2`
- 状态：`Scope Frozen`
- 冻结日期：2026-07-29
- 基线：[revision 1](POST-MVP-2-scope.md)
- 修订事件：[CR-2026-005](../changes/CR-2026-005-automatic-library-discovery.md)
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers

## 修订决定

Revision 1 的后端自动发现、安全、恢复、容量、部署和数据合同继续有效。Revision 2 只替换
打开页面的消费行为：

- 后端继续自动监听并在数秒内更新派生索引，用户不需要手动发起完整扫描；
- 前端不持续轮询 catalog revision，不增加 SSE 或 WebSocket；
- 浏览页和搜索页提供明确的“刷新”按钮；
- 用户进入或切换目录时重新获取目标目录的第一页、目录状态和媒体库计数；
- 手动刷新必须丢弃旧 cursor 后续页并从第一页建立新链，同时保留 URL、排序、筛选、
  布局、滚动锚点和有效焦点；
- 24 小时完整扫描仍只是正确性兜底，不是看到新增内容的日常操作。

`FR-SCN-014` 替换为：

> 每次成功的定向校准继续递增独立内容 revision；浏览和搜索页面必须提供键盘可用的刷新
> 操作，并在用户进入或切换目录时重新获取目标范围。刷新从第一页建立新 cursor 链，不进行
> 后台持续轮询或推送。

`WCH-AC-001` 中“进入当前页面”改为“进入索引，并在目录导航或点击刷新后显示”。
`WCH-AC-011` 替换为：

> 没有用户导航或刷新操作时不产生 catalog revision 轮询；目录导航和刷新只重新获取当前
> 相关范围，正确重建 cursor 链，并保持滚动、焦点与 URL 语义。

## 影响分析

- 不修改 migration 12、watcher、durable reconcile、content revision 或
  `GET /api/v1/catalog/state` 合同；该 endpoint 保留为已实现的轻量状态合同，但本版本 UI
  不以定时方式消费它。
- 不重新打开 WCH-S0/S1 后端架构与合同；WCH-S2 原生 amd64 阻塞保持不变。
- WCH-S3 必须按本 revision 实现刷新按钮、目录导航重取、中英文、键盘、焦点和 cursor
  回归；不得重新引入定时轮询、SSE 或 WebSocket。
- 相比 revision 1，删除持续前端请求和后台失效风暴风险，代价是已停留页面不会在用户完全
  无操作时自行变化。该权衡由产品用户明确接受。

Revision 1 未被本文件明确替换的全部 FR、NFR、非目标、验收和继承约束继续有效。
