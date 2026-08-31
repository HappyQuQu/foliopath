# INT-015：人脸与人物库隐私评审草案

## 状态

**Needs privacy/legal sign-off / INT-015 未完成。** 本记录是产品与工程控制矩阵，不是法律意见，
也不代表任一部署者已经取得处理照片中人物人脸信息的合法依据。

- 日期：2026-08-27
- Target：`POST-MVP-5` Slice E（匿名人脸聚类与用户人物库）
- Owner 角色：privacy、security、face、operations、product
- 需求：`FR-INT-009～014`、`NFR-INT-001～005`
- 相关风险：`R-027～R-029`
- Scope：[POST-MVP-5 scope proposal](../../releases/POST-MVP-5-scope-proposal.md)

## 适用边界与外部依据

FolioPath 是自托管软件，实际部署者通常决定媒体来源、处理目的和保留期限；项目不能替部署者判断
每张照片中人物的授权或其他合法处理依据。个人/家庭、组织内部、商业使用、公开图库以及不同法域的
规则可能不同，不能在产品中做统一的“点击即合规”承诺。

在中国境内适用场景中，官方《个人信息保护法》把生物识别列为敏感个人信息；国家网信办、公安部的
[《人脸识别技术应用安全管理办法》](https://www.cac.gov.cn/2025-03/21/c_1744174262156096.htm)
自 2025-06-01 施行，要求特定目的、充分必要、最小影响、显著告知、适用时的单独同意、最短保存、
事前个人信息保护影响评估和安全措施。达到 10 万人脸信息存储数量时还存在备案要求；官方另有
[备案公告](https://www.cac.gov.cn/2025-05/30/c_1750315544241157.htm)。这些要求是否适用于某个部署
以及如何履行，必须由部署者和合格顾问判断。项目默认按高敏感数据设计，但安全设计不替代合法依据。

## 数据清单与分类

| 数据 | 分类 | 存储提案 | 可重建 | 允许的用途 | 禁止 |
| --- | --- | --- | --- | --- | --- |
| 原媒体中的脸部像素 | 原媒体事实/可能含敏感个人信息 | 仍只在 `/library:ro`，不复制原图 | 否 | 仅受控读取以生成派生结果/预览 | 修改、移动、上传、诊断打包 |
| 瞬时对齐/裁剪张量 | 高敏感临时数据 | 只在有界内存，不落盘 | 是 | 单次 detector/embedding | 缓存 face crop、写日志、网络发送 |
| face box、quality、fingerprint | 高敏感派生数据 | `/app/data` SQLite，按 library/asset/generation | 是 | 定位、质量过滤、重建 | 公开 API 批量导出、诊断打包 |
| face embedding | 高敏感生物特征派生数据 | `/app/data` SQLite；原始向量不经 API | 是 | 本实例相似度/聚类 | 外部人物库、云查询、日志、遥测、分享链接 |
| anonymous cluster/core/edge | 假名化派生关系，不视为匿名化 | `/app/data` SQLite，按 generation | 是 | 用户复核候选 | 描述为身份事实、自动命名 |
| person ID/name | 用户应用状态/可能为个人信息 | `/app/data` SQLite | 否 | 用户建立的人物库 | 模型预测姓名、外部身份补全 |
| manual assignment/exclusion/cannot-link | 用户应用状态/敏感关系 | `/app/data` SQLite | 否 | 约束后续聚类和纠错 | 模型升级静默覆盖 |
| 查询、日志、诊断 | 潜在敏感操作数据 | 查询不持久化；日志仅稳定码/计数 | 不适用 | 故障定位 | 人名、向量、face crop、原路径、相似 pair |

“匿名组”只是界面中尚未命名的假名化记录；它仍可通过原图片或后续用户命名关联到自然人，产品文案
不得称其为已匿名化数据。

## 产品控制矩阵

| 场景 | 必须行为 | 禁止/失败关闭 | 证据 Gate |
| --- | --- | --- | --- |
| 默认状态 | 每个库默认关闭人脸分析；普通浏览不依赖它 | 首次启动/扫描自动启用 | S1 contract、S3 UI、S4 E2E |
| 启用前告知 | 显著说明本地处理、数据类型、目的、保留/删除、备份影响、风险和部署者责任 | 仅写“AI 功能”或把同意捆绑到普通浏览 | privacy sign-off + S3 |
| 合法依据 | 管理员确认自己有权启用；UI 链接部署者自定义隐私说明 | FolioPath 声称替照片中每个人取得同意 | privacy/legal owner |
| 访问 | 只有已认证管理员可看匿名组、人物、face 位置和纠错界面 | 匿名 LAN、分享链接、未授权 API | S1/S2 security tests |
| 本地边界 | inference 无网络；模型下载与 inference admission 分离；媒体/裁剪/向量/查询不出站 | 复用下载 client 上传内容或调用云 API | S2/S4 network evidence |
| 建库 | 先匿名 core/edge，名称由用户输入；edge 逐 face 确认 | 自动姓名/身份、edge 批量吸收、自动合并已命名人物 | quality + transaction tests |
| 跨库 | face rows 始终按 library 隔离；是否允许一个 person 关联多库必须在 S1 明示 | 删除一个库时误删其他库 face，或隐藏跨库关联 | S1 product/data decision |
| 禁用 | 停止新任务和 session；是否保留已有派生结果须由显式选择决定 | 禁用后仍后台处理 | S1 contract + S2 integration |
| 清除 | 支持按库清除 observations/embeddings/clusters；人物人工状态的保留/删除必须单独、明确选择 | 用“清除缓存”模糊删除人工人物关系；触碰原媒体 | S1 decision table + S2/S4 |
| 备份 | 明示完整 `/app/data` 备份会包含 embedding、person name 和人工关系；恢复保持 generation/关系一致 | 把备份说成不含人脸数据；诊断包夹带数据库 | deployment/security + S4 |
| 诊断 | 只含 model ID/version/hash、状态、计数、稳定错误、资源指标 | face crop、向量、名字、query、媒体路径、raw runtime error | S2 tests |
| 分享/导出 | revision 1 不通过分享链接或诊断导出人物/人脸数据 | 将人物页纳入未来分享而无新隐私评审 | scope/architecture Gate |
| 模型升级 | 新 generation 并行构建；人工关系独立，stale suggestions 失败关闭 | 先删旧代、跨代混排、覆盖 manual/cannot-link | S2 concurrency + S4 rollback |
| 媒体库删除 | 删除该库全部 face 派生数据和关联；原媒体逐字节不变 | offline 被当作删除；跨库人物关系部分损坏 | S2/S4 integration |

## 清除与备份待冻结语义

S1 必须选择并在 UI/API 使用不同动作，不允许一个含糊的“删除人脸数据”按钮同时承担全部语义：

1. `disable analysis`：停止新分析，不删除；普通浏览继续。
2. `clear derived face analysis for library`：删除该库 observation/embedding/anonymous generation，保留或
   拒绝保留人工人物关系必须由下一条规则决定。
3. `delete person`：删除一个人物名称及其 manual assignment/cannot-link 范围；不改原媒体。
4. `clear all face/person data for library`：删除该库派生与人工状态；跨库 person 的剩余关系必须一致。
5. `clear all instance face/person data`：全实例高风险操作，需重新认证、明确影响和可验证完成状态。

当前整库 SQLite 备份会自然包含上述记录。若 revision 1 不实现经过验证的选择性备份，部署文档必须
把备份视为同等级敏感数据，并要求部署者控制访问、加密存储、保留期限和销毁；不得承诺 FolioPath
自动加密宿主备份。

## 真实评测数据 Gate

- `INT-005B/007B` 的 ground truth 必须记录来源、许可/授权、允许的评测用途、保留期限、访问者和删除方式。
- 不得用用户真实媒体、开发者私人图库或生产 `/library` 自动生成测试集。
- 未成年人、私密场景、偷拍或来源/授权不明的数据不得进入仓库或共享 evidence。
- 仓库只提交不可逆的汇总指标、合成 fixture 或经审查可再分发的小型样例；原始真实 face 数据置于
  访问受控的评测环境，不能进入公开 CI artifact、日志或模型包。
- “照片有公开许可”不自动证明可以建立身份/生物特征 ground truth；privacy/legal owner 必须单独签署。
- 隔离 spike 的 dataset manifest v2 已把允许用途、授权访问角色、保留截止日、固定删除动作、禁止再分发、
  不透明授权引用和隐私评审引用变成自动校验；verification/clustering 还必须使用不透明 identity ID。
  仓库模板中的 `*-REQUIRED` 是占位符，不能当作签署证据，也不允许将实际姓名、授权文件或绝对路径写入 manifest。

## 未决问题与阻断结论

- [ ] 部署者告知/合法依据模板及各目标市场的适用性经 privacy/legal owner 评审。
- [ ] instance-level person 是否允许跨库关联，以及库删除后空人物语义冻结。
- [ ] 禁用时保留派生结果还是提示用户选择，API/任务语义冻结。
- [ ] 五种清除动作、重新认证、审计和恢复边界冻结。
- [ ] 完整备份的敏感数据提示、加密责任、保留/销毁说明冻结并实测。
- [ ] 生产诊断/日志/API/支持包的禁止字段形成 executable test；隔离 spike 已有封闭 DTO 与序列化回归，
  但没有生产接入，不能勾选本项。
- [ ] 合法真实 ground truth 的来源、访问、保留、删除和签署完成。
- [x] 真实人脸数据进入评测链路前的 manifest v2 结构和拒绝规则已有自动测试；这不关闭上一项。
- [ ] SFace 或替代 face embedding 的许可/来源获合规批准。
- [ ] 适用时的 10 万人脸信息备案/记录义务由部署者评估，产品不作错误豁免承诺。

在这些项目关闭前，`INT-015`、Slice E 和 INT-S0 人脸范围保持 **No-Go**。若无法取得合法评测数据、
隐私签署或合格权重，强制 fallback 是从 Frozen revision 1 删除 Slice E；图片/视频语义搜索不能借此
获得人脸数据处理授权。
