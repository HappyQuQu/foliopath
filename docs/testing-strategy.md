# FolioPath 测试策略

## 状态

本文同时定义目标验证体系和当前已落地的最小 Go 验证面。仓库已有路径边界、媒体库、
scanner、SQLite 单元测试，贯通 files → scanner → SQLite 的临时目录集成测试，OpenAPI
契约测试，真实 `httptest.Server` 边界 harness、合成媒体 fixture 和显式容量测试。尚无可启动
应用、React 产品界面、生产 HTTP handler、浏览器 E2E 或发布镜像。仓库已有固定 Node/npm、
确定性 OpenAPI TypeScript 生成、strict typecheck、依赖 audit、唯一 client 边界和双架构 CI
工作流；首次原生 amd64/arm64 PR CI 已通过。只有实际执行成功的目标才能声称可用。

当前证据分别见 [FS-01 路径边界](spikes/fs-01-path-boundary.md)、
[FS-02 SQLite/generation](spikes/fs-02-sqlite-generation.md)、
[FS-03 媒体矩阵](spikes/fs-03-media-matrix.md) 和
[FS-04 容量基线](spikes/fs-04-capacity-baseline.md)。FS-01 路径和 FS-02 当前正确性 scope
已通过；FS-03/04 只在明确子范围取得证据并保持 Conditional，不能替代完整媒体、容量、浏览器
和发布验证。FS-01 的 Stage 0 范围包括原生 Linux amd64/arm64 `openat2` mount 拒绝和 HTTP
harness；生产 handler/auth 与发布 volume/unmount 分别由后续 Backend/Release Gate 强制。

## 质量目标

测试优先保护以下不可逆或高风险边界：

1. FolioPath 永不修改、移动、改名或删除原始媒体。
2. 任意 API 输入都不能让读取逃出 `/library` 或目标媒体库。
3. 离线、权限失败和中断扫描不会误清已有索引。
4. 大型目录的扫描、列表和缩略图工作受内存、并发和事务边界限制。
5. 游标分页、虚拟列表和媒体 Range 在数据变化与取消请求时仍保持可用。
6. 发布镜像可备份、恢复和升级，不依赖开发机环境。

## 测试层次

### Go 单元测试

与源码同包，覆盖快速、确定且无外部进程的行为：

- 路径清理、组件级包含关系、重叠媒体库判定、唯一名称、根路径不可变和错误码。
- 扫描 generation 状态机、协作取消、失败是否具备清理资格、24 小时调度、跳过统计、任务幂等与重试退避。
- 游标编码/校验、稳定排序、查询过滤和 MIME/格式判定。
- 缓存键、源指纹、变换版本、10 GiB 默认配额、LRU 清理水位和临时文件提交规则。
- 单管理员原子初始化、密码验证、会话过期/撤销、退出、CSRF 和登录限流。
- HTTP handler 的请求校验、状态映射和错误脱敏（使用 fake service）。

路径解析、游标解码和媒体头解析加入 Go fuzz tests。任何 fuzz 失败输入都保存为回归样例。

当前已实现的 Go 单元覆盖包括：

- 共享相对路径策略、重复编码、无效 UTF-8/NUL、symlink、根移除/替换、A → B → A 身份回归、特殊节点、遍历取消和错误脱敏；
- Linux `openat2` 的失败关闭映射、`RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS |
  RESOLVE_NO_XDEV` 策略与旧子目录 FD 移出边界后的拒绝；
- 媒体库根规范化、组件边界重叠判断、唯一名称、改名与根不可变；
- 固定 MVP 扩展名/MIME 正向矩阵、SVG/HEIC/HEIF/AVIF/RAW 负向矩阵与系统目录跳过清单；
- 真实文件 SQLite、Goose migration、WAL/外键/busy timeout、SQLite 安全版本门槛、integrity/foreign-key/checkpoint；
- generation 的失败、取消、离线、受控重启、原子 finalize 回滚、活动扫描竞争与 complete/cancel 竞态；
- 128 层目录链的逐级直接/递归计数，以及同库循环、跨库目录/资产损坏、当前代次条目指向
  同库陈旧目录等损坏在 stale cleanup 前失败关闭且不丢失当前行或影响另一媒体库。

游标、缓存、认证、生产 HTTP handler、调度和 fuzz 仍是目标项；测试专用 HTTP seam 不能
替代 handler 与中间件实现。

### 前端单元与组件测试

