# POST-MVP-1 Scope Manifest — Revision 2

## 冻结记录

- 版本：`POST-MVP-1`
- 产品显示标识：`Post-MVP/1`
- Scope revision：`2`
- 状态：`Scope Frozen`
- 冻结日期：2026-07-30
- 基线事件：[CR-2026-004](../changes/CR-2026-004-video-storyboard-preview.md)、
  [CR-2026-010](../changes/CR-2026-010-avi-and-size-sort.md)
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- 前置版本：`MVP-2026-07-23`；不改变其 scope 或 Release Candidate 结论
- Scope-budget exception：N/A；这是独立后续版本

## 冻结能力

Revision 2 完整继承 [revision 1](POST-MVP-1-scope.md) 的 `FR-MED-009～011`、
`FR-UI-008`、非目标、验收 Gate 和全部产品/安全不变量，并追加：

- `FR-MED-012`：支持 AVI 索引、元数据、poster/storyboard 与原文件 Range；不转码，
  仅浏览器原生兼容 codec 可直接播放。
- `FR-BRW-011`：目录浏览、库内搜索和跨库搜索支持文件大小升序/降序，使用稳定 keyset，
  默认名称/修改时间排序保持不变。
- migration 14 只追加 AVI 格式 CHECK 与大小排序索引，保留既有 catalog、FTS、缩略图和
  media job 状态；不新增业务事实或持久化边界。
- `OPT-AC-001～003`：以
  [CR-2026-010](../changes/CR-2026-010-avi-and-size-sort.md) 的验收定义为准。

## 追加非目标

- `POST-MVP-1-NG-007`：AVI 或其他视频转码、codec 安装与兼容性承诺。
- `POST-MVP-1-NG-008`：按分配块、缩略图缓存或目录累计占用排序。

本 manifest 冻结范围，不表示未经实际执行的验证已经通过。
