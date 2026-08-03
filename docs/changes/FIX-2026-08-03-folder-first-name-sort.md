# 名称排序先按来源文件夹分组

- 日期：2026-08-03
- 类型：已批准浏览/搜索切片内的例行排序修复
- Requirement：`FR-BRW-009`、`FR-SRH-002`、`FR-SRH-004`
- 目标版本与阶段：MVP，已集成切片维护
- Owner：`internal/catalog`（排序与 cursor）、`internal/store/sqlite`（keyset 查询）
- 既有 Gate：[S3-002 Catalog 排序与游标](../gates/MVP-2026-07-23/s3-catalog-keyset.md)、[S4-002 搜索 keyset](../gates/MVP-2026-07-23/s4-search-keyset.md)
- 不变量：稳定 keyset pagination、URL 排序状态、可靠 generation/revision 绑定、原始媒体只读

## 问题与决定

显式按名称排序原先先比较自然文件名，再用相对路径区分同名项，导致递归或搜索结果中的不同
来源文件夹相互穿插。名称排序 v2 改为先比较媒体库（仅全局 scope）和来源文件夹路径，再在
同一文件夹内比较自然文件名，最后保留相对路径与资源 ID 作为稳定 tie-breaker。升序和倒序均
作用于完整 tuple；直接目录浏览因为所有结果具有同一来源文件夹，顺序保持自然文件名排序。

排序版本从 v1 推进到 v2，因此旧 cursor 会按既有契约返回 `invalid_cursor`，不会误用旧 tuple
继续翻页。此修复不改变默认排序、搜索 profile、API 参数或数据库权威字段。Migration 19
只为同一派生表达式追加覆盖索引，避免递归浏览退化为全量临时排序。

## 回归证据

- `internal/store/sqlite/catalog_test.go`：以每页一项遍历递归名称排序，固定“来源文件夹 →
  自然文件名”的完整次序并覆盖 cursor 链。
