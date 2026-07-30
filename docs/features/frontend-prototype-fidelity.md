# FTR-UIF-001：生产前端原型一致性

## 状态与版本

- 状态：`UIF-S4 Integrated Slice Done` Go；MVP Release Candidate 仍 No-Go
- 目标版本：`MVP-2026-07-23` / scope revision 4
- 当前阶段：`UIF-S4` 已签署；`UIF-001～408` 与受影响 Stage 5 复验完成，独立发布
  阻断继续由 Stage 5 Gate 持有
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/auth`、`internal/catalog`、`internal/thumbnail` 与 `web`
- Change Record：[CR-2026-009](../changes/CR-2026-009-frontend-prototype-fidelity.md)
- Scope：[MVP revision 4](../releases/MVP-2026-07-23-scope-r4.md)
- 详细实施方案：[生产前端与原型一致性改造方案](../frontend-prototype-fidelity-plan.md)
- 开发清单：[FTR-UIF-001 开发任务清单](frontend-prototype-fidelity-task-list.md)
- Architecture Gate：[UIF-S0 Architecture Ready](../gates/MVP-2026-07-23/uif-s0-architecture-ready.md)
- Contract Gate：[UIF-S1 Contract Ready](../gates/MVP-2026-07-23/uif-s1-contract-ready.md)
- Backend Gate：[UIF-S2 Backend Evidence Ready](../gates/MVP-2026-07-23/uif-s2-backend-evidence-ready.md)
- Consumer/UI Gate：[UIF-S3 Consumer/UI Ready](../gates/MVP-2026-07-23/uif-s3-consumer-ui-ready.md)
- Integrated Gate：[UIF-S4 Integrated Slice Done](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md)
- Integrated evidence：[UIF-401](../evidence/uif-401/README.md)、
  [UIF-402](../evidence/uif-402/README.md)、[UIF-403](../evidence/uif-403/README.md)、
  [UIF-404](../evidence/uif-404/README.md)、[UIF-405](../evidence/uif-405/README.md)、
  [UIF-406](../evidence/uif-406/README.md)、[UIF-407](../evidence/uif-407/README.md)、
  [UIF-408](../evidence/uif-408/README.md)
- 风险：[R-021](../risk-register.md)，并复用 R-010、R-012、R-015、R-016

## 用户问题与结果

现有生产 React 前端已经连接真实认证、媒体库、扫描、浏览、搜索、预览和查看器，但应用壳、
管理中心、工具栏和页面密度与已确认原型存在明显差异。同一能力在不同页面使用了不一致的
导航层级，管理功能仍集中在一个页面，当前目录关键字不能完整过滤所有分页目录，账户修改与
手动缓存清理没有生产合同。

该 feature 交付以下结果：

1. 已批准的生产页面在布局、样式、交互层级和响应式行为上与 Apple redesign 原型一致。
2. 认证后页面共用一个全局 Header；搜索、管理入口和页面级导航不再重复。
3. 通用、媒体库、扫描与缓存、账户成为四个独立且功能完整的生产页面。
4. 浏览页同时支持全局搜索和当前目录关键字过滤；目录过滤对全部 cursor 页生效。
5. 视觉一致性拥有可重复、可阻断的截图 Gate，而不是依赖人工印象。
6. 所有新写操作继续遵守后端优先、原媒体只读和单一 capability owner。

## 需求

### 新增需求

| ID | 需求 |
| --- | --- |
| `FR-BRW-010` | 浏览页必须提供当前目录关键字过滤，并对当前目录的全部直接子目录和符合当前 recursive 范围的媒体生效；目录与媒体查询均使用可靠索引、稳定 cursor 和统一 URL 状态，不能只过滤客户端已加载页。 |
| `FR-UI-010` | 生产认证后页面必须使用已确认原型定义的全局 Header、页面壳、管理导航、视觉 token 和响应式层级；同一页面不得出现重复 Header、重复主导航或无行为的静态控件。 |
| `NFR-UIF-001` | 生产页面必须在 1440×900、1265×800、768×1024、390×844 的同主题、同语言、同状态比较中关闭全部 P0/P1/P2 视觉差异；主要区域几何偏差不超过 2px，并由稳定视觉回归阻断后续漂移。 |

### 复用需求

- `FR-AUTH-005`：修改显示名称和密码，验证当前密码并撤销其他会话。
- `FR-UI-009`：通用、媒体库、扫描与缓存、账户四个独立功能页。
- `FR-BRW-001～009`、`FR-SRH-001～004`、`FR-MED-003～008`。
- `FR-UI-001～007`、`NFR-ACC-001`、`NFR-PRIV-001`。

## 范围

### 包含

- 认证页、欢迎页、浏览、搜索、非模态预览、完整查看器和管理中心的生产视觉还原；
- 正式 Logo、全局 Header、大尺寸全局搜索和去重后的管理员菜单；
- `BrowseShell`、`ManagementShell` 和 Search 无侧栏模式；
- 通用、媒体库、扫描与缓存、账户四个独立路由；
- 当前目录关键字的目录和媒体服务端查询；
- 管理员显示名称与密码修改；
- 缓存用量摘要与“清理可重建缓存”的最小有界操作；
- 中英文、浅色/深色、四档响应式、键盘、焦点、reduced-motion 和三浏览器回归；
- 同视口原型/生产组合比较与 Linux-owned 视觉基线。

### 不包含

- 缺失缓存补齐、全部重建和统一任务中心；
- 系统维护、完整性检查、应用数据备份、恢复和诊断包；
- AI 语义搜索、OCR、人脸识别；
- 多管理员、角色、会话设备管理、邮件恢复或外部身份提供商；
- 上传、转码、回收站、原媒体编辑；
- 新部署单元、独立 worker、Redis、外部数据库、WebSocket 或新主题机制。

## 产品与页面合同

### 全局 Header

- 左侧：正式 `BrandMark` 和 FolioPath 字标；
- 中间：视觉居中的全局搜索，提交到全部媒体库或用户明确选择的范围；
- 右侧：管理员菜单，只提供“管理中心”和“退出登录”；
- 浏览侧栏不再重复搜索、媒体库设置或账户导航；
- 每个认证后普通页面只有一个全局 Header。

### 浏览

- 桌面目录侧栏只包含媒体库切换器和目录树；移动端使用抽屉；
- 面包屑独立于全局 Header；
- 工具栏左侧为“包含子目录”和“全部 / 图片 / 视频”；
- 右侧为当前目录关键字、布局和排序；
- 关键字写入 URL，同时绑定目录 cursor 和资产 cursor；
- 只有一个主要垂直滚动容器，列表底部不得出现高度计算产生的空白区域；
- 右侧非模态预览保持父列表可操作。

### 搜索

- Search 页面不挂载重复主侧栏；
- 全局 Header 负责随处进入搜索，页面命令区负责编辑范围、类型、日期和排序；
- 复用唯一 `SearchInput`、`MediaCollection`、`MediaPreview` 和 URL codec；
- 无查询、无结果、离线、首屏失败和后续页失败分别表达。

### 管理中心

- `/settings/general`：主题、语言、布局与已批准的本机浏览偏好；
- `/settings/libraries`：现有媒体库完整生命周期；
- `/settings/storage`：扫描计划、扫描状态、缓存配额、缓存摘要和最小安全清理；
- `/settings/account`：显示名称、密码和退出；
- 桌面导航宽度 216～280px，内容占满剩余宽度；
- `≤ 720px` 变为横向分类条；
- 刷新、前进后退和复制 URL 保持当前模块。

## 后端合同

### 账户维护

S1 已接受、S2 已实现并验证：

- 管理员资料更新 operation；
- 密码修改 operation：当前密码、新密码；
- 当前密码错误、输入无效、并发变化、限流和认证失效的稳定错误；
- CSRF、防缓存和请求体限制；
- 成功后当前会话继续有效，其他会话原子撤销；
- 响应和日志不包含当前密码、新密码、hash、Cookie 或 CSRF secret。

`users` 与 sessions schema 复用显示名称、password verifier 和 auth version，并由只追加
migration 13 增加 account revision；没有修改已发布 migration。

### 当前目录关键字

- 资产查询复用现有 `directoryId + recursive + q + kind + sort + order`；勾选“包含子目录”
  后，当前目录关键字必须查询该目录及全部后代媒体，媒体库根目录也保持相同语义；
- 目录列表新增规范化 `q`，与 `libraryId + parentId + sort + cursor` 绑定；
- 目录匹配只查询可靠索引，不在请求时遍历文件系统；
- 离线媒体库继续查询最后可靠目录索引；
- query 改变后旧 cursor 返回 `invalid_cursor`；
- 10k 目录档已证明查询计划、内存和响应有界；实现使用 migration 13 的目录搜索键与既有
  parent-scoped browse index，没有引入目录 FTS 或客户端全量过滤。

### 缓存摘要与最小清理

- 只暴露可重建缓存的使用量、配额、水位/压力状态和安全摘要；
- 清理 operation 需要认证、CSRF 和幂等键；
- 清理只触发 thumbnail/cache owner 的有界 LRU eviction，不接受路径或任意文件选择；
- 优先保护 SQLite、临时写和安全磁盘余量；
- 失败或重启不能删除配置、索引或原始媒体；
- 本 feature 不提供 missing/all rebuild，也不建立任务中心公共历史。

精确路径、DTO、ETag、错误码和异步表示已由
[UIF-S1 Contract Ready](../gates/MVP-2026-07-23/uif-s1-contract-ready.md)写入
`api/openapi.yaml`；本文件不覆盖权威合同。

## 模块与唯一所有权

| 行为 | 唯一 Owner | 约束 |
| --- | --- | --- |
| 当前密码验证、hash、管理员资料、会话撤销 | `internal/auth` | API/SQLite adapter 不复制认证规则 |
| 目录关键字语义、cursor payload、排序 | `internal/catalog` | handler 不查询 SQLite |
| 目录/资产查询实现 | `internal/store/sqlite` | 实现 capability-owned interface |
| 缓存摘要、配额、水位和清理资格 | `internal/thumbnail` | 不开放路径或原媒体写入 |
| HTTP、DTO、认证、CSRF、错误映射 | `internal/api` | 只调用 service |
| 生命周期与依赖组装 | `internal/app` | 不在 handler 创建 worker |
| 全局 Header、壳和共享视觉模式 | `web/src/components/patterns` | feature 不复制共享组件 |
| 原语、token、主题、品牌 | `web/src/components/ui`、`web/src/styles` | 单一 owner |
| URL/query/invalidation | 对应 web feature 与共享 lib owner | 只通过生成 client |

该 feature 不改变部署、核心技术、信任边界、持久化类型或依赖方向，当前不需要新 ADR。若 S1
决定引入外部搜索服务、独立 worker、新数据库或新主题体系，本 S0 Gate 失效并要求 ADR。

## 数据、安全与可靠性

- 原媒体始终只读；所有新端点只接受 ID、受控枚举和文本 query；
- 目录搜索只作用于 SQLite 可靠索引；
- 密码修改使用现有 Argon2id 与统一凭据错误，不形成账号枚举；
- session 撤销在短事务内完成，不能出现密码已变但旧会话无限有效；
- 缓存清理只删除 `/app/data` 下可重建缓存，并保留磁盘安全余量；
- 不向浏览器暴露 host path、SQL、stack、stderr、hash 或 secret；
- URL 不保存密码、CSRF、绝对路径或内部 cursor payload；
- 视觉 fixture 使用合成媒体，不读取开发者真实媒体库。

## 视觉真相与防漂移

视觉真相按以下优先级解析：

1. `prototypes/apple-redesign` 对应页面和明确交互状态；
2. `prototypes/apple-redesign/qa` 中同状态确认截图；
3. `docs/ui-design.md` 的行为、响应式和可访问性约束；
4. 中心 token 和共享组件实现。

如果原型与产品/安全/API 合同冲突，先修正文档和原型，不能在生产代码中静默选择。

每个页面至少比较：

- 1440×900；
- 1265×800；
- 768×1024；
- 390×844；
- 浅色/深色；
- 简体中文/英文；
- 正常及适用的加载、空、离线、失败、冲突、取消和处理中状态。

设计 QA 必须把原型截图与生产截图放入同一比较输入，关闭所有 P0/P1/P2。主要区域几何偏差
不超过 2px；P3 可留下明确记录，但不能影响布局、换行、可访问性或主要操作。

## 验收

| ID | 验收 |
| --- | --- |
| `UIF-AC-001` | 所有认证后普通页面只有一个全局 Header；搜索和管理导航没有重复入口。 |
| `UIF-AC-002` | 通用、媒体库、扫描与缓存、账户均为独立 URL，刷新/历史/复制恢复当前页。 |
| `UIF-AC-003` | 管理页面的主要按钮连接真实操作；合同缺失时不显示假成功或无行为控件。 |
| `UIF-AC-004` | 修改显示名称和密码通过真实认证 API；错误保留非秘密输入，成功撤销其他会话。 |
| `UIF-AC-005` | 当前目录关键字对全部直接子目录和资产 cursor 页生效；URL 恢复且旧 cursor 失败关闭。 |
| `UIF-AC-006` | 缓存清理只作用于可重建缓存，盘满/失败/重启不影响索引、配置或原媒体。 |
| `UIF-AC-007` | 浏览页顶部、中段和底部均无额外空白区、嵌套滚动或隐藏持续控制。 |
| `UIF-AC-008` | 浏览、搜索和查看器复用唯一媒体集合、预览 controller 与查看器模式，固定预览和返回焦点不回退。 |
| `UIF-AC-009` | 1440、1265、768、390 四档同状态比较无 P0/P1/P2，主要几何偏差不超过 2px。 |
| `UIF-AC-010` | 浅色/深色、中英文、键盘、触摸、focus、reduced-motion、axe 与三浏览器无阻断回归。 |
| `UIF-AC-011` | 100k 媒体/10k 目录档仍使用稳定 cursor 和有界 DOM；目录过滤不退化为全量客户端加载。 |
| `UIF-AC-012` | 全流程不修改原始媒体，不泄露绝对路径、SQL、stack、密码或会话秘密。 |

## 失败与恢复矩阵

| 场景 | 必需行为 |
| --- | --- |
| session 过期 | 返回登录并保留安全返回位置，不展示陈旧管理表单成功 |
| 当前密码错误 | 统一安全错误；不修改 verifier 或撤销会话 |
| 密码修改事务失败 | 当前密码继续有效；其他会话状态不进入半撤销 |
| 目录 query/cursor 不匹配 | `invalid_cursor`；客户端重取第一页 |
| 媒体库离线 | 目录/媒体过滤使用最后可靠索引并标记 offline |
| 缓存清理盘满或中断 | 停止清理、保留 DB/索引和 ready 状态的一致性，可安全重试 |
| API 首屏失败 | 显示可恢复错误，不伪装成空内容 |
| 后续 cursor 失败 | 保留已载入内容，提供重试 |
| 窄屏导航打开 | 明确关闭、Escape、scrim 和焦点恢复 |
| 视觉回归基线变化 | PR 必须解释来源、更新参考或修复漂移，不能批量无解释接受 |

## Gate

```text
UIF-S0 Architecture Ready
  → UIF-S1 Contract Ready
  → UIF-S2 Backend Evidence Ready
  → UIF-S3 Consumer/UI Ready
  → UIF-S4 Integrated Slice Done
  → Stage 5 RC Gate rerun
```

- S0、S1、S2、S3 与 S4 已 Go；`UIF-401～408` 已完成并有实际证据；
- 账户修改、全量目录过滤与缓存清理已通过生成 client 接入真实 Backend Ready owner；
- reference manifest、token、壳、Storybook、逐页比较、Linux 基线、真实纵向链、候选浏览器、
  可访问性、100k/10k 容量和完整仓库验证已完成；
- feature 可称为 Integrated Slice Done，但不得称 MVP/RC 可发布；
- 受影响的 Stage 5 浏览器、容量、容器和 RC 聚合已重跑；最终 digest、物理辅助功能和
  供应链阻断保持 No-Go。
