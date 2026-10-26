# CR-2026-033：中文标签与实用类别

- 状态：用户接受方案；中文/双语展示与人工保护已上线。153类策略经真实Docker换代回归与停写全备份后更新6010，**2026-10-25 05:54四库新身份终态核验8690成功/10已知源失败/0待处理，无新增任务级错误，保护摘要一致**；见[部署记录](DEPLOY-2026-10-24-photo-tags-153.md)。非质量/发行通过。
- 范围：POST-MVP-5，FR-AUTOTAG-004（语言展示/别名）、FR-AUTOTAG-005（受控类别扩充）。
- Owner：Codex；沿用 CR-2026-031、ADR-0021 的自动任务/人工保护边界。

## 接受的行为与契约

1. AI 类别保留稳定英文 key 和标签 ID；中文、英文、双语是浏览器展示偏好，不是逐库重新识别设置。
   默认跟随界面语言；偏好在当前浏览器保留。切换语言不写数据库、不触发推理。
2. `Tag`/`AssetTag` 增加可选 `localization`（key、zhCN、en）。仅确认为 AI 的关联可本地化；
   全局词表存在人工关联或没有 AI 关联时保持原文。混合来源的同 ID 在单图尊重该关联来源。
   旧客户端继续使用 `name`；新客户端缺少元数据时原文回退，不猜测来源。
3. GET tags 的 q 在原始名称之外匹配固定中英文别名，仍保持库/种类范围、稳定 keyset 和上限。
   不将人工原文改名或强制合并；用户刻意创建的不同名称允许共存。自动输出始终使用唯一类别 key。
4. 人工原文与关联优先；AI 合并识别固定词表中的中英文同义人工名称，不再添加重复 AI 类别。
   移除这些名称产生的抑制对同类别的后续策略同样生效；显式手动恢复仍通过既有人工操作。
5. 固定扩充客观服饰、物体、场景类别，保持阈值 0.4、每图最多 20 和单调度器/进程生命周期。
   类别/策略改变使用新独立标签身份，不修改搜索/人物身份或把旧完成数标为新策略完成。
   自动补齐只发生在已启用库；部署前保留活动任务进度，不重复全量提交。

## 交付顺序与验证

先接受 API/策略合同 → 后端投影、别名/人工保护测试 → 生产组件/Storybook 同步 → 合成 Docker
真实模型回归（语言、旧名称/ID不变、人工优先、删除抑制、重跑/重启、源 hash）→ 更新证据。
普通照片质量、阈值校准、完整发行与大容量验收不由此功能烟测替代。

## 进度

| 切片 | 状态 |
| --- | --- |
| T1 契约、类别词表与兼容投影 | 已完成并通过专项回归 |
| T2 中文/英文/双语偏好与生产组件 | 已完成，正式组件和 Story 同步 |
| T3 独立扩充策略身份与共享调度 | 已完成，153 类；搜索/人物身份不变 |
| T4 本地 Docker、文档和候选预览 | 通过，独立合成实例可查看 |
| 原 6010 主应用与中文标签 | 已更新 e10d3bef；原数据/任务保留，正式页面中文查找/语言切换通过 |
| 原 6010 新 153 类策略补齐 | 已结束：AI52f4a40c/新cf642648身份四原任务均succeeded/attempt1/error=null，8690成功/10暂留源失败/0待处理；22PNG/保护摘要/数据库复核通过 |

## 历史：6010展示更新（2026-10-23 16:47–16:51 +08:00）

[计划升级交接修复](FIX-2026-10-23-planned-job-handoff.md)已通过 Go/race、真实合成 Docker 后部署。
原主应用已正常停写全备份，精确交接两个 running claim，再单次替换为
`sha256:e10d3bef357b6902e13bfd3551b645d3ac79cb395f7998f15bb12b136491c004`（schema 44）。
AI `3631b432`、模型身份、媒体/data 挂载和不限内存覆盖保持原状。双服务 healthy、RestartCount=0，
容器内 database-check 与数据保留摘要通过；无新提交/全量重建/清理原库任务。
coser 原自动标签 operation 从已处理 1672 续至 1704，pixiv 原语义 operation 从 464 续至 496，
两者原 ID 和断点保留。本数值是 16:51 快照/已处理数，不是全库成功数或终态。

正式标签云中文“汽车”查到原 `tag_33`；English 为 `Car`，双语为 `汽车 / Car`，仍 5 张图片、同 ID。
测试后恢复跟随界面语言、清除筛选。旧标签不改名、不需重跑；非空人工保护在合成库验证，
真实库核对时人工关联/收藏/人物整理为 0，没有添加测试数据。备份和逐项保留证据见上方维护记录。
**该历史时间点仅UI/中文标签上线，153类尚未激活；后续2026-10-24安全切换见页首新记录。**

## 历史：原 6010 更新前核验（2026-10-23 16:17 +08:00）

