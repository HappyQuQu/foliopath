# FS-04：容量基线 spike

## 状态

**状态：Conditional（当前扫描/索引子范围通过）**

**验证日期：2026-07-23**

**目标档：四核、4 GiB、约 10,000 个目录与 100,000 个媒体条目**

当前仓库已提供一个显式启用、默认跳过的可重复基准：
`tests/performance/TestCapacityBaseline`。它使用临时合成媒体树和真实 SQLite 文件，贯通
`library → files → scanner → SQLite`，并记录：

- fixture 创建时间；
- fixture 的最大深度、深链分支数和资产落点数；
- 完整 generation 扫描时间；
- 扫描期间索引读取次数、P95 与最大延迟；
- 扫描完成后的目录页与媒体页查询延迟；
- Go heap 的采样峰值；
- checkpoint 后数据库、WAL 与 SHM 的总尺寸；
- 根目录、每个目录的 direct count，以及选定深链每一级的 recursive count 正确性。

本轮只验证当前已经实现的目录遍历、格式扩展名分类、generation 写入、目录计数与基础索引读取。
它不验证尚未实现的媒体探测、缩略图、FTS5 搜索、正式 catalog API、HTTP 并发或前端虚拟化，
因此即使目标规模运行成功也不能把完整 FS-04 标记为 Passed。

## 已执行目标档与结果

目标档已在以下受限环境运行：

- Docker Server 29.5.2，Linux/arm64，kernel `6.12.76-linuxkit`；
- `golang:1.26.4-bookworm`，镜像 digest
  `sha256:b305420a68d0f229d91eb3b3ed9e519fcf2cf5461da4bef997bf927e8c0bfd2b`；
- `--cpus=4 --memory=4g`、`GOMAXPROCS=4`；
- 2 GiB tmpfs 保存测试目录与 SQLite；这不是 NAS HDD/SSD 延迟模型。

当前混合深度/宽度 fixture 的记录结果：

| 指标 | Linux/arm64 完整目标档 |
| --- | ---: |
| 合成目录 | 10,000 |
| 合成资产占位 | 100,000 |
| fixture 最大深度 / 深链分支 | 32 / 8 |
| 资产落点目录 | 9,655 |
| fixture 创建 | 301 ms |
| generation 完整扫描与 finalize | 10,449 ms |
| 扫描期间成功读取 | 417 次 |
| 扫描期间库内媒体页读取 P95 / max | 3,193 µs / 6,062 µs |
| 扫描后目录页读取 P50 / P95 | 37 µs / 144 µs |
| 扫描后媒体页读取 P50 / P95 | 51 µs / 75 µs |
| 采样 Go heap 峰值 | 39,234,160 B |
| checkpoint 后数据库族大小 | 30,793,728 B |

测试验证了 10,001 个目录行（包含媒体库根）、100,000 个资产行、扫描成功状态、扫描期间
读取无错误、根目录 recursive count 等于 100,000、全库 direct count 之和等于 100,000，
而且每个目录的 direct count 都与独立资产分组一致。第一条 32 层深链的每一级也逐行核对
recursive count。媒体页查询按 `library_id, mtime_ns DESC, id DESC` 走当前库内索引；它只
测第一页，还不是正式 keyset catalog API。

另行执行只针对 SQLite finalize、完全不创建宿主文件系统深目录的深链档：

| 环境 | 深度 | 目录 / 资产 | finalize |
| --- | ---: | ---: | ---: |
| Darwin/arm64，`GOMAXPROCS=4` | 1,000 | 1,001 / 1,001 | 67 ms |
| Linux/arm64，四核/4 GiB 容器 | 1,000 | 1,001 / 1,001 | 147 ms |

该深链档为根和每一级目录各放一个资产，核对根 recursive count、最深叶 direct/recursive
count 和全树 direct count 总和。它证明 1,000 层目录拓扑不会触发 SQLite 递归 CTE 上限或
asset×ancestor 展开；它不经过真实文件系统，所以不证明宿主路径长度、遍历或媒体处理性能。

性能修复经历了两次收敛：

1. 初版 finalize 对每个目录用路径前缀重新扫描全部资产，首次目标档在 10 分钟超时；
   同时基准读取漏掉 `library_id`，触发全表排序。
