# S4-005 原媒体内容实现完成

## 结论

**Implemented — 允许进入 S4-005B，不是 Backend Ready。**

资产 ID 到原媒体的生产读取链路已经接入 capability、SQLite、`internal/files`、HTTP 与
composition。S4-005B 仍须完成真实认证 HTTP、Linux `openat2` 路径边界、并发/取消、离线/
损坏/变化源和完整 Range 矩阵；在该 Gate 前，产品前端不得把 content operation 当作已交付能力。

## 实现范围

- 目标版本：`MVP-2026-07-23`
- Requirement：`FR-MED-004～006`、`NFR-SAFE-001`、`NFR-SEC-001`、
  `NFR-PRIV-001`、`NFR-PERF-001`
- Capability owner：`internal/media`
- Filesystem boundary：`internal/files`
- SQLite adapter：`internal/store/sqlite`
- HTTP adapter：`internal/api`
- Composition：`internal/app`
- Contract：`api/openapi.yaml`
- 风险：R-006、R-008、R-012、R-016

## 唯一所有权与数据流

1. HTTP 只接受 `ast_<id>`，不接受路径或 query；SQLite 解析可靠索引中的 library root 和
   library-relative path。
2. `internal/media.ContentService` 校验受支持格式、MIME、size/mtime fingerprint 和 offline
   状态；打开前后的源变化不会被当作可靠内容交付。
3. `internal/app.mediaRootService` 只组合 capability port，实际打开继续唯一委托
   `internal/files.Root.Open`，Linux 发布边界保持 `openat2` fail-closed。
4. HTTP adapter 唯一拥有条件请求与单 Range 语义；最多 16 个内容流同时占用读取槽位。

## 已固定行为

- 完整 GET：`200`，带精确长度、MIME、强 ETag、Last-Modified、Accept-Ranges 和安全 inline
  disposition。
- HEAD：返回完整表示元数据且忽略 Range，不发送 body。
- 条件请求：If-None-Match 优先于 If-Modified-Since；If-Range 不匹配时返回完整 `200`。
- Range：只允许一个 closed/open/suffix byte range；多段、畸形、溢出、空源范围和不可满足
  范围返回结构化 `416` 与 `Content-Range: bytes */<length>`。
- 只服务 JPEG、PNG、WebP、GIF、MP4、MOV、MKV；文件名经安全 header 编码，不暴露库根或
  宿主机路径。
- 离线返回 `source_offline`；索引源已变化或缺失/不可读分别收敛为稳定的 409 错误，不泄露
  SQLite、绝对路径或底层 filesystem 错误。

## 本 Gate 自动证据

- `internal/media`：受支持源、强 ETag、offline、fingerprint 变化和 MIME/format 状态校验。
- `internal/api`：完整 GET、HEAD、closed/open/suffix Range、304、If-Range fallback、
  query 拒绝和结构化 416。
- `internal/store/sqlite` 与 `internal/app`：repository/production composition 可编译并由
  现有 package 测试覆盖。
- OpenAPI 生成 client 已同步 `400` 响应和稳定错误码集合。

## S4-005B 尚未关闭

- 经真实 session middleware 的 full/HEAD/206/304/416 integration。
- Linux `openat2` traversal、symlink、nested mount、root replacement 和 fail-closed 矩阵。
- 请求取消、并发读取槽耗尽、文件在 stat/stream 边界变化、空文件和超大文件矩阵。
- offline、missing、unreadable、corrupt supported media 的最终安全错误与日志审计。
- 完整 race、integration、E2E、双架构 CI 汇总及媒体内容 Backend Ready 记录。

## Gate 判断

- S4-005 implementation：**完成**。
- S4-005B correctness/security audit：**Go**。
- 媒体内容 Backend Ready、前端集成、发布：**No-Go，等待 S4-005B**。

- 评审日期：2026-07-27
