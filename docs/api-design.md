# FolioPath API 设计草案

## 状态

本文是编码前的 REST API 提案，不代表接口已经实现。项目脚手架和 `api/openapi.yaml` 尚未创建；进入实现后，OpenAPI 文件将成为请求、响应和生成类型的结构化事实来源，本文保留资源边界与语义说明。

产品未决项统一记录在[需求确认清单](requirements-checklist.md)。认证方式、全局搜索范围和准确的支持格式确认后，接口需同步收敛。

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
- 列表的默认与最大 `limit` 在性能 spike 后确定；服务端可以把超大值收敛到最大值。
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
    "details": {
      "conflictingLibraryId": "lib_..."
    },
    "requestId": "req_..."
  }
}
```

- `code` 是稳定的机器可读值；`message` 可本地化，不用于客户端分支。
- `details` 只包含公开、安全且对修复有帮助的数据。
- 校验失败使用 `400` 或 `422`，未认证 `401`，无权限 `403`，不存在 `404`，状态冲突 `409`，限流 `429`。
- 意外错误使用 `500`；响应不包含堆栈、SQL、容器路径、原始 stderr 或秘密。

### 并发与重试

- 创建扫描任务时，若同一媒体库已有排队或运行任务，服务返回现有任务或 `409`，最终行为需在 OpenAPI 固定；不得悄悄启动并行完整扫描。
- `POST /libraries` 应支持一个短期幂等键，避免页面重试创建重复媒体库；保留期限待实现验证。
- 更新媒体库使用版本字段或 `If-Match` 防止覆盖并发修改，具体机制在 OpenAPI 前确定。
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

`rootPath` 是相对于 `/library` 的规范路径，不是宿主机路径。统计可以是最近成功扫描后的快照，并应明确是否正在更新。

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
  "thumbnail": {
    "status": "ready",
    "url": "/api/v1/assets/asset_.../thumbnail?variant=grid"
  }
}
```

拍摄时间、EXIF 和更丰富媒体信息是否进入 MVP 由需求确认决定；未可靠探测出的字段返回 `null`，不能用虚构值代替。

### Scan run

```json
{
  "id": "scan_...",
  "libraryId": "lib_...",
  "status": "running",
  "phase": "indexing",
  "discoveredDirectories": 128,
  "discoveredAssets": 3812,
  "processedAssets": 902,
  "errorCount": 2,
  "startedAt": "2026-07-23T12:00:00Z",
  "finishedAt": null,
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

认证与首次管理员初始化端点等待认证方案确认后补充；不能先发布无保护的“创建管理员”接口。

### 允许目录选择器

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/library-paths?parent={relativePath}` | 列出 `/library` 内某一允许目录的直接子目录 |

语义：

- `parent` 使用相对于 `/library` 的路径；根目录用空值，不接受 `/etc`、`..` 或 URL 编码绕过。
- 服务端每次解析真实路径并验证仍在 `/library` 内；列表不跟随目录符号链接。
- 响应只返回目录名、相对路径、可选择状态和安全原因码，不返回普通文件或宿主机路径。
- 已被媒体库占用、会造成祖先/后代重叠或不可读的目录必须标记不可选；创建接口仍需重复校验，不能信任选择器结果。
- 超大单层目录需要分页；游标和排序规则与普通列表相同。

