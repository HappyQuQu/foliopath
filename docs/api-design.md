# FolioPath API 设计与契约说明

## 状态

[`api/openapi.yaml`](../api/openapi.yaml) 已建立，并且是请求、响应、状态码、认证边界和生成
类型的权威结构化事实来源。本文保留设计动机、资源边界与人类可读语义；与 OpenAPI 冲突时
必须先停止实现并修复契约或本文，不能让 handler 成为第三个事实来源。Go/HTTP 运行骨架已经
可启动，但当前仍没有认证或其他业务 handler；仓库已有确定性生成的 TypeScript 类型和唯一
Web API client 基础，但它们只是消费者契约边界。因此“认证 Contract Ready”不等于认证 API 已实现。

用户已于 2026-07-23 确认[需求确认清单](requirements-checklist.md)中的全部 A 方案。
单管理员认证、三种搜索范围、格式矩阵、默认排序、扫描取消、不可修改媒体库根路径、分页
上限和缩略图 pending 响应均为当前契约基线；轮询退避等不改变 wire 的参数仍待实现验证。

## 设计目标

- 浏览器只使用媒体库、目录、媒体和任务 ID，不接触任意绝对路径。
- 对大型目录使用稳定、可恢复的游标分页，不返回无界集合。
- 扫描与缩略图等长任务异步执行，HTTP 请求不等待整库完成。
- 原始媒体支持条件请求和 HTTP Range；列表只返回元数据与派生资源 URL。
- 错误对用户可理解、对客户端可判断，同时不泄漏 SQL、宿主机路径或媒体工具输出。

## 通用约定

### URL 与传输

- 业务接口统一以 `/api/v1` 开头。
- JSON 字段使用 `camelCase`；时间使用 UTC RFC 3339 字符串；持续时间使用整数毫秒。
- ID 是不透明字符串。客户端不得解析、排序或从 ID 推导路径。
- 列表默认 `limit=50`、最大 `200`；越界值以 `400` 拒绝，不能静默 clamp。
- 修改请求使用 `Content-Type: application/json`。不支持通过 query string 修改状态。
- 健康检查使用 `/health/live` 和 `/health/ready`，不放入版本化业务 API。

### 成功与错误

单资源直接返回资源对象，集合统一返回：

```json
{
  "items": [],
  "nextCursor": null
}
```

错误统一返回：

```json
{
  "error": {
    "code": "library_path_overlap",
    "message": "所选目录与现有媒体库重叠。",
    "requestId": "req_..."
  }
}
```

- `code` 是稳定的机器可读值；`message` 可本地化，不用于客户端分支。
- MVP 错误对象只允许 `code`、`message`、`requestId`；不得通过任意扩展字段回传内部上下文。
- 校验失败使用 `400` 或 `422`，未认证 `401`，无权限 `403`，不存在 `404`，状态冲突 `409`，限流 `429`。
- 意外错误使用 `500`；响应不包含堆栈、SQL、容器路径、原始 stderr 或秘密。

### 并发与重试

- 创建扫描任务时，若同一媒体库已有排队或运行任务，服务以 `200` 返回现有任务；新任务以
  `202` 返回。不得悄悄启动并行完整扫描。
- `POST /libraries` 要求幂等键，服务端至少保留 24 小时，避免页面重试创建重复媒体库。
- 设置更新、媒体库改名和移除使用强 `ETag`/`If-Match` 防止覆盖并发修改；缺失返回 `428`，
  过期返回 `412`。
- `429` 和暂时不可用响应可带 `Retry-After`。

## 资源模型

### Library

建议公开字段：

```json
{
  "id": "lib_...",
  "name": "家庭照片",
  "rootPath": "family",
  "displayPath": "/library/family",
  "status": "ready",
  "lastSuccessfulScanAt": "2026-07-23T12:00:00Z",
  "latestScanId": "scan_...",
  "assetCount": 12034,
  "directoryCount": 481,
  "createdAt": "2026-07-23T10:00:00Z",
  "updatedAt": "2026-07-23T12:00:00Z"
}
```

