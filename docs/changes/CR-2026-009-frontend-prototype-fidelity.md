# CR-2026-009：生产前端原型一致性

## 状态

- 状态：Confirmed
- 变更等级：C2
- 目标版本：`MVP-2026-07-23`
- Scope revision / 范围状态：Frozen revision 4；替代 revision 3
- Change Record ID / 基线事件：CR-2026-009 / 2026-07-30 产品确认
- 提出日期：2026-07-30
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/auth`、`internal/catalog`、`internal/thumbnail`、`web`

## 用户问题与价值

- 用户 / JTBD：管理员需要以已确认原型为可靠开发合同，在一致的全局壳和独立功能页中完成
  找图、浏览、查看和管理，而不是在生产界面中面对多套导航和不完整按钮。
- 当前问题及证据：生产共享壳、管理中心、浏览工具栏和响应式与
  `prototypes/apple-redesign` 存在明显漂移；账户维护、全量目录过滤和缓存清理仍缺合同。
- 为什么必须进入目标版本：产品负责人把生产 UI 与原型一致性确认为当前首要任务，并明确将其
  作为独立 feature；稳定发布前必须避免原型和生产成为两套产品。

## 范围

- 新增或改变的 FR/NFR：`FR-BRW-010`、`FR-UI-010`、`NFR-UIF-001`；复用
  `FR-AUTH-005`、`FR-UI-009`。
- 明确包含：全局 Header、共享壳、管理四页、当前目录全量过滤、账户维护、最小缓存摘要/
  清理、逐页视觉 Gate。
- 明确不包含：任务中心、缓存 missing/all rebuild、系统维护、备份、AI/OCR/人脸、多用户、
  上传、转码或新部署单元。
- 被替代/延期的现有范围：替代把原型一致性当作零散 FIX 的交付方式；既有 FIX 继续保留为
  历史输入。任务中心和系统维护继续延期。
- Scope-budget exception：产品负责人接受该 feature 在 MVP 中增加 account、directory query、
  cache cleanup、共享壳重构和完整视觉回归的 API/后端/前端/测试预算；不以削弱安全、容量、
  原媒体只读或 Stage 5 阻断项换取。

## 架构影响

- Capability 与依赖方向：auth/catalog/thumbnail 继续拥有业务规则；API/SQLite/Web 只作
  adapter/consumer。共享前端系统拥有 token、原语和壳。
- API / 用户流程：新增账户维护、direct-directory q、cache summary/cleanup；浏览和管理流程
  更新。精确 wire 由 UIF-S1 固定。
- 数据 / migration / 派生状态：优先复用现有 users/session/cache；若查询索引或 cleanup run
  需要持久化，只追加 migration。
- 安全、隐私与信任边界：密码修改验证当前密码并撤销其他会话；目录查询不访问文件系统；
  缓存清理不触及原媒体；无新信任边界。
- 性能、容量与并发：10k 目录 query、100k 媒体 cache cleanup 和浏览并发必须有界。
- 部署、升级、备份、恢复与观测：不增加部署单元；新 migration/operation 进入既有升级恢复
  和结构化日志，不记录秘密。
- 平台、依赖、许可证与供应链：不计划引入新运行依赖；三浏览器与 linux/amd64/arm64 发布
  证据需重跑适用范围。
- ADR：N/A；不改变核心技术、部署、信任、持久化类型、事务 owner 或依赖方向。

## 质量属性场景

- 刺激：管理员在不同视口进入任意生产页面、过滤大目录、修改密码或清理缓存。
- 环境：可能存在 100k 媒体/10k 目录、离线库、并发扫描、session 竞争或磁盘压力。
- 系统响应：视觉与原型保持同层级；查询和写操作有界、可恢复；错误不伪装为空或成功；
  原媒体与可靠索引保持安全。
- 可测结果：`UIF-AC-001～012`、四档同视口无 P0/P1/P2、完整后端/前端/纵向证据。

## 风险与验证

- 风险 ID / 新风险：新增 R-021；复用 R-010、R-012、R-015、R-016。
- Fallback / 回滚：按共享壳和页面分批；未通过 Gate 的页面保留现有生产实现；新 API 可不被
  消费但不能由 mock 替代。
- 正常、边界、失败、恢复测试：见 feature failure matrix 和专用任务清单。
- Fixture 与目标环境：合成 100k 媒体/10k 目录、四核/4 GiB、四档视口、三浏览器、
  中英文、浅色/深色。
- 验收证据要求：OpenAPI、后端集成、安全/容量、生成 client、Storybook、视觉 comparison、
  三浏览器 E2E、原媒体 sentinel 和 Stage 5 Gate。

## Gate 影响与决定

- 需要重跑的阶段/切片 Gate：auth Contract/Backend、browse Contract/Backend、cache Backend、
  Consumer/UI、Integrated Slice，以及 Stage 5 browser/capacity/security/RC 适用 Gate。
- 产品决定：Go，作为当前首要 feature。
- 架构决定：Go 进入 UIF-S1；后端和业务 UI 继续由 S1/S2 失败关闭。
- 安全/数据评审：账户和缓存写 operation 在 S1 必须重新评审；无原媒体写入授权。
- 最终结论：Go for Architecture Ready / Conditional Go for production implementation。
