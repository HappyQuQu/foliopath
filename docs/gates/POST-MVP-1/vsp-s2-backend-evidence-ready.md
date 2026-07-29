# VSP-S2 视频故事板 Backend Evidence Ready

## 结论

**Go — VSP-S2 Backend Evidence Ready。**

`FTR-VID-001` 的后端领域规则、SQLite migration/repository、durable job、FFmpeg adapter、
认证 API、生产镜像运行时与目标容量证据已收敛。`VSP-201～208` 前端切片现在获准连接
真实生成 client 与 storyboard 状态；本 Gate 不代表前端或整个 feature 已完成。

## 已接受实现

- `internal/thumbnail` 唯一拥有 eligibility、4/10 帧均匀采样、布局、派生身份、
  all-or-nothing 发布和失败语义。
- `internal/media/videoffmpeg` 通过继承 FD、argv、input-side fast seek、单线程、
  45 秒 deadline、进程组取消和输出上限生成 PNG 临时帧与 WebP sprite。
- migration 11 只追加重建 `thumbnails`、`media_jobs` 和 per-priority library state；
  grid/storyboard 独立 variant，grid priority 0，storyboard priority 100。
- SQLite admission 每次最多 128 项；任何 eligible grid 严格先于 storyboard；
  storyboard 有效并发为 1，同 priority 继续跨库公平。
- 现有 worker、cache quota/LRU、atomic publisher、source fingerprint CAS 和
  `internal/files` 边界被复用，没有第二状态机或部署单元。
- `api/openapi.yaml`、生成 client 和 API adapter 已支持 storyboard DTO 与 binary delivery；
  session、ETag、304、immutable cache、`nosniff`、pending/offline/failed/not-found 映射
  复用现有权威 owner。

## 生产运行时证据

2026-07-29 执行 `make test-storyboard-runtime`：

- 从固定 FFmpeg 7.1.5 源码分别构建 `linux/arm64` 和 `linux/amd64`；
- 实际二进制确认 PNG 编解码、libwebp、`image2`、`scale`、`setsar`、`xstack`；
- 同一 10 秒 fixture 在两个架构均完成 10 次 fast seek 和 5×2 sprite；
- 两端输出均为 800×180 WebP、18,726 bytes，输入 hash 未变化。

该验证发现并修复了原生产 allowlist 缺少 PNG/zlib、`image2`、`setsar` 和 `xstack` 的问题。
`foliopath-ffmpeg` 7.1.5-2 现在显式启用 zlib，最终 distroless rootfs 也包含并登记
`zlib1g`。

2026-07-29 执行 `make test-storyboard-vertical`：

- 在四核、4 GiB、只读根和只读 `/library` 的真实生产镜像中完成
  setup/login → 建库 → 扫描 → grid → storyboard ready；
- 已认证 binary 为 200，条件请求为 304，未认证为 401；
- 删除 storyboard cache 后首先返回 202，后台重新生成后恢复 200；
- 原视频 hash 与 mtime 全程不变。

## 正确性、恢复与安全证据

- 真实 SQLite + `internal/files` + cache publisher + durable worker 集成证明 grid 先完成，
  storyboard 后 admission，并独立提交 10 帧/5×2 layout。
- 认证 HTTP composition 覆盖 200/202/304/401/404/409/422、DTO layout、ETag、
  immutable、`nosniff` 和错误脱敏。
- 单元/集成矩阵覆盖 migration fresh/v10 upgrade/downgrade fail-closed、约束、running
  lease、并发 claim、过期恢复、重试、取消、源变化、cache missing、LRU、库移除、
  invalid sprite、ENOSPC 和临时文件清理。
- 损坏视频与 hostile metadata 不产生可见派生；工具输出、路径、SQL 和临时文件名不进入
  API 错误。

## 目标容量证据

同一 100,000 资产 / 10,000 目录、10% 视频 fixture 以 `GOMAXPROCS=4` 执行：

| 环境 | storyboard 数 | admission | 入队耗时 | 入队期间浏览 P95 | Peak RSS | 预算违规 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Darwin arm64 | 10,000 | 80 次，最大 128 | 827 ms | 596 µs | 平台未提供 | 0 |
| Linux arm64，4 CPU / 4 GiB | 10,000 | 80 次，最大 128 | 983 ms | 786 µs | 45,010,944 B | 0 |

Linux 档数据库与 WAL 合计 132,620,288 bytes，完整扫描 52,199 ms。容量测试还在
10,000 个 queued storyboard 存在时确认下一次 claim 仍为 grid。该结果只证明 admission、
查询与调度边界，不把 10,000 个 FFmpeg job 的完成时间描述为吞吐承诺。

## 实际执行的回归

2026-07-29 全部成功：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-race
make test-e2e
make test-storyboard-runtime
make test-storyboard-vertical
make spike-capacity
```

另外在受限 Linux arm64 容器中执行相同完整容量测试，结果见上表。OpenAPI contract、
SQL/client deterministic generation、架构 fitness 与 `git diff --check` 均通过。

## 残余风险与下一授权

- `R-018` 的后端部分已缓解；hover timer、图片 decode、虚拟卡片回收和浏览器内存仍由
  `VSP-S3 Consumer/UI Ready` 阻断。
- 本 Gate 不授权宣称 feature 可用，不替代真实 Chromium/Firefox/WebKit、axe、
  reduced-motion、touch/keyboard 和 100 个可见视频前端容量证据。
- 前端必须从 `VSP-201` 开始，只通过生成 client 和唯一 availability adapter 消费；
  不得手写 wire type、直接 fetch 或复制 `MediaCollection`。