### 媒体库

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/libraries` | 列出当前主体可见的媒体库 |
| `POST` | `/api/v1/libraries` | 以名称和 `/library` 相对根路径创建媒体库并排队首次扫描 |
| `GET` | `/api/v1/libraries/{libraryId}` | 获取媒体库、状态和最近扫描摘要 |
| `PATCH` | `/api/v1/libraries/{libraryId}` | 修改名称及明确允许的设置；是否能修改根路径由 RQ-012 决定 |
| `DELETE` | `/api/v1/libraries/{libraryId}` | 删除配置、索引、任务和派生缓存，不触碰原文件 |

删除请求必须经过 UI 明确确认。API 返回已接受的清理任务或完成结果；大缓存清理不应让 HTTP 请求长时间阻塞。

### 扫描

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/libraries/{libraryId}/scans` | 请求完整扫描 |
| `GET` | `/api/v1/libraries/{libraryId}/scans?cursor=...` | 查看扫描历史 |
| `GET` | `/api/v1/scans/{scanId}` | 轮询一个扫描的阶段、计数和安全错误摘要 |
| `POST` | `/api/v1/scans/{scanId}/cancel` | 若 RQ-013 开放取消，请求协作式取消，非立即强杀 |

MVP 使用条件轮询或普通轮询，不引入 WebSocket。SSE 只有在实测轮询造成问题且通过 ADR 接受后才增加。

### 目录与媒体列表

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/libraries/{libraryId}/directories?parentId=...&cursor=...` | 获取直接子目录 |
| `GET` | `/api/v1/directories/{directoryId}` | 获取目录与面包屑 |
| `GET` | `/api/v1/libraries/{libraryId}/assets?directoryId=...` | 浏览当前目录或递归范围 |

若 RQ-005 确认 MVP 支持跨媒体库搜索，另提供 `GET /api/v1/assets?q=...&libraryId=...`；不要用虚构的媒体库 ID 复用库内端点。授权、排序与游标必须覆盖多库作用域。

媒体列表建议支持：

- `recursive=true|false`
- `q={query}`（是否仅库内搜索待确认）
- `kind=image|animated|video`
- `sort=name|modifiedAt|size` 与 `order=asc|desc`
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
- 支持 `ETag` 或 `Last-Modified`、`If-None-Match`/`If-Modified-Since` 和单范围 Range；准确范围支持以实现测试为准。
- 正确处理 `206`、`416`、`HEAD`、客户端取消和离线源文件。
- 设置准确的 `Content-Type`、`Content-Length`、`Accept-Ranges`、`X-Content-Type-Options: nosniff` 和安全内容处置。
- SVG、HTML 等主动内容不作为可信同源页面直接内联。
- 缩略图未生成时可以返回 `202` 加安全占位状态，或同步排队后返回占位响应；最终策略需用前端体验 spike 确定并固定在 OpenAPI。

## 游标规则

- 游标是不透明、带版本且可校验的编码，至少包含当前排序值、稳定 ID 和查询语义指纹。
- 改变媒体库、目录、递归、搜索、过滤或排序参数后，旧游标无效。
- 无下一页时 `nextCursor` 为 `null`。
- 游标格式错误返回稳定的 `invalid_cursor`，不能回退到第一页造成重复列表。
- 索引在翻页期间变化时允许最终一致，但不能因为只按非唯一字段排序而永久漏项或死循环。

## 安全与隐私

- 所有状态修改和媒体读取都必须套用最终确认的认证与授权中间件。
- Path picker 是受限的服务端资源浏览器，不是任意文件管理 API。
- 对列表、搜索、扫描启动、缩略图和内容读取设置独立并发或速率上限。
- 日志使用请求 ID、资源 ID 和必要的安全相对路径摘要，不记录令牌或宿主机路径。
- 错误详情、扫描错误与媒体元数据按不可信文本输出并正确编码。

## OpenAPI 落地门槛

创建 `api/openapi.yaml` 前至少确认：

1. MVP 的认证/初始化方式及匿名访问边界。
2. 支持格式和浏览器无法直接播放视频时的产品行为。
3. 搜索是仅当前媒体库、可选多库还是默认全局。
4. 日期字段使用文件修改时间还是拍摄时间，以及默认排序。
5. 扫描状态采用轮询的刷新与退避规则。
6. 删除媒体库、修改根路径和取消扫描的最终异步语义。

确认后先编写 OpenAPI 与契约测试，再实现 handler；不要从前后端临时代码反向猜接口。
