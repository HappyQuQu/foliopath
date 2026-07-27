# S2-005 媒体库文件系统安全矩阵

## 结论

**Go — S2-005 完成；进入 S2-006。**

媒体库目录选择、创建复检和扫描使用同一 `internal/pathpolicy` 词法策略与
`internal/files` kernel-anchored 文件边界。自动证据覆盖 traversal、重复编码、NUL、
symlink、nested mount、TOCTOU/ABA、重叠、离线、权限失败和公开错误脱敏。

本判断只完成媒体库后端的路径与故障矩阵，不把媒体库标记为 Backend Ready。原媒体不变证明
仍由 `S2-006` 完成，最终 capability/contract/evidence 审计仍由 `S2-007` 完成；扫描 worker
和扫描 Backend Ready 继续由 `S2-102～S2-107` 决定。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-LIB-001～008`、`NFR-SAFE-001`、`NFR-SEC-001～002`、
  `NFR-PRIV-001`、`NFR-REL-001`
- Contract Ready：[媒体库](s2-library-contract-ready.md)
- 路径词法 owner：`internal/pathpolicy`
- 文件系统边界 owner：`internal/files`
- 媒体库规则 owner：`internal/library`
- HTTP/composition：`internal/api`、`internal/app`
- 持久化 adapter：`internal/store/sqlite`
- 架构决策：ADR-0002、ADR-0003、ADR-0004、ADR-0008、ADR-0009

## 故障矩阵

| 风险 | 生产行为 | 自动证据 |
| --- | --- | --- |
| traversal / 歧义编码 | 在 I/O 前拒绝绝对路径、dot component、空 component、反斜杠、NUL、无效 UTF-8，以及单次或多次 percent view 形成的 dot、separator、NUL | `internal/pathpolicy/path_test.go`、`internal/library/library_test.go`、真实认证创建链路 `TestComposedLibraryCreationFailsClosedAcrossUnsafeRoots` |
| symlink escape | picker 标记直接 symlink 不可选；创建、打开和扫描不跟随内部或外部 symlink | `internal/files/enumerate_test.go`、`root_test.go`、`walk_test.go`、`scanner_test.go` 与真实创建链路 |
| nested mount | Linux `openat2` 使用 `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV`；创建复检稳定映射为 `library_root_mount_boundary` | 既有 `/proc` mount 测试、新增 production `mediaRootService` 分类测试，以及带 `fsboundary` 的跨设备、同设备和 self-bind 探针 |
| TOCTOU / ABA | 扫描在已打开 allowed-root fd 下捕获库根 identity；walk 前绑定同一 identity，结束前再验证。替换根或 A→B→A 不能获得 stale cleanup 资格 | `internal/files/root_test.go`、`scanner_test.go`、`tests/integration/full_scan_test.go` |
| 重叠根 | canonical component 级相同/祖先/后代检查由 `internal/library` 唯一拥有，SQLite immediate 事务内再次原子检查 | `internal/library/library_test.go`、SQLite 并发测试、真实认证创建链路的 same/descendant/allowed-root overlap |
| 离线 / 根替换 | 缺失、不可读、身份替换或边界失效返回 offline/unavailable，不伪装为空，不清除最后可靠 generation | `internal/files/root_test.go`、`scanner_test.go`、`tests/integration/full_scan_test.go`、SQLite scanner tests |
| 权限失败 | 文件边界把 `EACCES`/`EPERM` 分类为 offline；picker 的直接子项显示 `unreadable`，当前 parent/创建复检稳定返回 unavailable | `internal/files/unix_test.go` 与确定性的 permission error mapping 测试 |
| 信息披露 | 公开 API 只返回稳定 code/message/requestId；不回显 allowed root、宿主路径、errno 或权限细节 | `internal/api/library_paths_http_test.go`、`libraries_http_test.go` 与真实认证创建链路 |

## 生产链路证据

`TestComposedLibraryCreationFailsClosedAcrossUnsafeRoots` 启动真实 composition root 和文件
SQLite，完成管理员 setup 后，通过 session、Origin、CSRF 与幂等键调用正式
`POST /api/v1/libraries`。它验证：

- traversal、dot、编码/重复编码 traversal、编码 separator、绝对路径、反斜杠和 NUL 均为
  `422 validation_failed`；
- 不存在目录和普通文件均为 `409 library_root_unavailable`，symlink 为
  `409 library_root_symlink`；
- 所有失败请求均未建立数据库记录，随后第一个合法库仍得到 `lib_1`；
- 合法库建立后，相同根、后代根与 allowed root 都稳定返回
  `409 library_path_overlap`；
- 失败响应不包含测试机 allowed root、symlink target、普通文件绝对路径或权限细节。

Linux 默认测试另外从生产 `mediaRootService` 复检已有 `/proc` 后代 mount，证明底层
`EXDEV` 到领域 `ErrRootMountBoundary` 的接线；特权 `fsboundary` 测试继续验证同设备、
跨设备和 self-bind mount，而不依赖 device/inode 差异。

## 已执行验证

本切片实际执行并通过：

```text
go test ./internal/pathpolicy ./internal/files ./internal/library ./internal/api ./internal/app ./tests/integration
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

- Darwin/其他非 Linux adapter 仅提供开发证据；支持平台仍要求 Linux openat2 边界。
- `fsboundary` bind-mount 探针要求隔离环境中的 root 与 `CAP_SYS_ADMIN`，不属于普通本地
  `make test-integration`；CI 负责执行。
- 当前证据不替代 Stage 5 对正式 `/library:ro` volume、后代 mount 检测和 unmount 行为的
  发布验收。
- 下一任务：`S2-006`，对媒体库移除前后的 synthetic media fixture 做逐字节快照/摘要
  对比，并证明只删除 SQLite 与 `/app/data` 派生数据。
- 禁止声明：媒体库 Backend Ready、扫描 Backend Ready、前端已获准连接媒体库 API或 MVP
  可发布。
- 评审日期：2026-07-27
