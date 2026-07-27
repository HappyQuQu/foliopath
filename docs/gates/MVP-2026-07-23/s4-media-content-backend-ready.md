# S4-005B 原媒体内容 Backend Ready

## 结论

**Backend Ready — Go。**

`GET/HEAD /api/v1/assets/{assetId}/content` 已通过 capability、SQLite、`internal/files`、
认证 HTTP 和真实 composition 的分层验证。冻结 operation 可以交给前端生成 client adapter；
这不表示查看器、浏览器播放矩阵、正式发布镜像或网络暴露已经完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Requirement：`FR-MED-004～006`、`NFR-SAFE-001`、`NFR-SEC-001`、
  `NFR-PRIV-001`、`NFR-PERF-001`
- Capability owner：`internal/media.ContentService`
- HTTP/Range owner：`internal/api/content_http.go`
- Filesystem boundary：`internal/files.Root`
- SQLite adapter：`internal/store/sqlite`
- Composition：`internal/app`
- Contract：`api/openapi.yaml`
- 前置证据：[S4-005 实现记录](s4-media-content.md)
- 风险：R-002、R-006、R-008、R-012、R-016

## 通过的行为矩阵

### 认证与生产纵向链路

- 未认证 GET 在 content service 和文件打开前返回稳定 `401 authentication_required`，不返回
  原媒体字节。
- 真实 session、SQLite、startup reconciliation、生产 route 和 media root 贯通后，受支持
  资产按索引 ID 返回原始字节。
- fixture 故意使用不能解码为 JPEG 的 `.jpg`：内容端点不调用 libvips/FFmpeg，也不把解析
  失败变成任意路径读取；它只按已冻结格式契约安全服务原件。

### HTTP 内容语义

- 完整 GET：`200`、准确 `Content-Length`/MIME、强 ETag、Last-Modified、Accept-Ranges、
  `nosniff` 和不含库路径的 inline disposition。
- HEAD：返回完整表示元数据，忽略 Range 且 body 为空。
- 单 Range：closed/open/suffix 语义由一个 parser 拥有；有效范围返回准确 `206` 和
  Content-Range。
- 多段、畸形、溢出、空文件范围和不可满足范围返回结构化 `416` 与
  `Content-Range: bytes */<length>`。
- If-None-Match 优先于日期条件并返回 `304`；If-Range 失配返回完整 `200`。
- 重复或超长条件头在打开源文件前返回 `400`。

### 安全、离线与变化源

- HTTP 不接受绝对路径、相对路径或 query path；只接受规范 `ast_<id>`。
- production composition 的 poisoned catalog traversal 被 `pathpolicy` 与
  `internal/files` 边界拒绝，响应不泄露 catalog path、宿主机路径、SQLite 或文件错误。
- source fingerprint 与打开后 stat 不一致返回 `409 source_missing`；缺失/不可读源返回
  `409 source_unreadable`；offline library 在文件打开前返回 `409 source_offline`。
- symlink、重复编码、NUL、root identity、跨设备与 nested mount 继续由同一
  `internal/files.Root.Open` owner 失败关闭；content adapter 没有 fallback。
- Linux arm64 容器通过真实 route 与 `/proc` nested-mount 拒绝。当前机器的 amd64 QEMU
  即使关闭默认 seccomp 仍不提供所需 `openat2`，应用在 `OpenRoot` 阶段按设计失败关闭；
  既有 FS-01/native amd64 证据继续有效，仓库 CI billing 恢复后须重跑当前 PR 的 native
  amd64 job。

### 资源与取消

- 每个 route 实例最多 16 个 active content stream；槽耗尽立即返回 `429` 与
  `Retry-After: 1`，不会再打开源文件。
- request context 在 service open 和每个 64 KiB copy 周期检查；取消后停止继续读取并释放
  并发槽，不写新的公开错误 body。
- 空文件完整 GET 返回 `200`/长度 0；任何 byte Range 返回安全 `416`。

## 自动证据

- `internal/api/content_http_test.go`
  - full/HEAD/closed/open/suffix、304、If-Range、416、空文件；
  - header 上限、稳定错误映射、内部错误脱敏；
  - saturated slot、取消释放和 cooperative copy 停止。
- `internal/app/media_content_integration_test.go`
  - 真实认证、SQLite、scanner、production route/files composition；
  - 原字节、HEAD、206/304/416、If-Range；
  - poisoned path、source changed/missing/offline 与响应脱敏。
- `internal/app/media_root_test.go` / `media_root_linux_test.go`
  - production content adapter 的 traversal/编码/NUL 与 nested mount 拒绝。
- `tests/integration/http_content_boundary_test.go`
  - 独立 files/HTTP 边界的 symlink、absolute、encoded traversal 和不泄露回归。
- `tests/architecture/dependencies_test.go`
  - content capability、HTTP parser、SQLite adapter、media root 和 composition 的唯一 owner；
  - 禁止 HTTP 直接导入 files/SQLite/os/filepath 或退回 `http.ServeContent` 多范围实现。

## 剩余边界

- 前端 viewer、视频 browser codec 行为、交互取消和浏览器 E2E 属于产品集成 Gate。
- 正式只读 `/library` volume、运行期 unmount、可信代理、非回环监听、发布镜像 digest、
  native 双架构 CI 重跑属于 Stage 5 Release Gate。
- FolioPath 不锁定或修改外部原件。打开后宿主机就地覆盖可能让客户端遇到短读；响应不会
  回退到另一条路径，后续可靠 full scan 负责重新收敛 fingerprint。

## Gate 判断

- S4-005 implementation：**完成**。
- S4-005B correctness/security audit：**通过**。
- 媒体内容 Backend Ready：**Go**。
- 前端接入冻结 content operation：**Go**。
- 产品集成与发布：**No-Go，等待对应前端/Stage 5 Gate**。

- 评审日期：2026-07-27
