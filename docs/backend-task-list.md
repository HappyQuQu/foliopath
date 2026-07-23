# FolioPath 后端开发清单

## 当前状态

- 当前阶段：Stage 1
- 已完成：`S1-001`～`S1-008` 运行骨架；`S1-101`～`S1-103` 认证契约、初始化与安全会话
- 当前任务：`S1-104` CSRF、防缓存和业务 API 默认拒绝未认证访问
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
- [ ] `S1-104` 实现 CSRF、防缓存和全部业务 API 默认拒绝未认证访问。
- [ ] `S1-105` 覆盖错误脱敏、重复初始化、错误密码、过期会话和并发请求测试。
- [ ] `S1-106` 记录认证切片 `Backend Ready` Gate，允许前端连接真实认证 API。

## Stage 2：媒体库与可靠扫描后端

### 媒体库

- [ ] `S2-001` 固定目录选择、创建、列表、详情、改名和移除的契约与失败语义。
- [ ] `S2-002` 实现 `/library` 安全目录枚举；所有 I/O 只经过 `internal/files`。
- [ ] `S2-003` 实现唯一名称、相对根、不可变根和重叠根校验。
- [ ] `S2-004` 实现媒体库创建、改名、离线状态、重试和只删除派生数据的移除。
- [ ] `S2-005` 覆盖 traversal、symlink、nested mount、TOCTOU、重叠、离线和权限失败。
- [ ] `S2-006` 证明移除媒体库前后 fixture 原文件逐字节不变。
- [ ] `S2-007` 记录媒体库后端 `Backend Ready` Gate。

### 扫描

- [ ] `S2-101` 固定扫描创建、状态、取消、issues 和默认 24 小时计划的契约。
- [ ] `S2-102` 完成有界 generation 扫描服务和全局任务队列。
- [ ] `S2-103` 索引全部可读目录，包括空目录，并维护直接/递归计数。
- [ ] `S2-104` 实现媒体候选识别、fingerprint、增量 upsert 和成功后陈旧清理。
- [ ] `S2-105` 实现取消、离线、权限失败、重启恢复和失败保留旧索引。
- [ ] `S2-106` 覆盖并发扫描、队列上限、深目录、损坏拓扑和容量回归。
- [ ] `S2-107` 记录扫描后端 `Backend Ready` Gate。

## Stage 3：浏览与缩略图后端

- [ ] `S3-001` 固定目录树、面包屑、当前目录和递归媒体列表的游标契约。
- [ ] `S3-002` 实现稳定排序、opaque cursor、查询指纹和请求取消。
- [ ] `S3-003` 实现包含空目录及直接/递归计数的目录树。
- [ ] `S3-004` 实现 govips/FFmpeg 媒体探测、缩略图/视频封面和损坏状态。
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
