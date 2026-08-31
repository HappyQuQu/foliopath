# INT-019/021：模型分发、存储与恢复决策草案

## 状态

**Proposed / 未冻结。** 技术状态机已有 arm64 隔离证据，但真实发行源、镜像运营、selected model、
native amd64、生产合同和 release/compliance owner 尚未签署，因此 `INT-019`、`INT-021` 保持未完成。

- 日期：2026-08-27
- Target：`POST-MVP-5` Slice A
- Owner 角色：aimodel、security、release、compliance、operations
- 需求：`FR-INT-017～020`、`NFR-INT-007～010`
- 风险：`R-025`、`R-030`

## 两层模型

必须区分“从哪里取得包”和“验证后从哪里运行”，UI 不得把它们混成一个任意文件选择器。

### 来源层

| 来源 | revision 1 判断 | 条件 | 无条件时的行为 |
| --- | --- | --- | --- |
| 应用镜像内置权重 | Reject | 增大双架构镜像、补丁与撤回耦合；当前没有收益证明 | 不放入 production image |
| 项目签名发行源 | Conditional | 真实 owner、稳定 endpoint、catalog signing/checkpoint/rotation/revocation、SLA/成本 | 没 owner 就不显示“在线下载” |
| 项目运营国内镜像 | Reject until operated | 项目实际运营且复制同一签名包/清单，持续可用并有撤回机制 | 不以 UI 文案或境外反代冒充国内源 |
| 部署者预配置镜像 | Conditional | 仅启动配置给 base URL；对象路径/digest 仍由项目签名 catalog 固定 | 不能由 API/UI 提交 URL，失败回到离线目录 |
| 固定 `/models:ro` | Required baseline | 单一可选只读模型 mount、无 symlink/特殊文件/嵌套 mount，包在内建兼容清单 | 受限网络用户仍可离线安装 |
| 任意 URL/本机文件选择 | Reject | 无产品必要性，扩大 SSRF/路径/凭据/供应链面 | API 永不接受 URL 或绝对路径 |

项目签名发行源与部署者镜像不是两套信任根：镜像只改变获取位置，不能改变 catalog、package digest、
license metadata 或撤回状态。离线包也必须匹配同一受信 catalog/envelope；“用户手动下载”不跳过校验。

### 运行存储层

| 模式 | 路径 | 所有权与修改 | unavailable/恢复 |
| --- | --- | --- | --- |
| managed（默认） | `/app/data/models/<package-digest>/` | aimodel owner 校验后同文件系统原子发布；只删除未 active、无 operation 引用且经确认的包 | 源消失不影响；包本身损坏则 unavailable，恢复相同 digest 或重新安装 |
| direct（显式高级选项） | `/models/<package>` | 部署者拥有；FolioPath 永不写、移动、重命名、删除；每次加载前复核只读/source kind/hash | 消失、可写、hash/source kind 改变即 unavailable；相同包 remount 后可恢复，不自动换代 |

UI 的“选择文件”实际只能选择后端返回的 opaque candidate ID；浏览器看不到容器/宿主路径。direct
模式必须显示空间收益与恢复责任，不能成为默认，也不能从 `/library` 或任意目录运行模型。

## Canonical owner 与状态

`internal/aimodel`（提议）唯一拥有 catalog、download/import、package validation、managed/direct
provenance、availability、selected/active generation、operation cleanup 与 reconciliation。HTTP handler、
jobs、inference adapter 和 UI 只能调用该 service，不能复制 URL/hash/path/retry/activation 规则。

```text
catalog candidate
  -> downloading | scanning
  -> verifying
  -> installed(managed | direct)
  -> available
  -> building generation
  -> active

any validation/source failure -> unavailable or failed operation
active pointer changes only after a complete validated generation
```

- catalog checkpoint 与 active pointer 的数据库事务不能持有 filesystem/network work。
- package 先完整写 staging、校验、fsync、no-replace rename、parent fsync，再允许数据库登记 available。
- 文件系统和数据库重启 reconciliation 只登记完整 orphan，不自动激活；missing/corrupt 不删除 registry、
  embedding 或人工状态。
- unavailable 是模型状态，不是整个实例 readiness 失败；普通浏览和字面搜索继续。
- 依赖该模型的语义 query 返回稳定 `model_unavailable`，不能静默使用不匹配代次或换模型。

## 获取、配额与失败语义

