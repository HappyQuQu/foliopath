
# FolioPath 后端开发清单

## 当前状态

- 当前阶段：Stage 3 浏览与缩略图后端；媒体库与扫描 Backend Ready
- 已完成：`S1-001`～`S1-008` 运行骨架；`S1-101`～`S1-106` 认证 Backend Ready；
  `S2-001`～`S2-007` 媒体库 Backend Ready；`S2-101～107` 可靠扫描 Backend Ready
- 当前任务：`S3-005` 有界媒体任务、缓存失效与磁盘保护
- 授权边界：[媒体库 Backend Ready](gates/MVP-2026-07-23/s2-library-backend-ready.md)和
  [扫描 Backend Ready](gates/MVP-2026-07-23/s2-scan-backend-ready.md)已完成后端交接；
  Stage 3 仍须独立通过 Contract Ready 与 Backend Ready
- 代码所有权：`cmd/`、`internal/`、`migrations/`、`api/openapi.yaml`、后端测试和部署适配

后端负责业务规则、API、数据库、文件安全、任务与媒体处理，不实现 React 页面。HTTP 结构以
`api/openapi.yaml` 为唯一来源；每个业务切片达到 `Backend Ready` 后才交给正式前端集成。

## Stage 1：运行基础与认证

### Go 运行骨架

- [x] `S1-001` 创建最小 `cmd/foliopath`，只负责参数解析、启动委托和退出码。
  - 完成证据：默认/`serve` 委托、`version`、help、usage/失败退出码与错误脱敏测试。
- [x] `S1-002` 创建 `internal/app` composition root，集中组装依赖和生命周期。
  - 完成证据：进程根取消上下文、唯一 `compose` 接线点、顺序启动、启动失败回滚、运行期
    故障传播、反向关闭和有界停机测试。
- [x] `S1-003` 实现经过验证的配置加载；固定 `/library`、`/app/data` 和单 HTTP 端口边界。
  - 完成证据：固定根目录、默认/环境/参数优先级、唯一参数、数值端口、未知配置和认证前
    非回环监听失败关闭测试；当前唯一可配置项为 `FOLIOPATH_LISTEN` / `--listen`。
- [x] `S1-004` 实现结构化日志、request ID、错误脱敏和优雅停机。
  - 完成证据：JSON 结构化日志与敏感属性脱敏、服务端 request ID、统一安全 JSON
    404/500、panic 前后响应提交行为、HTTP runtime 错误隐藏、监听失败和在途请求有界排空测试。
- [x] `S1-005` 实现 `/health/live`、`/health/ready` 与 `/api/v1/status`。
  - 完成证据：liveness 最小披露、readiness 安全原因/未知原因脱敏/Retry-After/停机转换、
    status 默认拒绝未认证及授权成功/内部失败测试；数据库未就绪时 readiness 明确返回 503。
- [x] `S1-006` 从空数据目录启动并执行嵌入 migration；验证重复启动和迁移失败行为。
  - 完成证据：固定 `/app/data` 下创建 `cache`/`tmp`/`foliopath.db`，数据库 → HTTP →
    readiness 顺序启动并反向关闭；空目录迁移、重复启动、冲突 schema 迁移失败分类和
    不可用数据目录失败关闭测试。
- [x] `S1-007` 建立 `sqlc` 配置、查询源、生成目录和确定性 `generate-check`。
  - 完成证据：固定 sqlc `v1.31.1`；`queries/` 为媒体库 SQL 唯一来源，`dbgen/` 为提交的
    生成包；library adapter 已消费生成查询；临时目录重生成 diff、生成标记和禁止 adapter
    复制同组 SQL 的架构检查已接入 `make generate-check` 与 CI。
- [x] `S1-008` 增加运行应用的集成测试、取消测试和最小容器 smoke。
  - 完成证据：真实 composition root 使用临时数据卷和随机回环端口连续启动两次；验证
    migration、live/ready、status 默认 401、根取消、listener 关闭和数据目录。测试专用
    非 root 容器以 `/library:ro`、`/app/data` 运行正式 `cmd/foliopath`，验证内部 health、
    SIGTERM 零退出、停机日志、重复启动和媒体 sentinel 不变；`make test-e2e` 与原生
    amd64/arm64 CI job 使用同一脚本。该镜像仅为测试证据，不是 Stage 5 发布镜像。