2. 第一轮修复改成 `asset_ancestors` 递归 CTE。它让两层浅树在 8,237～10,785 ms 完成，
   但独立 P1 审计指出复杂度仍是 O(A × depth)，而当时 fixture 最深只有两层，不能形成
   一般容量结论。
3. 当前实现为每个目录在 SQLite TEMP 工作表保存一行拓扑/计数状态，以 500 个叶节点为
   上限批量向父目录归并。资产与目录分组各扫描一次，每条目录边只传播一次；在现有索引下
   可说明为 O(A + D log D)，Go 侧为常数状态，SQLite 临时状态为 O(D)。无叶可处理但仍有
   未归并节点时将其判定为循环并让整个 finalize 事务回滚；跨库/缺失关系和当前代次条目
   指向同库陈旧目录的损坏也在 stale cleanup 前失败关闭。

当前算法继续保留“成功 finalize 才原子清理 stale row、更新全部目录计数和提交 generation”
的一致性边界。原两层浅树数字只保留为问题定位历史，不再作为当前容量接受证据。

以上时间只适用于该 Docker Desktop Linux VM 与 tmpfs，不是发布 SLA；采样值是 Go heap，
不是容器 RSS。结果没有证明真实媒体探测、缩略图、FTS、HTTP 或浏览器并发能力。

## 运行方式

完整目标档：

```sh
FOLIOPATH_CAPACITY=1 GOMAXPROCS=4 \
  go test -timeout=20m -count=1 -run '^TestCapacityBaseline$' -v ./tests/performance
```

快速检查可以显式缩小数据，但结果不得冒充目标容量：

```sh
FOLIOPATH_CAPACITY=1 \
FOLIOPATH_CAPACITY_DIRS=1000 \
FOLIOPATH_CAPACITY_ASSETS=10000 \
GOMAXPROCS=4 \
  go test -timeout=20m -count=1 -run '^TestCapacityBaseline$' -v ./tests/performance
```

独立 1,000 层 SQLite rollup 档：

```sh
FOLIOPATH_CAPACITY=1 \
FOLIOPATH_CAPACITY_DEEP_CHAIN=1000 \
GOMAXPROCS=4 \
  go test -timeout=20m -count=1 \
  -run '^TestDirectoryRollupDeepChainBaseline$' -v ./tests/performance
```

测试输出一行 `FOLIOPATH_CAPACITY_METRICS` JSON。所有目录、媒体占位内容和数据库均在
`t.TempDir()` 中创建，不读取真实媒体库。合成 `.jpg` 只用于测量当前索引路径，不是有效图片
兼容性 fixture；媒体真实性与解码由 FS-03 单独验证。独立深链档输出
`FOLIOPATH_DEEP_ROLLUP_METRICS`，只构造 catalog row，不在宿主文件系统创建 1,000 层路径。

若使用容器模拟目标资源，必须同时记录镜像摘要、平台、CPU/内存限制和存储类型。例如：

```sh
docker run --rm --cpus=4 --memory=4g --tmpfs /tmp:rw,exec,size=2g \
  -e GOMAXPROCS=4 -e FOLIOPATH_CAPACITY=1 \
  -v "$PWD":/workspace -w /workspace \
  golang:1.26.4-bookworm \
  go test -timeout=20m -count=1 -run '^TestCapacityBaseline$' -v ./tests/performance
```

Docker Desktop 的 Linux VM 和 tmpfs 不是代表性 NAS 磁盘。该结果可以验证四核/4 GiB 进程约束
下没有明显无界内存或正确性故障，但不能据此固定真实 HDD、SSD、SMB 或 NFS 的延迟预算。
`exec` 是必需的，因为 Go 会从临时构建目录执行测试二进制；Docker 的默认 tmpfs 选项会以
`permission denied` 终止测试，不能形成有效容量证据。

## 通过条件

完整 FS-04 至少还需要：

1. 在记录清楚的四核、4 GiB 代表性设备和本地可靠文件系统运行目标档；
2. 增加媒体探测与缩略图队列后复测全局并发、RSS、队列深度和磁盘增长；
3. FTS5、自然排序、稳定 keyset catalog API 完成后测量列表与搜索；
4. 扫描、缩略图和 HTTP 浏览并发时记录尾延迟、WAL/checkpoint 与失败行为；
5. 根据实测结果固定可验收预算，未达标时降低并发、缩略图规格或声明支持上限。

在这些条件满足前，FS-04 保持 Conditional，不以单次本机结果承诺产品性能。
