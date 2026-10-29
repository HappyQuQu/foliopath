# ZIP-S1 来源与浏览合同

2026-10-29；CR-2026-055 / ADR-0040；POST-MVP 已接受范围，Owner 为 media / files / scanner / catalog / SQLite / app。

- `ArchiveMember` 保存库内 ZIP 相对路径、包内成员路径、包大小/修改时间与成员 CRC；资产大小/修改时间描述解压后的成员。成员展示路径为 ZIP 路径加成员路径，仅用于派生目录与展示，绝不作为磁盘打开目标。
- `v1:z` 源指纹绑定包版本和成员版本，既有普通文件 `v1` 不变。稳定资产 ID 仍以 library_id 与逻辑 relative_path 唯一；重建缓存不删除收藏或人工标签。
- 原包只能经 `internal/files` 打开；ZIP 清单解析统一校验，单成员展开统一检查实际输出、CRC、取消与包版本，生成可 seek 的请求临时流。失败不返回路径或原始异常。
- 目录的新增公开字段为 `archivePath`（库内相对路径或 null）、`archiveStatus`（null/ready/unreadable/encrypted/unsupported/limit_exceeded）、`coverAssetId`（已生成可用缩略图的成员 ID 或 null）。包根及内部目录都携带来源；包根的 relativePath 等于 archivePath。现有目录/资产端点、分页、认证与 Range 合同复用。
- 可读包成员按父目录优先落原扫描批次。失败包根记录可见状态并保留旧子树；失败/取消全扫描不能清理。仅完整成功清单可删除被移除的成员；成功全扫描可删除已经移除的包。watcher 快速校准遇到 ZIP 请求权威全扫描。
- 封面只引用当前来源的 ready 缩略图，图片加载失败退回 ZIP 图标；首次生成封面期间显示图标，完成后刷新目录可见封面；包内默认名称自然升序，显式排序优先。离线沿用原库状态，包错误给出重新扫描/检查压缩包的用户提示。
- AI 候选、缺失统计和任务准入排除 ZIP；普通字面搜索/收藏/手动标签沿用稳定资产合同，不新增界面或开关。

额度与首版不支持范围以 [ADR-0040](../adr/0040-read-only-zip-images.md) 为准。OpenAPI 为公开字段权威来源；实现证据记录于 [CR-2026-055](../changes/CR-2026-055-zip-image-browsing.md)。