### 单管理员认证后端

- [x] `S1-101` 固定初始化、登录、会话、退出和 CSRF 的 OpenAPI/数据契约。
  - 完成证据：[认证 Contract Ready](gates/MVP-2026-07-23/s1-auth-contract-ready.md)；
    OpenAPI 固定匿名/受保护边界、同源 Origin、Cookie/CSRF、防缓存和逐状态错误码；
    `00002_authentication.sql` 固定单管理员、密码 verifier、会话/CSRF 摘要、期限、撤销和
    认证版本，并由并发/约束/兼容性测试验证。
- [x] `S1-102` 实现管理员首次初始化状态机和密码安全存储。
  - 完成证据：`internal/auth` 负责 Unicode NFKC/full case folding 身份规范化、输入边界、
    setup 状态机和 Argon2id verifier；SQLite adapter 通过 sqlc 查询和写事务原子关闭二次
    初始化。composition root 已接入真实 setup 状态，单元、并发、持久化与重启集成测试覆盖
    正确/错误密码、严格 verifier 解析、重复/并发初始化、错误脱敏和重启后状态。
- [x] `S1-103` 实现安全 Cookie 会话、过期、轮换与退出失效。
  - 完成证据：初始化与初始会话由同一 SQLite 写事务提交；登录使用统一
    `invalid_credentials` 领域错误和虚拟 Argon2id 校验。每次认证发放独立 32-byte
    Cookie/CSRF 秘密组成的复合 HttpOnly Cookie，数据库只保存 SHA-256 摘要；固定 7 天
    服务端绝对期限，不滑动续期。session 读取恢复 CSRF 并更新 last-seen，退出原子撤销；
    过期/撤销记录保留 24 小时后随新会话清理。单元、SQLite、composition root 重启和
    race 测试已覆盖，`internal/api` 固定 host-only、Path=/、HttpOnly、SameSite=Strict
    以及经验证 HTTPS 下 Secure 的 Cookie 策略。
- [x] `S1-104` 实现 CSRF、防缓存和全部业务 API 默认拒绝未认证访问。
  - 完成证据：五个认证端点已接入真实 `internal/auth` service；仅 health、精确方法匹配的
    auth status/setup/login 匿名，其余 `/api/v1` 请求先验证唯一会话 Cookie，未知业务路由
    同样默认拒绝。状态修改使用会话绑定 `X-CSRF-Token` 常量时间比较；setup/login 在读取
    JSON 凭据前验证完整同源 Origin。认证 JSON 和公共错误统一 `no-store`，请求体有大小、
    MIME、未知字段和单 JSON 值限制；登录/setup 具备按直连 peer 的有界并发安全限流，
    不信任转发头。真实 composition root HTTP 集成测试覆盖 setup、跨重启 session、
    受保护 status、重新登录、CSRF logout 与撤销后失败；架构检查约束唯一认证路由和
    middleware 所有者。
- [x] `S1-105` 覆盖错误脱敏、重复初始化、错误密码、过期会话和并发请求测试。
  - 完成证据：HTTP 表驱动测试验证 setup/status/login/session/logout 的内部错误统一脱敏，
    CSRF 拒绝不会触发退出；领域测试固定“过期前 1 ms 有效、等于期限即失效”。真实
    composition root、HTTP 与 SQLite 测试同时发起 8 个 setup，仅一个创建管理员，其余稳定
    返回 `setup_in_progress`/`setup_closed`；重复 setup 关闭，未知账号与错误密码返回相同
    `invalid_credentials`，32 个并发 session 请求全部成功。相关包测试与 race 测试执行。
