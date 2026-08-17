# FIX-2026-08-17：派生媒体浏览器缓存版本

- 类型：`FR-MED-011` / `FTR-VID-001` 已批准切片内的例行缺陷修复
- Gate：`VSP-S1 Contract Ready`
- Owner：`internal/api` 的派生媒体 URL 与 HTTP 查询合同
- 影响不变量：派生媒体必须绑定资产身份、源指纹、variant 与 transform 语义

## 问题

grid poster 与 storyboard sprite 的公开 URL 过去只包含 `assetId` 和 `variant`，响应却使用一年
`private, immutable`。实例重建后资产 ID 可能重新分配，同一路径源变化也可能保留资产 ID，浏览器
因而可把旧视频的故事板用于当前视频；禁用浏览器缓存后刷新会暂时恢复。

## 修复

资产响应发出的 ready grid/storyboard URL 增加 128-bit 十六进制不透明 `v`。它由 library ID、
library-relative path、source fingerprint、variant 与 transform version 单向派生，不暴露源元
数据。服务端继续兼容不带 `v` 的旧 URL，但校验新参数格式；强 ETag 与 immutable 缓存策略
保持不变。

## 证据

- API 单元测试固定同一资产 ID 在路径、指纹或 variant 变化时生成不同 URL，并验证版本参数。
- OpenAPI 合同声明可选兼容参数及格式；生成客户端和合同检查保持同步。