- URL 状态与路由往返：媒体库、目录、递归、搜索、过滤、排序。
- 目录树、面包屑、媒体卡片、扫描状态、离线提示和删除确认。
- 单管理员初始化/登录/过期流程，中英切换和浏览器语言默认值。
- 键盘导航、焦点恢复、可访问名称、动态状态公告和 reduced-motion 分支。
- Query 缓存更新、分页去重、错误重试边界和取消过期请求。
- 瀑布流尺寸估算与虚拟化适配逻辑，不以大量脆弱 DOM 快照代替行为断言。
- 查看器适应视口、缩放/平移、1:1、全屏与基本信息；验证没有完整 EXIF、显式下载按钮或移动滑动手势。

### 集成测试

放在 `tests/integration`，使用临时目录、真实 SQLite 和可控的媒体工具替身或合成 fixture：

- 创建媒体库、首次扫描、再次扫描、删除与离线恢复的完整后端路径。
- 唯一媒体库名称、只允许改名、拒绝根路径修改，以及 24 小时可配置调度与协作取消。
- generation 批量 upsert 与仅在完整成功后清理旧记录。
- 进程中断后的 `running` 任务恢复和原子缓存写入。
- FTS 搜索、Unicode 文件名、大小写、自然排序与游标稳定性。
- 真实文件变化：新增、修改、删除、重命名、权限错误、深目录和空目录。
- 所有可读目录的直接/递归计数，以及隐藏项、系统派生目录和回收站的跳过清单/统计。
- HTTP 条件请求、`HEAD`、合法/非法 Range、`416`、客户端取消。
- libvips/FFprobe/FFmpeg 的超时、损坏输入和并发限制；与 CLI 的调用使用参数数组而非 shell。
- 数据库迁移从每个已发布版本前进，并验证失败时不部分就绪。

测试只使用临时 `/library` 等价目录，绝不读取开发者或 CI 宿主机的真实照片。

当前 `tests/integration` 已使用 `t.TempDir()` 和真实文件 SQLite 覆盖首次递归扫描、空目录、
直接/递归计数、格式及系统目录跳过、失败/取消/离线/根替换与 A → B → A 替换保留、后续
成功收敛和跨媒体库隔离。测试专用 HTTP capability seam 还覆盖 opaque asset ID 到
`internal/files` 的 GET、HEAD、条件请求、单 Range、416、路径攻击和错误脱敏。FS-03 另以
运行时合成 fixture 调用真实 FFmpeg CLI；当前仍没有生产 handler/media adapter、真实
`SIGKILL`、磁盘满或备份恢复集成测试，“重启”证据仅为关闭并重新打开同一数据库文件。
带 `linux && fsboundary` tag 的隔离高权限探针已在原生 Linux amd64/arm64 覆盖同设备、
跨设备和 self-bind mount，普通非 root 双架构 Go/race 回归也已通过；这些证据不覆盖最终
只读发布 volume 或运行期 unmount。

### 浏览器端到端测试

放在 `tests/e2e`，优先保持少而关键：

1. 首次进入 → 原子创建单管理员 → 登录 → 创建媒体库 → 看见扫描状态 → 浏览第一批媒体。
2. 切换目录与递归模式 → 刷新/复制 URL → 状态仍可恢复。
3. 在当前目录、当前媒体库和全部媒体库之间搜索/过滤/排序 → 翻页或滚动 → 打开查看器 → 返回后恢复位置与焦点。
4. 媒体库离线 → 保留索引提示 → 挂载恢复并重新扫描。
5. 删除媒体库 → 明确非破坏性确认 → 配置消失且 fixture 原文件仍存在。
6. 桌面固定侧栏、移动目录抽屉、网格/瀑布流切换与键盘关键路径；检查中英界面、reduced motion 与高对比模式。
7. 会话过期、退出、CSRF 拒绝和再次访问受保护媒体，不泄露内容或敏感状态。

端到端测试不能通过固定长时间 sleep 等待扫描，应轮询可观察状态并设置明确超时。

### 容器与发布测试

- 最终镜像以非 root 用户运行，媒体目录 `:ro`，数据目录可写。
- `linux/amd64` 与 `linux/arm64` 运行相同的 smoke suite。
- PID 1 收到终止信号时停止接收请求、取消任务、提交安全状态并按时退出。
- readiness 与 liveness 在迁移、扫描、单库离线和数据库不可写时语义正确。
- Compose 启动、反向代理 Range、备份/恢复、向前迁移和失败回滚演练。
- 磁盘已满、缓存不可写、媒体挂载断连与容器重启不会修改 fixture 原件。

