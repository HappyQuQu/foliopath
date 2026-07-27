# S2-104 媒体候选与增量收敛实现记录

## 结论

**Go — S2-104 实现完成；进入 S2-105。**

本记录确认冻结 MVP 格式候选、source fingerprint、同路径增量 upsert、资产处理计数以及
成功完整扫描后的 stale 收敛已经进入生产扫描链。它不把扫描后端标记为 Backend Ready，
也不提前证明媒体内容有效、派生缩略图可用、完整故障/重启矩阵或目标容量；这些仍由
`S2-105～S2-107` 和 Stage 3 完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-SCN-002`、`FR-SCN-003`、`FR-SCN-004`、`NFR-REL-001`
- Contract Ready：[可靠扫描](s2-scan-contract-ready.md)
- 候选格式与 source fingerprint owner：`internal/media`
- generation 与候选发射 owner：`internal/scanner`
- upsert、计数与 stale cleanup adapter：`internal/store/sqlite`
- 只读文件系统和正式执行链：`internal/files`、`internal/jobs`、`internal/app`

## 已实现行为

- 媒体候选只来自唯一的 `internal/media` 注册表：JPEG、PNG、WebP、GIF、MP4、MOV 和
  MKV；扩展名不区分大小写。SVG、HEIC/HEIF、AVIF、RAW、无扩展名及其他普通文件不进入
  catalog，并计入 skipped files。候选只代表需要索引，不代表内容已经通过媒体探测。
- 每个候选由大小和纳秒修改时间形成版本化 `v1:<size>:<mtime_ns>` source fingerprint。
  它用于派生数据失效，不是内容哈希、重复检测或跨路径身份。
- migration 6 为已有 `assets` 追加非空 fingerprint，并从原有 `size_bytes` 与
  `mtime_ns` 逐行回填；空库和 version 5 升级走同一嵌入式 Goose 链。
- SQLite 以 `(library_id, relative_path)` upsert。同路径未变化时保留 asset ID 和
  fingerprint，只推进 generation；大小或纳秒 mtime 变化时保留 ID 并更新 fingerprint；
  重命名表现为新路径/新 ID。
- 每个成功写入 catalog 的唯一候选同时推进 `discovered_assets` 与
  `processed_assets`。批次事务失败时不会只推进公开计数。
- 只有完整遍历、根 identity 复核和 catalog 关系验证全部成功后，finalize 事务才删除旧
  generation 的资产/目录并发布新 generation 与目录计数。失败代次提交的安全新增或更新
  可以保留，但不能删除未见旧记录。

## 自动约束与证据

- `internal/media/fingerprint_test.go` 固定 fingerprint 版本、大小与纳秒 mtime 的变化
  语义以及非法大小拒绝。
- `internal/store/sqlite/asset_fingerprint_schema_test.go` 从真实 version 5 catalog 升级，
  验证 migration 6 字段与逐行回填。
- `tests/integration/full_scan_test.go` 通过真实文件 walker 与 SQLite 连续扫描，验证未变化
  同路径保留 ID/fingerprint、mtime 变化更新 fingerprint、重命名产生新 ID，并只在成功
  generation 后删除旧路径。
- `internal/app/runtime_integration_test.go` 通过认证 HTTP、production composition、
  creation worker 和文件 SQLite，验证三个候选都写入 canonical fingerprint，且
  discovered/processed 计数一致。
- `tests/architecture/dependencies_test.go` 强制格式注册表与 fingerprint 编码各有唯一 owner，
  scanner 委托格式分类，SQLite adapter 统一派生/persist fingerprint 并拥有 stale SQL。

本切片要求的完整验证入口为：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-race
make test-integration
make test-e2e
```

## 保留限制与交接

- 部分不可读目录、权限变化、取消窗口、lease 到期、启动恢复和强杀矩阵属于 `S2-105`。
- 1 万目录/10 万媒体、跨库公平、深目录与队列压力主档属于 `S2-106`。
- 魔数验证、govips/FFprobe 元数据、损坏媒体、缩略图/封面及 fingerprint 驱动的派生任务
  失效属于 Stage 3；本切片不把扩展名候选宣称为可解码媒体。
- fingerprint 不读取媒体内容，不用于去重，也不承诺在大小和 mtime 均被外部保留时发现
  字节变化；定期完整扫描仍是文件存在与路径层级的正确性基线。
- 禁止声明：扫描 Backend Ready、Stage 3 已授权、正式前端可依赖完整扫描流程或 MVP
  可发布。
- 评审日期：2026-07-27