`rootPath` 是相对于 `/library` 的规范路径，不是宿主机路径；空字符串唯一表示 `/library` 本身。
`displayPath` 只是经过服务端构造的容器内允许路径标签，管理界面可显示，媒体内容端点不接受它作为
文件参数。asset/directory 的路径始终相对于具体媒体库根，宿主机路径永不进入 API。统计可以是最近
成功扫描后的快照，并应明确是否正在更新。

### Directory

```json
{
  "id": "dir_...",
  "libraryId": "lib_...",
  "parentId": null,
  "name": "2026",
  "relativePath": "2026",
  "directAssetCount": 25,
  "recursiveAssetCount": 932,
  "hasChildren": true
}
```

目录响应可以附带 `breadcrumbs`，但面包屑元素只包含 ID、名称和相对路径，不返回真实容器路径。

### Asset

```json
{
  "id": "asset_...",
  "libraryId": "lib_...",
  "libraryName": "家庭照片",
  "directoryId": "dir_...",
  "name": "IMG_0001.jpg",
  "relativePath": "2026/Shanghai/IMG_0001.jpg",
  "kind": "image",
  "mimeType": "image/jpeg",
  "sizeBytes": 3145728,
  "width": 4032,
  "height": 3024,
  "durationMs": null,
  "modifiedAt": "2026-07-20T08:30:00Z",
  "probeStatus": "ready",
  "playbackStatus": "not_applicable",
  "sourceAvailability": "available",
  "thumbnail": {
    "status": "ready",
    "url": "/api/v1/assets/asset_.../thumbnail?variant=grid",
    "errorCode": null
  }
}
```

MVP 的日期语义是文件修改时间；不提供完整 EXIF 面板。未可靠探测出的尺寸、时长等字段返回 `null`，不能用虚构值代替。

### Scan run

```json
{
  "id": "scan_...",
  "libraryId": "lib_...",
  "trigger": "manual",
  "status": "running",
  "phase": "indexing",
  "generation": 3,
  "discoveredDirectories": 128,
  "discoveredAssets": 3812,
  "processedAssets": 902,
  "skippedDirectories": 1,
  "skippedFiles": 3,
  "errorCount": 2,
  "issues": [],
  "progressRatio": null,
  "createdAt": "2026-07-23T11:59:59Z",
  "startedAt": "2026-07-23T12:00:00Z",
  "finishedAt": null,
  "cancelRequestedAt": null,
  "canCancel": true
}
```

目录总量通常事先未知，因此 UI 不应假装存在准确百分比。服务可以返回计数和阶段；只有确有可靠分母时才返回 `progressRatio`。

## 端点草案

### 系统与设置

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/status` | 返回版本、初始化状态和安全的能力标志 |
| `GET` | `/api/v1/settings` | 获取用户可调整的应用设置 |
| `PATCH` | `/api/v1/settings` | 更新 schema 已知且当前主体有权修改的设置 |

设置至少包括 24 小时默认完整扫描周期（可修改或关闭）、10 GiB 默认缩略图缓存配额和中英语言偏好。

### 初始化、认证与会话

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/auth/status` | 返回是否需要首次初始化，不泄露账号信息 |
| `POST` | `/api/v1/auth/setup` | 仅未初始化实例可原子创建唯一管理员并建立会话 |
| `POST` | `/api/v1/auth/login` | 验证管理员凭据并建立安全 Cookie 会话 |
| `GET` | `/api/v1/auth/session` | 返回当前管理员的安全会话摘要 |
| `POST` | `/api/v1/auth/logout` | 撤销当前会话并清理 Cookie |

初始化完成后 `setup` 永久关闭并安全失败。除 `auth/status`、`auth/setup`、`auth/login` 和健康检查外，所有业务端点都要求有效会话；状态修改还要求会话绑定的 `X-CSRF-Token`。首次初始化和登录尚无会话令牌，因此必须校验同源 `Origin`。Cookie wire 名称和安全属性已由 OpenAPI 固定；S1-103 已固定 7 天服务端绝对期限、每次认证整体轮换、摘要存储、退出撤销和 24 小时过期记录宽限。