## 风险覆盖矩阵

| 风险 | 单元 | 集成 | E2E / 容器 | 发布阻断 |
| --- | --- | --- | --- | --- |
| 路径遍历或符号链接逃逸 | 是 | 是 | 容器抽样 | 是 |
| 删除/修改原媒体 | 服务行为 | 只读 fixture 校验 | 删除库流程 | 是 |
| 扫描失败误清索引 | 状态机 | 故障注入 | 离线恢复 | 是 |
| 游标重复、漏项或死循环 | 是 | 数据变化场景 | 长滚动抽样 | 是 |
| 损坏媒体耗尽资源 | 限制逻辑 | 真实工具超时 | 容器资源限制 | 是 |
| Range/视频取消错误 | handler | HTTP 流 | 浏览器播放 | 是 |
| 可访问性回退 | 组件 | — | 键盘/自动审计 | 严重项阻断 |
| 迁移/备份不可恢复 | 迁移测试 | 数据快照 | 恢复演练 | 是 |
| 大库性能失控 | 算法基准 | 合成规模测试 | 浏览器性能 | 超过已确认预算时阻断 |

更完整的产品与工程风险见[风险登记](risk-register.md)。

## Fixture 设计

`tests/fixtures` 只保存小型、合成或许可明确的数据，并带一份 manifest 描述预期：

- JPEG、PNG、WebP、GIF，以及 MP4、MOV、MKV 中的最小有效样例。
- 扩展名与魔数不一致、截断、畸形、超大像素声明、长动画元数据。
- 横竖方向、透明度、Unicode/emoji/组合字符、大小写近似和超长文件名。
- 深目录、空目录、隐藏目录、系统垃圾目录和权限受限子树。
- 浏览器兼容与不兼容的视频编码样例；容器可索引与编码可直放必须分别断言。
- SVG、HEIC/HEIF、AVIF 与 RAW 的非契约样例，验证不会被误报为 MVP 支持。

符号链接、权限、断连和磁盘已满应在测试运行时动态构造，避免提交平台相关链接。Windows 文件系统不属于容器运行目标，但跨平台开发工具应明确哪些测试只在 Linux 运行。

## 性能与容量验证

主要容量验收档已经确认为约 10 万媒体、1 万目录、4 GiB 内存的四核 NAS/家庭服务器。
FS-04 当前在 Linux/arm64 Docker Desktop VM 与 tmpfs 上完成混合宽度、最大深度 32 的
扫描/索引目标档：generation 扫描/finalize 为 10.449 秒，扫描期间库内页读取 P95 为
3.193 ms，采样 Go heap 峰值约 39.2 MB。测试还对账根 recursive count、全树 direct count
及选定 32 层链的每一级聚合。独立的 1,000 层 SQLite-only 档在同一受限 Linux 容器中
finalize 为 147 ms；它不创建宿主深目录，只证明目录 rollup 算法而不证明文件系统遍历。
该环境不是代表性 NAS 存储，且未包含媒体、FTS、正式 HTTP 或前端，因此这些数字不是发布
SLA，也不能代替完整容量门槛。

目录计数实现不得回到每目录扫描全部资产或展开 asset×ancestor 的复杂度。当前基线在
SQLite O(D) 临时工作表中以最多 500 个叶节点为一批向父目录传播，每条目录边只处理一次；
现有索引下按 O(A + D log D) 评估，Go 侧不加载完整目录树。循环、无效根、缺失/跨库关系，
或当前代次条目指向同库陈旧目录，必须使同一 finalize 事务回滚，不能发布部分计数或让
stale cleanup 级联删除另一媒体库或当前代次条目。

至少测量：

- 首次扫描与无变化复扫的文件/秒、峰值 RSS、SQLite 写入时间和磁盘增长。
- 缩略图吞吐、单任务峰值内存、缓存命中率和不同并发下的尾延迟。
- 目录树首屏、媒体列表 API p50/p95、搜索和游标翻页延迟。
- 浏览器首屏可交互时间、滚动掉帧、DOM 节点数、图片解码内存和查看器切换。
- 服务扫描期间 API 延迟与取消响应，证明后台任务不会饿死交互请求。

