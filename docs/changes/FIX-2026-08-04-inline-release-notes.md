# FIX-2026-08-04：关于页直接展示更新内容

## 状态

Accepted / Implemented

## 归属

- 需求：`FR-OPS-009` 关于与版本更新
- 目标：`POST-MVP-3` revision 1 已批准纵向切片
- Owner：`internal/releaseinfo`、`GET /api/v1/releases` 与 `web/src/features/release-info`
- Gate：`CR-2026-015` 关于/发布信息纵向切片

## 问题

关于页的发布记录只显示版本和外部链接，管理员必须离开应用才能看到本次实际更新内容。

## 决定

- 官方稳定 GitHub Release 仍是发布内容来源，后端有界返回完整 Release Markdown。
- 关于页直接按标题、列表和段落展示用户可见内容，隐藏比较链接和提交链接。
- Release 正文缺失时继续显示既有摘要；不改变更新检查缓存、超时或失败降级语义。

## 证据

- `internal/releaseinfo/github/source_test.go`
- `internal/api/release_info_http_test.go`
- `web/src/features/release-info/components/ReleaseNotes.test.tsx`
- OpenAPI 生成检查与 Web 类型检查