- [x] `S1-106` 记录认证切片 `Backend Ready` Gate，允许前端连接真实认证 API。
  - 完成证据：[认证 Backend Evidence Ready](gates/MVP-2026-07-23/s1-auth-backend-ready.md)
    逐项审计 capability/adapter/composition 所有权、OpenAPI/数据契约、真实 SQLite/HTTP、
    故障/并发/重启/时间/脱敏证据和最终 CI；结论为 `Go`，只授权认证前端进入 Gate S3。
    回环监听、Stage 2 和 Stage 5 发布边界保持不变。

## Stage 2：媒体库与可靠扫描后端

Stage 2 已通过 Architecture Ready。执行顺序是先共同固定媒体库创建与 durable 首次扫描的
交接，再分别实现；媒体库和扫描各自仍需独立通过 Contract Ready 与 Backend Ready。

### 媒体库

- [x] `S2-001` 固定目录选择、创建、列表、详情、改名和移除的契约与失败语义。
  - 完成证据：[媒体库 Contract Ready](gates/MVP-2026-07-23/s2-library-contract-ready.md)；
    OpenAPI 固定七个端点的逐状态错误码、ETag/If-Match、摘要化幂等、创建与首次 queued
    scan 同事务、offline 保留和 restart-safe 异步移除；追加 migration 与 SQLite/契约测试
    验证 version 2→3、唯一 creation scan/active removal、commit/rollback 和 24 小时保留。
- [x] `S2-002` 实现 `/library` 安全目录枚举；所有 I/O 只经过 `internal/files`。
  - 完成证据：已认证 `GET /api/v1/library-paths` 通过 `internal/library` 的单一用例进入
    `internal/files` anchored adapter；直接子目录以 256 项批次流式读取、页面堆最多保留
    `limit+1` 项，使用 Unicode numeric natural key、AES-GCM opaque/query-bound keyset
    cursor 和默认 50/最大 200 限制。单元/race、真实 app composition、路径/文件/symlink、
    错误脱敏及带 `fsboundary` 的同设备/跨设备/self-bind mount 探针均有回归覆盖。
- [x] `S2-003` 实现唯一名称、相对根、不可变根和重叠根校验。
  - 完成证据：`internal/library` 统一负责名称 NFC 显示、NFKC + Unicode full case-fold
    唯一键、相对根规范化和组件级重叠规则；SQLite adapter 在 `BEGIN IMMEDIATE` 短写事务中
    原子执行名称/根冲突检查，并由唯一约束和不可变 root trigger 纵深防御。改名通过
    `UPDATE ... RETURNING` 返回自身提交表示，无变化改名不推进 revision。目录选择器使用
    权威库快照标注相同/祖先/后代冲突并在快照损坏或不可用时失败关闭；Unicode、边界值、
    两个 Store 并发创建/改名与 picker 冲突矩阵均有 race 回归。
- [x] `S2-004` 实现媒体库创建、改名、离线状态、重试和只删除派生数据的移除。
  - 完成证据：[S2-004 实现记录](gates/MVP-2026-07-23/s2-library-lifecycle-implemented.md)；
    创建在同一短事务提交媒体库、唯一 creation scan 与摘要幂等记录，真实根检查位于事务外，
    提交后只发送有界 wake hint。已认证 API 提供自然 keyset 列表、详情、强 ETag 改名、
    offline manual durable admission，以及 restart-safe removal；移除取消/等待活动扫描并分批
    清理 SQLite 与 `/app/data/cache`，没有 `/library` 写入/删除端口。
- [x] `S2-005` 覆盖 traversal、symlink、nested mount、TOCTOU、重叠、离线和权限失败。
  - 完成证据：[S2-005 文件系统安全矩阵](gates/MVP-2026-07-23/s2-library-safety-matrix.md)；
    词法、anchored I/O、真实认证创建/SQLite、Linux production mount 分类、特权
    same/cross-device/self-bind、ABA、离线保留、权限映射和公开错误脱敏形成连续证据。