S1-104 已实现上述 HTTP 边界：匿名白名单同时匹配方法和路径，其余 `/api/v1`（包括未知
业务路由）先认证；非 GET/HEAD/OPTIONS 请求再校验 session-bound CSRF。Origin 按实际
请求 scheme、完整 host 和有效端口比较，不接受缺失、`null`、userinfo、path 或多值；
当前只使用真实 TLS 与直连 peer，不信任 `Forwarded`/`X-Forwarded-*`。setup/login 每个
直连 peer 每分钟最多 10 次，status/session 每分钟 120 次，logout 每分钟 60 次；限流
bucket 最多 4096 个并在满载时失败关闭。可信代理解析与发布网络拓扑仍由 Stage 5 固定。
认证架构边界见 [ADR-0005](adr/0005-built-in-single-admin-auth.md)。

认证端点按状态码声明稳定 `x-error-codes`；未知账号与错误密码统一为
`invalid_credentials`。setup、login、session、logout 与认证状态成功响应，以及统一 JSON
错误响应，都要求 `Cache-Control: no-store`。用户名 setup 保持已冻结的 Unicode 输入范围；
服务保存 NFKC 显示值，并使用 NFKC 后 Unicode full case folding 的 `username_key` 登录，
避免 handler、service 和 SQLite 各自实现不同的比较规则。完整冻结证据见
[认证 Contract Ready](gates/MVP-2026-07-23/s1-auth-contract-ready.md)。

### 允许目录选择器

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/library-paths?parent={relativePath}` | 列出 `/library` 内某一允许目录的直接子目录 |

语义：

- `parent` 使用相对于 `/library` 的路径；根目录用空值，不接受 `/etc`、`..` 或 URL 编码绕过。
- Linux 服务端每次从已锚定的 `/library` 根 FD 解析，并以 `openat2` 原子拒绝越界、
  符号链接和后代 mount crossing；不使用 realpath 后再打开的双阶段替代方案。
- 响应只返回目录名、相对路径、可选择状态和安全原因码，不返回普通文件或宿主机路径。
- 已被媒体库占用、会造成祖先/后代重叠或不可读的目录必须标记不可选；创建接口仍需重复校验，不能信任选择器结果。
- 超大单层目录需要分页；游标和排序规则与普通列表相同。

### 媒体库

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/libraries` | 列出当前主体可见的媒体库 |
| `POST` | `/api/v1/libraries` | 以实例内唯一名称和 `/library` 相对根路径创建媒体库并排队首次扫描 |
| `GET` | `/api/v1/libraries/{libraryId}` | 获取媒体库、状态和最近扫描摘要 |
| `PATCH` | `/api/v1/libraries/{libraryId}` | 修改唯一名称；MVP 请求体不接受根路径字段 |
| `DELETE` | `/api/v1/libraries/{libraryId}` | 删除配置、索引、任务和派生缓存，不触碰原文件 |

删除请求必须经过 UI 明确确认。API 返回已接受的清理任务或完成结果；大缓存清理不应让 HTTP 请求长时间阻塞。

### 扫描

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/libraries/{libraryId}/scans` | 请求完整扫描 |
| `GET` | `/api/v1/libraries/{libraryId}/scans?cursor=...` | 查看扫描历史 |
| `GET` | `/api/v1/scans/{scanId}` | 轮询一个扫描的阶段、计数和安全错误摘要 |
| `POST` | `/api/v1/scans/{scanId}/cancel` | 请求协作式取消，非立即强杀；保留可靠索引与安全提交的新增记录 |

MVP 使用条件轮询或普通轮询，不引入 WebSocket。SSE 只有在实测轮询造成问题且通过 ADR 接受后才增加。

### 目录与媒体列表

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/libraries/{libraryId}/directories?parentId=...&cursor=...` | 获取直接子目录 |
| `GET` | `/api/v1/directories/{directoryId}` | 获取目录与面包屑 |
| `GET` | `/api/v1/libraries/{libraryId}/assets?directoryId=...` | 浏览当前目录或递归范围 |
| `GET` | `/api/v1/assets?q=...` | 跨全部媒体库搜索；限制当前库或目录时使用库内端点 |

默认搜索当前媒体库；UI 可用库内端点限制当前目录（并通过 `recursive` 决定后代），或使用全局端点搜索全部媒体库。授权、排序与游标必须编码准确的多库作用域。

媒体列表契约支持：