建立小/中/目标上限三档合成数据集，其中目标档固定为上述规模，并在[可行性研究](feasibility-study.md)的 spike 后把实测结果与延迟预算写回本文。

## 安全验证

- 路径输入表：绝对路径、`..`、双重编码、NUL、不同分隔符、Unicode 归一化和 symlink race。
- HTTP：认证绕过、CSRF、开放重定向、可信代理头、限流、错误信息与安全响应头。
- 媒体：像素炸弹、损坏容器、探测超时、命令参数注入和主动内容同源执行。
- 依赖：Go/npm 系统依赖漏洞扫描、镜像 SBOM、第三方许可证检查和固定构建来源。
- 日志：令牌、Cookie、宿主机路径、SQL 和原始 stderr 不得出现在正常或故障日志中。

认证范围已经确认；上述单管理员安全测试是稳定版发布阻断项，不能用“以后补”作为公网发布依据。

## CI 与合并门槛

目标验证入口由 `Makefile` 或等价任务统一，名称与根 `AGENTS.md` 对齐：

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

当前 `Makefile` 已存在 `fmt`、`fmt-check`、`arch-check`、`contract-check`、`lint`、`test`、
`test-race`、`test-integration` 和显式 `spike-capacity`；`arch-check` 验证 Go 依赖方向与
禁止的通用包。`contract-check` 固定使用 `kin-openapi v0.142.0` 和纯 Go ECMAScript
pattern 编译器，每次禁用 Go test 缓存，强制完整解析 YAML、解析本地引用、执行 OpenAPI
结构/pattern 验证，并用 AST/Schema 与跨源检查固定认证、错误、健康状态、分页、路径、
Range，以及 scanner/migration 的选择性关键不变量（当前为 `queued`、`animated` 与可空
`startedAt`）；这不是完整领域实现一致性证明，其余 ScanRun phase/issues/cancel 等语义仍须在
对应 handler 前的 Contract/Backend Gate 补齐。运行时不依赖 Ruby、Node 或网络。权威 OpenAPI 还通过了
Redocly 外部交叉验证；当前只有两条 health endpoint 未声明虚构 4xx 响应的规则 warning，
没有结构错误。`generate-check`、`web-check`、摘要锁和语义兼容入口已在本地通过，CI 已定义
PR 基线比较与原生 amd64/arm64 jobs，但尚无一次执行证据。`sqlc` 生成、前端组件/token 门禁
与 `test-e2e` 尚不存在。阶段 1 必须随相关源码补齐缺口，并保证本地与 CI 使用同一入口。

计划中的每次合并至少要求：

- 格式、架构依赖、静态检查、生成一致性和单元测试通过。
- 受影响的集成/组件测试通过；数据库、路径或扫描改动运行完整相关矩阵。
- OpenAPI、迁移、设计文档与实现没有可检测漂移。
- 新增依赖通过许可证与安全检查。
- 不允许以重试掩盖不稳定测试；先隔离并登记 owner 与修复条件。

发布候选额外要求：

- 全量集成、E2E、双架构容器和恢复演练通过。
- 没有未处置的高危安全问题或会修改原媒体的缺陷。
- 性能没有超过已确认预算，或退化已明确接受。
- 支持格式、部署参数、迁移和已知限制与 README/发布说明一致。

## 当前可执行检查

当前可执行：

- `go test ./...`
- `go test -race ./...`
- `make fmt` / `make fmt-check`
- `make arch-check`
- `make contract-check`
- `make generate-check`
- `make web-check`
- `make openapi-lint`
- `make compatibility-check OPENAPI_BASELINE=api/openapi.yaml`
- `make lint`
- `make test`
- `make test-race`
- `make test-integration`
- `make spike-capacity`（显式重型目标档）
- `git diff --check`
- 仓库内 Markdown 相对链接检查
- 可用时运行 Markdown linter
- 人工核对 README、PRD、ADR、API、部署与安全语义

`make test-e2e`、`sqlc` 生成检查、前端 import/token lint、Storybook/组件/视觉回归、
生产 HTTP/认证错误边界测试、只读发布 volume/运行期 unmount、浏览器测试、
完整媒体/搜索/前端容量、双架构发布镜像和恢复演练仍不可执行或尚不存在；定义好的 CI
执行现有 Go、双架构 openat2/mount、HTTP harness 或 tmpfs 容量检查不能替代这些缺失门槛。架构检查
的完整状态与最晚落地阶段见[架构适配度检查](architecture/fitness-functions.md)。