- [x] `S2-006` 证明移除媒体库前后 fixture 原文件逐字节不变。
  - 完成证据：[S2-006 原媒体不变证明](gates/MVP-2026-07-23/s2-library-removal-invariance.md)；
    真实认证 HTTP/composition/SQLite/worker 删除链路逐项比较完整媒体树和文件字节，同时
    验证 `/app/data` 派生缓存清理、bounded cleanup 重启续作及 removal 无媒体写能力。
- [x] `S2-007` 记录媒体库后端 `Backend Ready` Gate。
  - 完成证据：[S2-007 媒体库 Backend Ready](gates/MVP-2026-07-23/s2-library-backend-ready.md)；
    最终审计确认 7 个媒体库 operation、capability/adapter/composition 所有权、真实
    SQLite/HTTP、路径故障矩阵、幂等/并发、重启移除与逐字节原媒体不变证据完整。允许前端
    建立真实媒体库 adapter；依赖扫描执行的产品流程继续等待 `S2-107`。

### 扫描

- [x] `S2-101` 固定扫描创建、状态、取消、issues 和默认 24 小时计划的契约。
  - 完成证据：[扫描 Contract Ready](gates/MVP-2026-07-23/s2-scan-contract-ready.md)；
    OpenAPI 固定 durable manual admission/coalesce、历史/条件详情、协作取消逐状态语义；
    version 4 migration 固定 phase/counters/cancel、heartbeat/lease/attempt、50 条 issues 上限
    和默认 24 小时 typed setting。契约同时冻结 256 active、2 workers、256 batch、
    15 秒 heartbeat/120 秒 lease、三次恢复和公平领取顺序。
- [x] `S2-102` 完成有界 generation 扫描服务和全局任务队列。
  - 完成证据：[S2-102 有界扫描 worker](gates/MVP-2026-07-23/s2-bounded-scan-worker.md)；
    `internal/jobs` 统一拥有 lease queue、2-worker 全局并发、15 秒 heartbeat/120 秒 lease
    和取消传播，SQLite 按 available/created/ID 原子领取并执行最多三次恢复；正式
    composition 已消费 creation scan，单元、SQLite、架构、真实应用与完整仓库门禁通过。
- [x] `S2-103` 索引全部可读目录，包括空目录，并维护直接/递归计数。
  - 完成证据：[S2-103 目录索引与计数](gates/MVP-2026-07-23/s2-directory-counts.md)；
    scanner 流式发射根和全部可读/空/隐藏目录，SQLite 在成功 finalize 中以 O(D) 有界叶批次
    发布全部直接/递归计数。production creation scan、128 层链、失败保留、损坏拓扑回滚和
    分类型跳过统计均有自动证据。
- [x] `S2-104` 实现媒体候选识别、fingerprint、增量 upsert 和成功后陈旧清理。
  - 完成证据：[S2-104 媒体候选与增量收敛](gates/MVP-2026-07-23/s2-media-convergence.md)；
    唯一媒体注册表固定 MVP 候选，version 6 migration 回填大小/纳秒 mtime fingerprint；
    真实 walker、SQLite 与 production creation worker 已验证同路径 ID 保持、变化失效、
    重命名新 ID、processed counter 以及仅成功 generation 的 stale cleanup。
- [x] `S2-105` 实现取消、离线、权限失败、重启恢复和失败保留旧索引。
  - 完成证据：[S2-105 扫描故障与重启恢复](gates/MVP-2026-07-23/s2-scan-recovery.md)；
    启动时按媒体库 ID 分页 admission/coalesce startup scan，worker 持续恢复后来到期的
    lease；取消、离线、部分不可读、nested mount、根替换和任意内部失败只写稳定错误码，
    不执行 stale cleanup。真实应用重启验证 startup scan 与晚到期 lease 都能收敛。
- [x] `S2-106` 覆盖并发扫描、队列上限、深目录、损坏拓扑和容量回归。
  - 完成证据：[S2-106 扫描容量与并发回归](gates/MVP-2026-07-23/s2-scan-capacity.md)；
    唯一 256 active/256 batch 边界、跨连接 admission 竞态、2-worker 跨库公平、深目录/
    损坏拓扑、SQLite 满页保留可靠 generation 与 10k/100k 强制预算均有自动证据；
    Linux amd64/arm64 容量 job 已接入 CI。