- `recursive=true|false`
- `q={query}`
- `kind=image|animated|video`
- `sort=name|modifiedAt` 与 `order=asc|desc`；普通目录默认自然名称升序，递归与搜索默认修改时间倒序
- `cursor` 与 `limit`

查询必须基于索引完成，不能在请求路径中现场递归文件系统。每种排序都以稳定唯一 ID 作为最后比较项。目录不存在或媒体库离线时，应区分“索引中无此目录”和“当前原文件不可访问”。

### 媒体详情与内容

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/assets/{assetId}` | 获取媒体详情和派生状态 |
| `GET` | `/api/v1/assets/{assetId}/thumbnail?variant=grid` | 返回允许的缩略图变体 |
| `GET` | `/api/v1/assets/{assetId}/content` | 返回原始图片或浏览器兼容视频 |

内容响应要求：

- 通过媒体 ID 查索引后由 `internal/files` 安全打开，不能接收客户端文件路径。
- 只有 JPEG、PNG、WebP、GIF 和 MP4、MOV、MKV 属于 MVP 媒体契约；视频容器被索引不等于其编码可由浏览器直接播放。
- 支持 `ETag`、`Last-Modified`、`If-None-Match`/`If-Modified-Since` 和准确的单范围
  Range；多范围、畸形或不可满足范围统一返回 `416`。
- 正确处理 `206`、`416`、`HEAD`、客户端取消和离线源文件。
- 设置准确的 `Content-Type`、`Content-Length`、`Accept-Ranges`、`X-Content-Type-Options: nosniff` 和安全内容处置。
- SVG、HTML 等主动内容不作为可信同源页面直接内联。
- 缩略图未生成时返回 `202`、结构化 `ThumbnailPending` 和 `Retry-After`，不返回伪装成图片的
  占位字节；前端使用统一的本地占位状态并按退避规则轮询。

## 游标规则

- 游标是不透明、带版本且可校验的编码，至少包含当前排序值、稳定 ID 和查询语义指纹。
- 改变媒体库、目录、递归、搜索、过滤或排序参数后，旧游标无效。
- 无下一页时 `nextCursor` 为 `null`。
- 游标格式错误返回稳定的 `invalid_cursor`，不能回退到第一页造成重复列表。
- 索引在翻页期间变化时允许最终一致，但不能因为只按非唯一字段排序而永久漏项或死循环。

## 安全与隐私

- 除受限初始化/登录和健康端点外，所有状态修改和媒体读取都必须套用单管理员会话中间件；状态修改还必须通过 CSRF 防护。
- Path picker 是受限的服务端资源浏览器，不是任意文件管理 API。
- 对列表、搜索、扫描启动、缩略图和内容读取设置独立并发或速率上限。
- 日志使用请求 ID、资源 ID 和必要的安全相对路径摘要，不记录令牌或宿主机路径。
- 错误详情、扫描错误与媒体元数据按不可信文本输出并正确编码。

## 第一版已固定决策与剩余实现参数

OpenAPI 第一版已经固定：

1. 业务端点使用 Cookie 会话；状态修改同时要求 CSRF，setup/login 使用同源 `Origin`。
2. 列表默认 `limit=50`、最大 `200`；游标必须版本化、完整性保护、绑定查询指纹，错误游标
   返回 `invalid_cursor`，不得回退第一页。
3. 新扫描返回 `202`，重复排队/运行请求合并并以 `200` 返回现有扫描。
4. 媒体库移除返回 `202` 和可轮询的 `LibraryRemoval`，大缓存清理不阻塞请求。
5. 缩略图未就绪返回 `202` JSON 与 `Retry-After`。
6. 原媒体支持完整响应或单一 Range，以及 `200`、`206`、`304`、`416`；多段、畸形和
   不可满足 Range 都返回 `416`。

认证 HTTP/数据契约已在 `S1-101` 固定。仍需在后续实现任务固定并验证的内部参数包括：
密码哈希算法与成本、会话绝对期限、
登录限流阈值、可信代理清单、轮询退避曲线、自然排序键、游标签名密钥轮换和缓存安全余量。
这些参数不得改变已固定 wire 行为；若必须改变外部契约，先更新 OpenAPI、契约测试和受影响
Gate，再实现 handler。生产 handler 与生成客户端不得反向改写本说明。
