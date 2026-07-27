# S2-006 媒体库移除原媒体不变证明

## 结论

**Go — S2-006 完成；进入 S2-007 Backend Ready 审计。**

真实应用链路已经证明：管理员移除媒体库后，媒体库配置、扫描记录和
`/app/data/cache/libraries/lib_<id>` 派生缓存被清理，terminal removal 仍可轮询，而
allowed media root 中的目录、普通文件内容、文件模式、空文件和 symlink 均与操作前逐项、
逐字节一致。

本记录不把媒体库后端提前标记为 Backend Ready；`S2-007` 仍须对 capability、HTTP、SQLite、
文件系统边界、故障矩阵和合并证据作最终审计。扫描 worker 与扫描 Backend Ready 继续由
`S2-102～S2-107` 决定。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-LIB-007～008`、`NFR-SAFE-001`、`NFR-REL-001`
- Contract Ready：[媒体库](s2-library-contract-ready.md)
- 实现记录：[S2-004](s2-library-lifecycle-implemented.md)
- 安全矩阵：[S2-005](s2-library-safety-matrix.md)
- removal workflow owner：`internal/library`
- SQLite adapter：`internal/store/sqlite`
- application-data cache adapter 与 composition：`internal/app`
- HTTP adapter：`internal/api`
- 架构决策：ADR-0002、ADR-0003、ADR-0004、ADR-0008、ADR-0009

## 真实应用不变性证据

`TestComposedLibraryRemovalPreservesOriginalMediaByteForByte` 启动真实 composition root、
文件 SQLite、认证 session 和后台 removal worker，然后执行：

1. 在临时 allowed media root 创建被移除库、另一个不相关目录、嵌套目录、空目录、二进制
   JPEG/MOV fixture、空文件和相对 symlink；
2. 对完整媒体树记录 allowed-root-relative 路径、文件类型与模式、symlink target，以及每个
   普通文件的完整字节切片；
3. 通过正式 setup、CSRF、幂等键和 `POST /api/v1/libraries` 创建 `lib_1`；
4. 在 `/app/data/cache/libraries/lib_1` 写入可重建缓存 fixture；
5. 使用创建响应的强 ETag 调用正式 `DELETE /api/v1/libraries/lib_1`，再轮询正式
   `GET /api/v1/library-removals/rmv_1` 到 `succeeded`；
6. 验证库详情为 `library_not_found`、派生缓存不存在、terminal removal 可读取；
7. 再次快照媒体树，要求 entry 集合、类型/模式、symlink target 和普通文件完整字节逐项相等。

比较不是仅检查“文件仍存在”，也不是只比较数量或弱 fingerprint；任一删除、改名、类型/
模式改变、symlink target 改变、截断或单字节变化都会使测试失败。

## 中断、重启与能力隔离

- `TestLibraryRemovalResumesIdempotentlyAfterRestart` 在真实文件 SQLite 中领取 removal，以
  batch size 1 完成第一批派生记录清理后关闭 Store；重新打开同一数据库可以再次领取
  `running` removal，幂等完成剩余清理并保留 `succeeded` terminal record。
- 架构 fitness test 固定 `internal/library` removal worker 只依赖
  `RemovalRepository` 与 `DerivedCacheCleaner` 两个窄端口，不允许引入 `os`、
  `path/filepath` 或 `internal/files`。
- application cache cleaner 只能由 `configuration.dataRoot` 组装，并只构造
  `cache/libraries/lib_<id>` 目标；fitness test 禁止它引用 media root、`/library` 或
  `internal/files`。
- SQLite cleanup 只删除 library-scoped assets、directories、scan issues/runs、library
  configuration 和应用幂等/terminal 状态；它没有媒体文件句柄或路径能力。

## 同切片修复

真实链路首次运行时发现：安全 404 wrapper 调用 `ServeMux.Handler` 返回的 handler 后直接
执行，绕过了 `ServeMux.ServeHTTP` 对 `{libraryId}` / `{removalId}` 的 `PathValue` 填充，
导致生产参数化路由误报 404。修复后的 wrapper 仍先检查未知路由并返回统一安全错误，再由
`ServeMux.ServeHTTP` 执行已匹配路由。

`TestRouteFallbackPreservesServeMuxPathValues` 固定该 transport 语义，真实删除测试同时证明
GET、DELETE 和 removal polling 的生产参数化路由完整可用。这是批准切片内的 transport
回归修复，不改变公开 API、架构方向或产品范围。

## 已执行验证

本切片实际执行并通过：

```text
go test ./tests/architecture ./internal/api ./internal/app ./internal/store/sqlite
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

CI 的 Linux amd64/arm64、race、特权 mount boundary、生成契约、容器 runtime/recovery、
媒体矩阵与依赖审计结果由本实现 PR 记录。

## 保留限制与交接

- 当前逐字节证据使用仓库测试创建的临时 synthetic fixtures，绝不读取开发者真实媒体。
- Stage 5 仍须在正式非 root 镜像和真实 `/library:ro` volume 上重复删除 smoke，并验证
  volume/unmount、备份恢复和升级行为。
- 浏览器删除确认、进度/终态展示、错误恢复和可访问性属于后续 Consumer/UI Ready。
- 下一任务：`S2-007`，审计并记录媒体库 Backend Ready；只有该 Gate 通过后，正式媒体库
  前端才能连接这些 API。
- 禁止声明：扫描 Backend Ready、媒体库 UI 已完成、稳定 MVP 可发布。
- 评审日期：2026-07-27