用户要求保留原数据和任务进度更新 6010。核验发现两个不能忽略的限制，未直接停止或替换服务：

1. coser 自动标签仍使用旧 92 类策略运行，直接启用 153 类身份会令原任务因
   `model_unavailable` 失败。中文/双语展示可先随主应用更新，153 类识别策略必须另行安全切换，
   不能将旧策略完成数标成新策略覆盖率。
2. 原任务 `autotag_0e6072a5243a9a0c7ed8947e2fce74ed` 的 `attempt_count=3`。
   现有主应用退出保留 running 断点，重启后 `RecoverAutomaticTagJobs` 对到期且 attempt>=3
   的任务终结为 `operation_interrupted`，而非继续排队。因此本轮连“只更新主应用”也暂缓。
   不通过直接改计数、重置任务或新建全量任务绕过这项限制。

同一应用容器内执行固定只读任务查询（`mode=ro`、`query_only=1`，10 秒超时），快照如下；
checkpoint 是资产 ID，不是已处理数量，以下不是全库完成声明：

| 能力 / 库 | 原 operation ID | 状态 | 本任务 completed_items | checkpoint | attempt |
| --- | --- | --- | ---: | ---: | ---: |
| 自动标签 / coser | `autotag_0e6072a5243a9a0c7ed8947e2fce74ed` | running | 1375 | 2274 | 3 |
| 语义 / 4k | `aio_cc93bd76daf70e00683bf2371a72c40e` | queued | 368 / 707 | 6589 | 2 |
| 语义 / pixiv | `aio_479074a1d4d99ad663dd86f2087b1d8d` | running | 160 / 999 | 7880 | 1 |
| 语义 / coser | `aio_08d77ae32263d0ada16a64c6117051e6` | queued | 0 / 6202 | 0 | 0 |

前后两次查询 coser 自动标签 completed_items 从 1355 增至 1375、pixiv 语义从 144 增至 160，
说明任务仍有实际进展，不将 queued/running 记作完成。此查询仅取活动任务，未复核其他终态。
临时诊断可执行文件已移除，未从 macOS 打开活跃 SQLite，未执行迁移或写任务表。

原主应用仍为 `1806724d`，AI 为 `3631b432`；双服务 healthy、自动 RestartCount=0，
容器 RAM/swap cap 均为 0，媒体挂载和 `/app/data` 原位置不变。
`docker exec foliopath-ai-original-review-foliopath-1 /app/foliopath database-check` 实际通过。
本轮未停止服务、未改 `ai.env`、未启动新任务、未操作原件或人工整理、未连接 server222；
**尚未进行停写备份、版本替换及新版 6010 验收**，候选合成测试不替代这些步骤。

当时待决策：等待旧任务结束后更新，或先实现并合成验证受控的计划升级任务交接，再更新；
不可把现有 crash 恢复机制当作已支持无损计划升级。

## 实现与已知语义

- `internal/curation/tag_categories.json` 唯一拥有 153 类（保留旧 92，新增 61）的 key/名称/别名；
  `tag_localization.go` 拥有投影、别名检索与人工同义保护。未增加迁移、不改名或合并原数据。
- `Tag.localization` / `AssetTag.localization` 是可选元数据；OpenAPI 客户端已 regenerate，
  revision 摘要经新增可选字段与检索语义复核后同步。缺少元数据时客户端显示原文。
- 全局混合来源的同一个 tag ID 仍显示原文；其中 AI 单图关联可以本地化，人工关联保持原文。
  “将 AI 保留为人工”之后显示该标签的原始存储名称，不把用户操作变成隐式改名。
- 用户手动新建的不同同义名称仍可共存；AI 不在同一张图重复添加同类别。不是任意自由文本语义合并。
- 新策略仅扩充输出过滤，不改变 JoyTag 权重、448px 输入处理、0.4 阈值、每图最多 20。
  不增加大模型、常驻会话或另一个调度器。显示文字/别名不参与识别策略摘要，改展示不导致模型换代；
  类别 key 集合和实际执行源分别绑定 policy/runtime 摘要。
- 首轮候选仅核验多语言 XLM 基础镜像；未宣布旧 English 512 组合的独立新 tag runtime 已可发布。

## 实际验证（2026-10-23，局部功能验收，不是发行）

- Go：`go test -count=1` 的 `internal/curation`、`internal/store/sqlite`、`internal/api`、
  `internal/autotag`、`internal/inference/remoteml`、`tests/contract`、`tests/architecture` 通过；
  新 DTO/契约/架构测试添加后再次定向通过。`GOTOOLCHAIN=go1.26.5 CGO_ENABLED=0`。
- `go test -race -count=1 ./internal/curation ./internal/store/sqlite -run 'TestLocalized|TestAutomaticTag'`
  通过；相应三包 curation/api/sqlite 的 `go vet` 通过。