- [x] `S2-107` 记录扫描后端 `Backend Ready` Gate。
  - 完成证据：[可靠扫描 Backend Ready](gates/MVP-2026-07-23/s2-scan-backend-ready.md)；
    扫描历史、详情/304、手动 admission、协作取消、设置与 24 小时 scheduler 均已接入
    真实 HTTP/SQLite/composition，S2-101～106 的一致性、故障与双架构容量证据汇总通过。

## Stage 3：浏览与缩略图后端

- [x] `S3-001` 固定目录树、面包屑、当前目录和递归媒体列表的游标契约。
  - 完成证据：[目录与媒体浏览 Contract Ready](gates/MVP-2026-07-23/s3-browse-contract-ready.md)；
    已固定可序列化 root、root-to-current breadcrumb、direct/recursive scope、自然名称/
    修改时间 tuple、generation-bound cursor、跨库 404、offline preserved index 与请求不访问
    文件系统。允许进入 S3-002/003，不授权缩略图、搜索、前端集成或发布。
- [x] `S3-002` 实现稳定排序、opaque cursor、查询指纹和请求取消。
  - 完成证据：[Catalog 排序与游标](gates/MVP-2026-07-23/s3-catalog-keyset.md)；
    migration 7、唯一 catalog query/root owner、目录/资产完整 tuple keyset、
    generation-bound 指纹、跨库隔离、offline 保留与 context cancellation 已由真实
    SQLite 测试固定；搜索明确 fail closed，未被静默忽略。当前进入 S3-003。
- [x] `S3-003` 实现包含空目录及直接/递归计数的目录树。
  - 完成证据：[S3-003 目录树与详情](gates/MVP-2026-07-23/s3-directory-tree.md)；
    direct-child page、root detail、完整 breadcrumb、空目录/可靠计数、1000 层链与损坏
    parent fail-closed 已由领域、SQLite、HTTP 和真实认证 composition 测试固定。浏览请求
    在源目录移走后仍只读保留索引；当前进入 S3-004。
- [x] `S3-004` 实现 govips/FFmpeg 媒体探测、缩略图/视频封面和损坏状态。
  - 完成证据：[S3-004 媒体处理](gates/MVP-2026-07-23/s3-media-processing.md)；
    govips/FFmpeg production adapter、统一结果/错误、fingerprint-bound 派生键、原子
    cache publisher 与 migration 8 ready/failed 状态均有单元、SQLite、真实组合和原生
    依赖 fixture。尚未接入后台任务或 HTTP；当前进入 S3-005。
- [ ] `S3-005` 实现有界媒体任务队列、fingerprint 失效、默认 10 GiB LRU 和磁盘余量保护。
- [ ] `S3-006` 覆盖损坏媒体、像素炸弹、超时、取消、磁盘满和并发限制。
- [ ] `S3-007` 记录浏览/缩略图后端 `Backend Ready` Gate。

## Stage 4：搜索与媒体内容后端

- [ ] `S4-001` 固定文件名、类型、日期、路径和三种搜索范围的契约。
- [ ] `S4-002` 实现 SQLite FTS/keyset 搜索、稳定排序、取消和离线语义。
- [ ] `S4-003` 完成搜索后端正确性、容量和 `Backend Ready` Gate。
- [ ] `S4-005` 固定并实现资产详情、原内容、HEAD、单 Range、条件请求和 416 契约。
- [ ] `S4-005B` 覆盖认证、路径边界、Range、取消、损坏/离线资产并记录媒体内容
  `Backend Ready` Gate。

## 后端完成任务时

1. 更新 OpenAPI、migration、风险和相关架构文档。
2. 运行适用的架构、契约、单元、race、集成和容器测试。
3. 在对应 `Backend Ready` Gate 记录成功/失败/离线语义与前端可依赖的接口版本。
4. 不在后端任务中顺带创建页面、组件、前端 mock 行为或第二套 wire 类型。
