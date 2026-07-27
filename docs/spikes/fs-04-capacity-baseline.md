# FS-04：容量基线 spike

## 状态

**状态：Passed（Stage 0 扫描/索引可行性范围）；完整产品/发布范围仍为 Conditional**

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
- Linux 进程 RSS 高水位（`/proc/self/status` 的 `VmHWM`）；
- checkpoint 后数据库、WAL 与 SHM 的总尺寸；
- 根目录、每个目录的 direct count，以及选定深链每一级的 recursive count 正确性。

本轮只验证当前已经实现的目录遍历、格式扩展名分类、generation 写入、目录计数与基础索引读取。
它不验证尚未实现的媒体探测、缩略图、FTS5 搜索、正式 catalog API、HTTP 并发或前端虚拟化，
因此即使目标规模运行成功也不能把完整 FS-04 标记为 Passed。依据
[S0-106 容量证据 Gate 分配记录](../gates/MVP-2026-07-23/s0-106-capacity-gate-order.md)，
这些生产能力在最早能产生真实证据的 Backend/UI Gate 验证，代表性设备与最终镜像在
Performance/Release Gate 验证；它们不再与禁止提前实现生产功能的 Stage 0 形成循环。

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
不是容器 RSS。后续三档运行已补进程 RSS 高水位，但结果仍没有证明真实媒体探测、缩略图、
FTS、HTTP 或浏览器并发能力。

## 三档趋势、RSS 与暂定预算

同一 `golang:1.26.4-bookworm` 镜像、Linux/arm64、四核、4 GiB、2 GiB tmpfs 环境中，
`make capacity-trend` 为每档启动独立测试进程并强制 `stage0-comparable-v1`：

| 目录 / 资产 | fixture | 扫描/finalize | 并发读取 P95 | 目录/资产页 P95 | Go heap 采样峰值 | 进程 RSS 高水位 | DB 族 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 / 10,000 | 28 ms | 1,079 ms | 468 µs | 27 / 83 µs | 30.4 MiB | 41.4 MiB | 3.0 MiB |
| 5,000 / 50,000 | 142 ms | 6,353 ms | 433 µs | 35 / 66 µs | 37.9 MiB | 50.1 MiB | 14.7 MiB |
| 10,000 / 100,000 | 283 ms | 11,675 ms | 545 µs | 44 / 68 µs | 23.2 MiB | 37.0 MiB | 29.3 MiB |

三档均无预算超限。资产数量扩大十倍时，本次扫描时间约扩大 10.8 倍，DB 族约扩大 9.9 倍，
没有出现数量级恶化。heap 与 RSS 来自三个独立进程，受 GC、allocator 和采样时机影响且不单调；
它们只证明本轮均远低于 4 GiB，不用于拟合每资产内存公式。

`stage0-comparable-v1` 的上限是：扫描 120 秒、扫描期间基础资产页读取 P95 250 ms、
扫描后目录/资产页读取 P95 各 100 ms、Linux 进程 RSS 高水位 1 GiB、checkpoint 后数据库族
1 GiB。它们是相同容器/tmpfs 条件下的宽松回归护栏，不是用户可见 SLA，也不能替代
Performance/Release Gate 在代表性设备上冻结的发布预算。

## 运行方式

### S2-106 持续回归

S2-106 把主档从手动 spike 提升为正式扫描回归：基准使用
`scanner.DefaultBatchSize`（当前 256），`make spike-capacity` 默认强制
`stage0-comparable-v1` 预算；CI 的独立 `scan-capacity` job 在 Linux amd64/arm64、
2 CPU/4 GiB、2 GiB tmpfs 中运行主档与 1,000 层 rollup。托管 runner 只有 2 个 CPU，
因此 CI 是更严格的持续回归护栏，四核目标档仍由明确记录的本地/代表性设备运行证明。
架构检查同时固定唯一 active/batch 常量和 CI 资源参数，避免测试以比生产更宽松的批次或
未受限环境形成假证据。

2026-07-27 本地 Darwin/arm64、`GOMAXPROCS=4`、正式 256 批次结果为：10,000 目录、
100,000 资产，fixture 6,029 ms、扫描 19,221 ms；扫描期间 768 次读取，P95 462 µs、
max 686 µs；扫描后目录/资产 P95 39/67 µs；heap 37,607,368 B，DB 族
28,618,752 B，预算超限为 0。1,000 层 rollup 为 70 ms。该数字是本地回归记录，不替代
CI 双架构结果或代表性发布设备。

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

三档趋势和暂定预算：

```sh
make capacity-trend
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

## 后续 Gate 条件

Stage 0 扫描/索引可行性范围已通过。完整 FS-04 仍至少需要：

1. **扫描/媒体 Backend Gate**：增加生产媒体探测与缩略图队列后复测全局并发、RSS、
   队列深度、磁盘增长和失败行为；
2. **浏览/搜索 Backend Gate**：FTS5、自然排序、稳定 keyset catalog API 完成后测量列表、
   搜索、扫描并发和 cursor 稳定性；
3. **Browse/Search UI Gate**：通过真实 API 测虚拟化 DOM 上限、焦点/滚动稳定性与用户可感知
   延迟；
4. **Performance/Release Gate**：在记录清楚的四核、4 GiB 代表性设备、本地可靠文件系统和
   最终镜像运行目标档，记录生产 RSS、WAL/checkpoint、HTTP 并发和退化行为；
5. 根据代表性设备实测固定发布预算，未达标时降低并发、缩略图规格或声明支持上限。

在这些条件满足前，完整 FS-04 保持 Conditional，不以本轮数据承诺产品性能；但它们不再
反向阻断已完成的 Stage 0 可行性范围。