- 管理员明确触发；不会因启动、进入设置页或发现缺模型自动联网。
- 下载全实例并发 1，固定 origin/解析后 IP/TLS server name、禁用环境 proxy、禁止越界重定向。
- resume 绑定 catalog digest、URL origin、ETag、长度和已验证 prefix；任一变化重新开始。
- 空间预检至少覆盖 remaining download、staging、managed final（若复制）、新 generation 临时空间、
  SQLite/WAL 和全局安全余量；确切 quota 在 selected package 后由 S1 冻结，不能先写猜测值。
- 取消、超时、错 hash、错架构、撤回、磁盘满、强杀只终止当前 operation；现用 package/generation 不变。
- package 只能含 manifest 列出的数据文件；首版 ONNX 全部 initializer 内嵌，拒绝 external-data、代码、
  plugin、动态库、脚本、symlink/hardlink/device 和未声明文件。

## 备份与恢复

| 数据 | required backup | 恢复语义 |
| --- | --- | --- |
| SQLite model registry/checkpoint/active generation | 是，属于 SQLite family | 恢复后与人工状态/embedding generation 配对；不能单独恢复 active pointer |
| person/manual assignment/exclusion | 是，不可重建应用状态 | 必须与数据库一致恢复；模型缺失不删除 |
| embedding/cluster | 当前整库备份会包含；逻辑上可重建 | 可以保留加速恢复；若未来选择性排除，必须明确重建/coverage |
| `/app/data/models` managed package | 条件必需 | 若项目/部署者仍能取得完全相同 digest 可省略；离线唯一副本必须备份，否则恢复后 unavailable |
| `/models:ro` direct package | FolioPath 不备份 | 部署者恢复相同只读包/挂载；不同 digest 不自动代替 |
| `/app/data/ai-indexes` 可重建 index | 否 | 从 SQLite embedding 重建并原子激活 |
| partial/staging/tmp | 否 | 启动对账后安全清理；不得被视为 installed |

现有“备份整个 `/app/data`”流程会自然包含 managed model 和高敏感 face 数据。S1 部署合同必须说明
空间与敏感性；若增加选择性备份，必须用真实恢复演练证明 model unavailable、reinstall 和 generation
配对，不能简单排除目录后声称完整恢复。

## 升级、撤回和清理

- 选择新版本只创建新 generation；成功验证后单事务切 active，旧 generation 在 retention window 内保留。
- catalog 撤回阻止新安装/激活；对已 active 包是立即禁用还是允许有界迁移期，须由 security/release
  在 S1 按撤回级别冻结，不能让普通模型升级逻辑决定安全事件。
- 清理前必须证明 package 非 active、非 fallback、无 building operation/lease 引用且备份/恢复文案一致。
- direct package 永不由 FolioPath 清理；仅移除 registry/config 引用。
- 删除某库 AI 派生数据不必删除共享模型包；删除包也不能删除人物人工状态或原媒体。

## 诊断与 provenance

允许输出：model/package opaque ID、版本、generation、source kind、package digest、catalog sequence/key ID、
availability、operation stage、稳定错误码、公开 license ID、文件计数/总大小和资源指标。

禁止输出：宿主路径、完整 container path、真实 download/mirror URL、resolved IP、凭据/header、raw runtime
错误、模型内容、face crop/embedding、query 和 person name。发布 provenance 必须显式记录 ORT/native
runtime component、source commit/archive digest/license/notices；普通文件 SBOM 未识别 `.so` 时必须补组件。

## 未决项与冻结条件

- [ ] 项目签名发行源是否进入 revision 1，以及真实 owner/endpoint/key/checkpoint/撤回/SLA。
- [ ] 若提供部署者镜像，配置键、TLS/CA、base URL 拼接和故障回退合同。
- [ ] selected package 的精确大小、managed quota、临时空间和全局安全余量。
- [ ] catalog 撤回对已 active 包的紧急/普通级别语义。
- [ ] managed package retention、删除确认和备份选择。
- [ ] native amd64 与 arm64 完整下载/scan/direct/final-image/SBOM/provenance 证据。
- [ ] SFace hold 或替代 face package 的合规结论。
- [ ] S1 OpenAPI/data/deployment/security/backup/diagnostic 合同。

若冻结时没有真实在线 source owner，revision 1 只能承诺 `/models:ro` 扫描、managed copy 和严格 direct；
“获取模型”界面显示离线安装引导与扫描，不显示不可用的下载按钮。这个 fallback 满足无法访问境外站点
的部署，不虚构项目国内镜像。
