# S5-005 发布候选容量 Gate

## 结论

**Go — `S5-005` 发布候选容量验收完成。**

生产候选现在可在 4 CPU、4 GiB、10,000 个目录和 100,000 个独立有效媒体文件下运行
真实首次设置、建库、完整扫描、后台缩略图、认证 HTTP 浏览与搜索，并验证原媒体保持只读。
2026-07-28 本机 Docker Desktop linux/arm64 目标档通过候选预算，指定原生 amd64
服务器又分别通过 31,899 项真实 ZFS 媒体诊断和 100k/10k 合成目标档。按操作者决定，
本轮原生 amd64 服务器与本机 arm64 结果构成发布容量架构证据，不等待计费阻断的
amd64 CI。三引擎浏览器 FPS/RSS、全量媒体吞吐、cache 水位与持续运行预算均已验证。
真实浏览器和物理设备签署归 `S5-006B`，不重复阻断容量 Gate。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-005A`、`S5-005B`
- 需求/质量：`NFR-PERF-001～002`、`NFR-OPS-001`
- owner：后端负责人拥有扫描、SQLite 与媒体后台任务；前端负责人拥有浏览器 FPS/RSS；
  QA/发布负责人拥有目标设备矩阵与证据归档
- 合同：`tests/fixtures/capacitygen`、`tests/release/capacity_smoke.sh`、
  `tests/performance/capacity_test.go`、`web/scripts/measure-capacity.mjs`、根
  `Dockerfile` 与 `.github/workflows/ci.yml`
- 风险：R-005、R-009、R-013、R-016
- 架构影响：没有改变部署单元、持久化/信任边界或模块方向；本轮只落实既有 SQLite
  单写者规则并补齐候选性能证据，不新增 ADR

## 候选档与预算

合成器创建精确目录数和资产数。每个资产是独立、有效的 1×1 PNG 文件，不使用
硬链接；容量脚本在退出前复核哨兵文件 SHA-256。候选容器固定 4 CPU/4 GiB、只读根、
只读 `/library`、`cap-drop ALL` 和 `no-new-privileges`，通过真实管理员会话调用 API。

| 指标 | 本机候选预算 | 原生 Linux 预算 |
| --- | ---: | ---: |
| 完整扫描 | 240 s | 180 s |
| 递归浏览、库内搜索、全局搜索 P95 | 各 250 ms | 各 250 ms |
| 容器 memory peak | 1.5 GiB | 1.5 GiB |
| SQLite 家族 / thumbnail cache | 各 1 GiB | 各 1 GiB |

目标浏览器的 100,000 项虚拟长列表预算固定为 FPS 不低于 45、帧间隔 P95 不高于
34 ms、浏览器进程树峰值 RSS 不高于 1.5 GiB、同时挂载媒体项不超过 64。RSS 预算与
4 GiB 主验收机及 1.5 GiB 服务端峰值护栏配对，避免单个客户端或服务端耗尽整机内存。

本机扫描预算比原生 Linux 宽，是因为媒体树通过 macOS Docker Desktop bind mount 提供；它不是
NAS HDD/SSD 延迟模型。原生 Linux 使用 180 秒回归护栏。二者都不是用户可见 SLA。

## 2026-07-28 本机证据

环境为 Docker Desktop linux/arm64、4 CPU、4 GiB，候选镜像启用真实 libvips/FFmpeg：

| 指标 | 结果 |
| --- | ---: |
| 目录 / 媒体 | 10,000 / 100,000 |
| fixture / 完整扫描 | 5,849 ms / 189,000 ms |
| 递归浏览 P95 | 57.311 ms |
| 库内 / 全局搜索 P95 | 45.602 / 72.586 ms |
| 容器 memory peak | 350,633,984 B |
| SQLite DB/WAL/SHM | 146,919,424 B |
| 已生成 cache | 6,311,936 B |
| 候选镜像 | 221,991,705 B |

同一修复候选的 2,500 目录/25,000 媒体诊断档得到浏览 17.139 ms、库内搜索
54.944 ms、全局搜索 33.244 ms，峰值内存 196,079,616 B。

## 2026-07-28 原生 amd64 真实媒体诊断

[S5-002A/S5-005C](s5-native-amd64-real-media.md) 在 4 CPU/4 GiB 原生 amd64
服务器，以只读 bind 扫描本地 ZFS 上 1,709 个目录和 31,899 个支持资产：完整扫描
37.369 s，30 次递归浏览 P95 24.680 ms，抽样内存 142.7 MiB，原文件哈希不变。
本轮还发现并修复最小 FFprobe 不支持 `-nostdin` 的真实视频派生缺陷；重测有 228 个
视频封面 ready、0 failed，认证缩略图返回 HTTP 200 WebP。

该结果证明真实混合媒体和原生 amd64 运行路径，但规模、存储形态和单轮抽样不足以冻结
发布容量预算。

## 2026-07-28 原生 amd64 目标容量

[S5-005D](s5-native-amd64-capacity.md) 在同一 4 CPU/4 GiB 原生 amd64 服务器完成
精确 100,000 媒体/10,000 子目录目标档。首次扫描 148.293 s；修复后的两次纯无变化
重扫为 100.303 s 和 105.548 s；恢复 94,234 个媒体后台任务调度后重扫为 140.190 s。
浏览、库内搜索、全局搜索 P95 分别为 83.512、91.208、126.205 ms；优化后的扫描并发
浏览 P95 为 4.560 ms。取消和 offline 均保留 100,000 资产/10,001 目录可靠索引，媒体
只读哨兵未变化。

## 2026-07-28 目标浏览器长列表

`web/scripts/measure-capacity.mjs` 在物理 arm64 Mac 上分别启动 Playwright 自带的
Chromium、Firefox 与 WebKit 进程，以 1280×720 视口对
`Patterns/MediaCollection/Capacity100k` 连续滚动 5 秒，同时采集 animation-frame
间隔、挂载 DOM 和整个浏览器进程树 RSS：

| 引擎 | FPS | 帧间隔 P95 | 长帧（>50 ms） | 挂载项 | 峰值 RSS |
| --- | ---: | ---: | ---: | ---: | ---: |
| Chromium | 60.002 | 16.8 ms | 0 | 60 | 723,714,048 B |
| Firefox | 57.400 | 18.58 ms | 0 | 60 | 1,349,386,240 B |
| WebKit | 59.964 | 22.0 ms | 0 | 60 | 116,162,560 B |

三引擎均通过上述预算；结果写入 `build/stage5-browser-capacity.json`。这关闭自动化
FPS/RSS 与有界 DOM 子范围，但不把 Playwright WebKit 冒充 Safari 真机，也不替代
`S5-006B` 的最终物理浏览器、读屏和移动设备签署。

## 2026-07-28 全量媒体与缓存水位

同一原生 amd64 4 CPU/4 GiB 目标档继续处理全部 100,000 个有效 PNG。初始已有 965 个
任务成功，剩余 99,035 个任务在 6,202,317 ms（103 分 22 秒）内完成，平均
15.967 项/秒；最终任务为 100,000 succeeded、0 queued/running/failed，资产为
100,000、目录为 10,001。派生完成时数据库记录 100,000 个 ready thumbnail、
4,400,000 B，cache 目录为 4,500,776 B；SQLite DB/WAL/SHM 分别为
149,979,136 / 11,350,632 / 32,768 B。容器全程峰值内存 176,287,744 B，
未发生 OOM，媒体只读哨兵 SHA-256 未变化。

把 cache 配额在线降至 4,194,304 B 后，首轮测试发现淘汰写绕过 SQLite `writeGate`，
会与其他写入竞争并使应用退出；改为统一写门后，又发现逐文件事务在代表性 ZFS 上仅约
4 项/秒。最终实现先在事务外删除最多 64 个可重建文件，再以一个有界事务批量删除对应
状态；若文件删除中途失败，已删除文件对应状态仍先收敛，未删除项保持不变。回归测试固定
缓存维护写必须等待现有事务，并验证 LRU 顺序与文件失败语义。

修复候选以保留的 100k 数据重启后，从 3,783,824 B 收敛到 80% 低水位
3,355,440 B（上限 3,355,443 B），约处理 52 项/秒；随后持续观察超过 60 秒，数值保持
不变。最终有 76,260 个 ready thumbnail、0 pending deletion、cache 目录
3,432,476 B、100,000 个 succeeded job；容器 healthy、OOM=0，重启后 memory peak
45,142,016 B，媒体哨兵仍未变化。

由代表性设备结果冻结回归护栏：100k 全量派生不超过 2.5 小时、terminal failure 为零、
服务端峰值内存不超过 1.5 GiB，并须把超配额缓存收敛到 80% 低水位。这些是回归护栏，
不是用户可见 SLA。

## 本轮发现并关闭的并发、重扫与缓存缺陷

初始候选在媒体任务连续写入时，认证中间件的每请求 session touch 绕过 SQLite
`writeGate`，会与媒体事务竞争，最终在 5 秒 `busy_timeout` 后把正常浏览返回为 500。
`TouchSession` 现在通过同一 `withWriteTx` 串行门和事务执行；认证仍逐请求读取并验证，
审计 `last_seen` 写入限制为最多每 30 秒一次。跨过该边界的并发浏览 P95 为 6.614 ms。

原生目标档随后暴露无变化重扫仍会重复失效派生状态并 upsert 已存在媒体任务。scanner
现在一次读取已有 fingerprint/generation/匹配任务，并仅在源变化或任务缺失时写派生状态；
原生纯重扫从 241.949 s 降至 100.303 s。回归测试固定 unchanged fingerprint 不得删除
ready thumbnail 或重置成功任务。

govips 运行时还把默认逐操作 Info 日志收敛为 Warning 及以上，避免大库后台派生形成无界
常规日志流；native concurrency、64 MiB cache、32 entry 和 0 cached files 的资源边界不变。

缓存配额测试进一步关闭了 SQLite 缓存维护写未进入单写门、逐项事务写放大的缺陷；最终
实现与验证数据见上节。

## 自动化与结论

- `make test-release-capacity` 运行小档；`make release-capacity` 运行并强制目标档预算；
  `make test-browser-capacity` 构建 100k Storybook 主档并强制三引擎 FPS/RSS/DOM 预算。
- CI 保留原生 amd64/arm64 `release-capacity` job 和 JSON artifact 契约；本轮按操作者
  决定，以指定原生 amd64 服务器和本机原生 arm64 实测替代计费阻断的 CI 运行。
- `tests/performance/capacity_test.go` 增加正式 catalog service 的递归浏览 P50/P95，
  防止只测底层 SQL 而遗漏业务校验。

`S5-005A～D` 的扫描、查询、全量媒体、cache 水位、FPS/RSS、只读性和资源预算均已有
证据，因此 `S5-005` 关闭。`S5-009` 和 Release Candidate 仍由 `S5-001`、`S5-006`
与 `S5-007` 的独立关闭条件阻断。