- `make fmt-check arch-check generate-check` 通过；此前 fmt-check 因新增测试尚未 gofmt 失败，
  按范围格式化后通过，未批量格式化用户其他改动。
- Node 22.22.2：最终前端 **67 文件 / 414 项通过**；类型、前端架构、视觉引用清单、OpenAPI lint
  和 Storybook 构建通过。lint 保留两个 health 缺少 4XX 的既有 warning；清单检查不是像素矩阵。
- Python：固定 AI 镜像内 **36 项通过**（挂载测试源），包含全部类别实际存在于固定模型标签表、
  fail-closed 输出、身份/快照/队列回收及 **新 ASGI overlay 的真实 TCP 断开回收**。
  首次 discover 的工作目录错误导致 0 tests，已更正 `/opt/foliopath-inference` 后重跑，不计作通过。
- 本地 Docker 两轮通过，合成 3 张有效 PNG / 1 张故意损坏 JPG、两库、固定真实权重；原始媒体挂载只读。
  第二轮 `fp-localized-tags-5d8328c1` 额外实际执行 XLM 中文搜索与 legacy 人脸推理，确认共享调度运行；
  各能力在 alpha 为 2 成功 / 1 输入失败，beta 自动标签关闭。该失败是故意负例，不是程序新错误。
  真实 JoyTag 返回新增 `sweater`；仅证明新类别链路，不声称普通照片准确率或阈值校准。
- 中文别名/库隔离、稳定名称/ID、人工原文、中英文同义优先、中文移除后重跑不补回、显式人工恢复、
  多次真实重跑、显式主应用重启、源文件 SHA256 不变及容器内 `database-check` 均通过。
- 第二轮 AI peak **2,760,032,256 bytes**，本轮 4 GiB cgroup 中 max/oom/oom_kill/oom_group_kill 均 0；
  两个容器自动 RestartCount=0。显式 restart 不冒称未重启；此证据不覆盖大图/长期/默认 2304 MiB 档。
- 浏览器正式组件：中文标签查“毛衣”返回同一 sweater ID；中英文/双语即时切换，刷新保留，设置页同步。
  原文 `grey_background` 人工标签保持英文；人工中文保留。390/768/1440/1920 的 scrollWidth=clientWidth
  （375/753/1425/1905），截图在 `docs/evidence/ui-audits/localized-tags-2026-10-23/`。
  初轮四档截图对应相同候选镜像，最终固定端口补有中文搜索和设置页截图；最终浏览器 error 日志为空。
- 未执行完整 `make lint/test/test-integration/test-e2e`、三浏览器/双主题全矩阵、AMD64、质量标注校准、
  大容量或完整发行检查，不将以上局部结果记作全部门禁通过。

## 固定候选与查看入口

- 主应用 `sha256:3a322ee9b6db2c6a425fe3a4e04026b1d66a6e541b8cdada4dfff8c7376946f9`。
- AI `sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626`。
- 标签 policy `a2f8b363f83db8d76cbc903bb532d2eb90c985d6eced957ec9b68cc80b6eb628`，
  runtime `71ae7a7694c66ca3e4b19cb1fda4a4a5d0ba2344cd8fb5d796b6c0acc8229d90`；权重与 cache 摘要未变。
- 增量 native 主镜像构建源 `spikes/automatic-tags/Dockerfile.localized-app`，不是完整供应链构建。
  AI 构建源 `services/inference/Dockerfile.photo-tags`，基于已验证 XLM 镜像，未覆盖旧能力源文件。
- 预览：[本地合成标签库](http://localhost:55103/tags/tag_6?libraryId=lib_1&kind=image)，
  [自动标签设置](http://localhost:55103/settings/ai?library=lib_1)。已在本聊天浏览器登录。
  仅测试账户：`localized-tag-test` / `Synthetic-Localized-Tags-2026!`，不是 6010 的账户。
- 第二轮完整合成数据保留在
  `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-localized-tags-5d8328c1-Bf6ldr`。
  首轮 `fp-localized-tags-3851a0c5` 的两个容器、网络和专用 scratch volume 已核验归属后清理；
  首轮合成数据与镜像保留，可重建，不影响最终预览或真实媒体。
- **首轮候选验证阶段未替换 6010、未修改 ai.env、未提交/重试真实库任务、未连接 server222。**
  后续先完成上述展示分阶段部署，再于2026-10-24 21:49按新记录安全切换153类tag.identity；不改写此处首轮候选历史证据。

## 复查中修正

首次 API 契约测试因 revision lock 未更新失败，复核为可选字段增量后同步摘要并重跑通过。
前端全量测试发现收藏数量到达前即断言的异步时序问题，改为等待真实计数，保留原断言而不删测试。
首个 AI build 使用裸 sha256 被当作 registry 名，改为已核验本地名称@digest 后构建；没有拉取未知镜像。
类别计数实际为 153，早期口头 154 已更正，不把未开放的其余 JoyTag 类别声称已支持。
