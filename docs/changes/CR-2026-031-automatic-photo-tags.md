# CR-2026-031：自动图片标签与库内标签云

## 已确认范围与交付边界

- 状态：2026-10-22 用户接受 V2 原型，随后确认开始第一项实施；不是功能验收通过。
- 目标：POST-MVP-5 增量，不改变冻结 MVP。需求 `FR-AUTOTAG-001～003`；关联 `FR-CUR-005/006`。
- 产品负责人：产品用户；实施 Owner：Codex。模型、任务归属现有 AI 服务与主应用；
  标签规范化、人工整理和聚合归属 `internal/curation`，持久化归属 SQLite adapter。
- 自动标签直接关联到图片，不要求用户维护候选词表，不经过建议审批。旧建议接口不自动迁移或审批。
- 单图可手动增删；重新分析必须保留人工标签，删除的 AI 标签不得自动加回，手动恢复除外。
- 标签云按当前库不同图片数量显示大小和颜色，附真实数字，可点击筛选；不得用全局或已加载页计数代替。
- 人物整理、混合搜索来源、OCR 是后续切片，本变更不宣称实现。

## 分阶段实施与 Gate

| 阶段 | 产物与验收 | 当前状态 |
| --- | --- | --- |
| T1 库内真实统计 | 向后兼容扩展 GET /tags 的 libraryId/kind；库隔离、图片去重、游标绑定与异常测试 | 本轮通过：4 包/架构与最终 Docker 真实 API 烟测 |
| T2 标签云生产接入 | 复用正式组件，按库图片计数、有限分页、移动抽屉、手工增删后刷新、稳定 Storybook | 本轮通过：专项 44 项、全前端 345 项、四断点及真实 Docker 界面；6010 已更新 |
| T3 自动标签数据保护 | 来源、删除抑制、人工恢复、原子合并、并发与源文件代次合同、追加 migration | 已实现并通过服务/SQLite/真实 Docker 回归；migration 41/42 |
| T4 模型与有界任务 | 固定模型与身份、有界资源、按库开关、失败/取消/重启恢复 | 本地 ARM64 工程验收通过：同一 2304 MiB 预算，18 次真实模型请求无 OOM；生产自动写入和故障恢复通过。普通照片质量仍待验收 |
| T5 本地 Docker 联验 | 合成图片验证真实 API、持久化、任务及四断点界面；原始媒体只读 | 最终固定镜像通过：加严故障 smoke、四断点界面、真实人工编辑/重跑/重启持久化，见下方；不等于全产品发行验收 |

T1/T2 的历史结果不能代替 T3/T4 验收。当前 T3/T4 已依据 ADR-0021 接入独立 JoyTag 标签身份、
真实模型输出和 curation 原子合并；没有自动批准旧建议。以下早期共驻 OOM 记录保留为失败证据，
已由请求级子进程方案解决本地同预算可行性，不再是当前实现阻塞。

## T3/T4 当前交付：直接打标、人工保护与有界内存

- `internal/autotag` 拥有按库开关、覆盖率和持久化任务；`internal/curation` 唯一拥有标签来源、
  人工优先及删除抑制。模型结果、原子合并和 checkpoint 在同一 SQLite fenced transaction 提交。
- 每张图显示 `AI` / `手动` 来源；既有标签按人工数据迁移。手动添加/锁定会转为人工，重新识别
  不删除人工标签；手动移除按图片与规范化名称持久抑制，显式添加才解除。标签 ID 重建、模型换代
  或重跑不能绕过抑制。全局改名/删除、20 标签上限、CAS 冲突和源指纹变化均有回归。
- 自动标签独立启用，图片/动图参与，视频排除；不要求候选词库或审批。JoyTag 使用固定身份、
  92 个允许类别、阈值 0.4、最多 20 项，类别为英文模型名称，不随界面语言翻译。此政策不是质量校准结论。
- 每库一个活动任务，全局有界；重启以 lease/checkpoint 恢复。失败与用户取消后停止自动重试，
  明确重新识别可以恢复。`503 busy` 不当作永久失败：短重试后原任务延迟 30 秒再排队，不创建任务风暴、
  不消耗 crash recovery 次数、不改变已经提交的标签。
- AI 服务保持两个部署单元和原资源限额；只有一个运行子进程及一个等待名额。按请求核验/复制当前
  模型子集，26 秒总时限包括排队、复制、加载和推理；结束或断连后回收子进程及快照再释放名额。
  真实 18 请求的 cgroup 峰值 **2,185,469,952 bytes（2084.23 MiB）/ 2304 MiB**，OOM 计数为 0；
  包含搜索、人脸、标签交替、1600px 最大输入、排队及第三请求 busy。
  [固定镜像测量与实际 TCP 断连回归](../../spikes/automatic-tags/SCHEDULED-RESULT.md)。
- UI 复用正式查询、设置/单图编辑组件及稳定 Storybook。生产不再通过普通浏览侧栏推送“标签建议”；
  旧审核深链/API 保留兼容，没有将历史建议伪装成自动输出。人物、混合搜索/OCR 原型仍是后续切片。

功能联验的实际测试与镜像证据在本文件下方；随后用户授权的原 6010 升级另记于末尾。

## T1 已冻结合同

- `GET /api/v1/tags` 保留不带过滤条件时的全局词表兼容行为（包括零关联标签）。
- 可选 `libraryId` 与 `kind` 和标签资产列表使用同样的 ID/媒体种类规范。
- 指定库或种类时只返回有匹配资产的标签；`assetCount` 只统计过滤范围内的 distinct asset ID。
  图片云请求当前库及 `image,animated`，不把视频加入图片数量。
- 搜索、库、规范化种类和 curation revision 均绑定游标。错误/未知库明确返回错误，不回退全局。
- 每页上限沿用 200，按标签名称与 ID 稳定 keyset 排序；不无界拉全量、不修改原始媒体。
- 无新部署单元、模型下载或数据库结构变化；OpenAPI 是可执行权威合同。

## 验证记录

T1 实际执行：

- `GOTOOLCHAIN=go1.26.5 go test -count=1 ./internal/curation ./internal/api ./internal/store/sqlite ./tests/contract` 通过。
- `GOTOOLCHAIN=go1.26.5 go test -count=1 ./tests/architecture` 通过。
- 8 个新增顶层回归覆盖双库/空库、静态图/动图/视频、零关联兼容、离线保留、未知库、
  种类归一化、跨库/搜索/revision 游标失效及旧无过滤 cursor 兼容。
- `make generate` 在 Node 22.22.2 下执行成功，生成 TypeScript 合同；OpenAPI SHA256 同步。

T1 本地 Docker：以本轮首个实际镜像 `sha256:13fee5dea5a6f1779a3e1f99787105266927dbe12d183af98681db4e9680974e`
执行 `FOLIOPATH_TEST_APP_IMAGE=foliopath-app:tag-cloud-20261022 node tests/release/tag_cloud_smoke.mjs` 通过。
脚本只使用复制的合成 PNG/MP4、临时数据目录与独立容器，经真实鉴权/CSRF/库创建/扫描 API 验证
双库隔离、视频排除、全局词表兼容、分页/非法作用域、手动删除计数、重启持久化和媒体 SHA256 不变。
临时容器已移除，合成数据保留以便复查；未接触真实照片库。这是 T1 镜像，不是 T2 UI 最终候选。

模型隔离实验：[可重复脚本及测量](../../spikes/automatic-tags/README.md)。固定 revision/hash 的 JoyTag
在 Linux ARM64 本地 Docker 完成 11 张合成图、34 次实际推理；单图 p50 827 ms、p95 859 ms，
进程峰值 RSS 647.25 MiB、容器约 940.6 MiB，5 项单测通过。仅证明单模型运行，非普通照片质量、
现有搜索/人脸会话共驻或生产自动标签验收；未改活动模型缓存/服务，未使用 server222。

随后执行[同预算共驻实验](../../spikes/automatic-tags/CO-RESIDENCY-RESULT.md)：保持 Compose 的
2304 MiB RAM / 1200 MiB 私有快照限制，原搜索双编码器、人脸检测和识别四会话均加载成功；
加载第五个 JoyTag 会话时 Docker 确认 `OOMKilled=true` / exit 137，未执行 JoyTag 推理。
快照合计 1,168,346,001 bytes 可容纳，瓶颈是总驻留内存，而非模型目录容量。launcher 单测 3/3
通过；实验容器与临时快照已清理，未提高资源限制，未重启或修改原 AI 服务。

当时的下一步是设计并实测共享的按需加载/卸载或互斥模型准入策略（不能假定删除 Python 对象即可回收
ORT 内存），或提出有证据的部署资源方案。无论哪条路径，均不得把单模型成功记为当前 Compose
兼容；身份协议、人工保护、普通照片质量和重复加载稳定性仍须分别验收。

## T2 最终验证与本地交付

- 后端上述 4 包及 `tests/architecture` 在最终改动后以 `-count=1` 重跑通过；
  `make generate-check arch-check` 通过。前端最终 `npm test` **60 文件 345 项通过**；
  本切片 adapter/query/cloud/page/editor/shared 专项 **7 文件 44 项通过**。
- Node 22.22.2 下 `check:types`、`check:architecture`、`check:visual-references`、
  `build:storybook` 通过；Storybook 有既有的大 chunk 警告。`git diff --check` 通过。
- `npm run lint:openapi` 通过，保留两个 health 端点缺 4XX 声明的既有 warning；不等于完整 `make lint`。
- 最终 Docker 镜像：`sha256:d23d3190625ec0b883df155326a1ffd2299d09568859c385888746a53573781e`。
  `tests/release/tag_cloud_smoke.mjs` 在该镜像重跑全部通过，独立容器使用两库、三张合成 PNG 与一段
  合成 MP4；人工修改、重启持久化与原文件 SHA256 不变均经真实 API 验证。
- 浏览器在同一合成 Docker 实例验证：选蓝色显示真实标签标题和一张图；单图新建“界面测试”立即
  出现在云中，移除后消失；切至 beta 只剩该库蓝色标签，搜索 alpha 独有“样片”为空。
  390px 抽屉打开、选标签收起且焦点回菜单通过；1920/1440/768/390 下 `scrollWidth` 均等于
  `clientWidth`（1905/1425/753/375），无横向溢出；控制台 error 为空。
- 生产 Storybook `Features/Curation/Photo tags` 使用正式组件及显式合成 fixture；五档字号按固定
  1–2 / 3–9 / 10–49 / 50–199 / 200+ 计数，不随当前页最大值漂移。多档字体/颜色在浏览器检查。
- 复查修复了云区域无水平留白、分页内存栈与浏览器返回错位、手机选标签后抽屉不关闭和主区无
  当前筛选上下文。最终未知前驱使用“返回第一页”，失效游标明确重置，未知库不回落到其他库。
- 6010 仅替换主应用，更新前后校验环境、端口、挂载、安全/资源配置一致，health 为 healthy。
  AI 容器 ID `3e552b18cd9cc35f73aef756dc7327a72d8a081d8bdc9af7a516b2cfdab82a47`、
  StartedAt `2026-10-12T16:32:37.648825333Z` 均未变。没有部署 server222。
  真实 `coser` 库无已关联标签，页面如实显示空状态；未为截图修改真实标签或启动识别。

证据保留在 `test-results/tag-cloud-2026-10-22/`，设计复核见根目录 `design-qa.md`。
测试结束后按已核验容器 ID/归属标签移除了独立 `fp-tags-66eac0f4` 容器；其合成媒体与测试数据仍在
临时目录中，未删除真实数据，6010 主应用及 AI 服务继续运行。
本轮没有执行完整 `make test`、`make lint`、`make test-integration`、`make test-e2e` 或全浏览器
主题/语言矩阵；专项通过不能替代这些全面验收，也不关闭自动标签、模型质量或发行门禁。

### T1/T2 当时的差异与下一步（历史，T3/T4 已由上方取代）

- 仅把标签云和既有人工编辑联动迁移到生产；原型的 AI 来源、自动重跑、人工删除抑制仍等待 T3/T4，
  不显示假开关或成功数据。旧“标签建议”链路保留兼容，不代表接受该旧产品方向。
- 标签名来自当前结果的真实 `curation.tags`，与云分页解耦；零结果且无元数据时显示“已按标签筛选”
  和清除按钮，不臆造名称或无界拉全局词表。以后单标签元数据合同可以补齐这一轻微展示限制。
- `/tags` 未选择标签时保留选择引导，不冒充全量照片查询。照片集合仍走原有分页与虚拟化。
- 下一切片为 T3 来源/人工保护合同与迁移，以及 T4 同预算模型调度可行性；不能把本轮
  **T1/T2 两项完成**解释为“自动给每张图打标签”已经完成。

## T3/T4/T5 验证记录（2026-10-22，替代上方 T1/T2 的待办结论）

实际执行，不将局部通过等同于完整发行：

- `make fmt generate-check arch-check` 通过；OpenAPI 已 regenerate，SHA256 同步。
- 使用 `GOTOOLCHAIN=go1.26.5 CGO_ENABLED=0`，独立扩展盘 `mktemp` 临时目录下执行
  `go test -count=1 ./cmd/... ./internal/... ./tests/integration/... ./tests/contract/... ./tests/architecture/...`
  全部通过。相同包范围 `go vet` 通过。
- 人工保护/自动任务、busy 延迟、模型换代、并发 fencing 与数据库包的专项 race 回归通过。
- 最终前端 `npm test` **63 文件、387 项通过**；类型检查、前端架构检查、视觉引用清单校验与
  Storybook 构建通过。视觉引用校验仅检查清单，不代表完整像素回归；保留既有 chunk-size 警告。
- `make test-libvips` 在本地 Docker Linux native 测试通过，覆盖真实图片解码及 1600px/输入边界；
  `make test-e2e` 的 runtime smoke 通过，包含 native 镜像、迁移、鉴权、health、重启和只读哨兵。
  它不是 Playwright 三浏览器完整业务矩阵。
- AI 最终镜像的 Python 单测 **28/28** 通过；真实 TCP 断连在 1 秒断言内回收子进程，下一请求成功。
- `make lint` / `make test` 不能记为通过：宿主缺少 pkg-config/libvips；CGO=0 仍被既有 facedecode
  spike 的 `vips.ShutdownThread` 构建阻塞。初始系统临时目录还触发了低空间保护；后续生产/集成
  包改用扩展盘临时目录通过，没有关闭或降低空间安全门槛。

真实双服务烟测脚本：`tests/release/automatic_tags_smoke.mjs`。命令：

```sh
FOLIOPATH_TEST_APP_IMAGE=foliopath-app:automatic-tags-candidate \
FOLIOPATH_TEST_AI_IMAGE=foliopath-inference:automatic-tags-candidate \
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-experiments/automatic-tags-integration-cache \
node tests/release/automatic_tags_smoke.mjs
```

脚本创建独立 Compose 项目、干净数据库及 6 张合成 PNG / 1 段合成 MP4，随机 loopback 端口；
只读挂载原文件，AI 无媒体/数据库挂载或公开端口，保持 2304 MiB RAM/swap 上限。
通过的断言包括真实模型非空标签、视频排除/库隔离、手工优先、移除后重跑不补回、显式恢复变人工、
幂等/CAS/CSRF、**推理进程存在时**重启主应用并在新进程启动后完成 5/5、运行中取消后 35 秒无任务
自动重试且两库全部资产的实际标签不变、真实停 AI 后失败并保留标签、恢复服务后不自动死循环而显式
重新识别恢复、关闭并重启后状态/标签持久化、全部源文件 SHA256 不变、两个容器均无 OOM。

已通过的加严轮次 `fp-auto-tags-af4b5818` 使用主镜像
`sha256:1d7f8311b0074b8b6b729725601d30c3bff7454ed96273859b00f70be91d7bdb`，
AI `sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。
该轮容器/网络按归属清理，合成数据保留。随后修复单图标签控件消失后的键盘焦点和普通标签云
缓存刷新错误，重新构建并再次从干净库跑完整烟测，不沿用中间镜像作为最终验收。

### 最终固定镜像与本地交付

- 主应用：`sha256:c4dae9d22bf1ac2efcb63308344a2d752eb550e35236ca6f3fe111aca42772ea`。
- AI：`sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。
- 最后一轮项目 `fp-auto-tags-83f1e5ee`：脚本 exit 0，以上所有断言通过。测试容器和网络按精确
  项目归属清理；临时合成数据保留在
  `/private/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/fp-auto-tags-83f1e5ee-rO1eJF`。
  其临时 55035 端口已关闭，不作为交付链接。
- 留作界面复查的项目 `fp-auto-tags-f3dc3ad1` 已替换成相同最终主镜像并固定 loopback 55026；
  [打开本地合成标签测试](http://localhost:55026/tags/tag_5?libraryId=lib_1&kind=image)。
  主应用与 AI healthy，无 OOM；合成媒体只读，不与 6010 共用数据库或媒体挂载。
- 浏览器实操：alpha 开关保存/刷新，beta 保持关闭；单图新增/删除手动标签、AI 转人工、真实
  “重新识别”确认提交并回到 5/5，容器重启及重新加载后仍显示人工来源。不是只看标签未消失的旧缓存。
- 最终设置和单图各 390/768/1440/1920 无横向溢出，英文窄屏补查通过；新增/移除/转人工的键盘
  焦点为稳定标题。最终新 tab 在编辑/重跑/重启/刷新后控制台 error 为 0。截图及完整 UI 范围见
  [design-qa](../../design-qa.md) 和 `docs/evidence/ui-audits/automatic-tags-2026-10-22/`。
- 测试应用仅配置搜索与标签的应用端部署身份，人物区不可载入未作伪装；人脸模型资源循环来自
  独立 sidecar 实测。此例不作为人物应用端验收。6010/真实图库/server222 均没有被这轮替换或写入。

### 复查中真实发现并修复的问题

1. 全模型快照常驻仍能压到内存硬限制；最终改为请求能力子集的核验快照，不只减少 ORT 会话。
2. 旧 HTTP middleware 吞掉真实 TCP 断连信号；mock 通过但 native 子进程不回收。改纯 ASGI，
   以真实 uvicorn/socket 关闭测试替代仅靠 mock 的结论。
3. busy 曾被当作永久失败；改为同任务带时间门槛的持久延迟，保留提交数据并释放后台许可。
4. 复核修正 degraded 时无效补齐按钮、切库残留确认框、CAS 失败后旧数据、20 标签上限时无法
   将 AI 锁为人工、切图残留输入和长标签挤压操作按钮；均有专项测试。
5. 实际浏览器发现移除/转人工后焦点掉到 BODY；改为操作发出前同步聚焦稳定标题，异步完成不
   抢用户的新焦点，并测试切图后旧回调不抢焦点。
6. 第一轮加严 smoke 因 `docker top -eo args` 缺少 PID 列中断；修为 `pid,args`，从新项目全量重跑。
   这是脚本错误，不写成产品成功。后续加严重启时间与取消后的实际标签比较，已通过上述轮次。
7. 浏览器日志发现 catalog 刷新将普通标签云当作 infinite cache，访问 `pages.length` 抛错。
   修复共享 owner 的形状检查，保留普通查询并失效刷新；新增混合缓存及真实异步刷新回归。
   最终新浏览器 tab 完成编辑、重跑、重启和刷新后 error 为 0，旧失败不抹除。

### 仍然没有验收或部署的内容

普通照片标注准确率/阈值校准、长时间与大容量负载、AMD64、全浏览器/主题/语言回归和完整发行检查
均没有由这次合成数据测试关闭。92 类英文名称及固定阈值是当前真实政策，不承诺任意图片/中文标签质量。
上述独立功能联验阶段未替换原 6010；之后的原环境更新见下节。未部署 server222，未对原库启动自动标签，未修改原始媒体。

## 原 6010 更新记录（2026-10-22）

以下为自动标签版本首次部署的历史配对记录；同日后续任务页改版主镜像更新见
[任务页部署与验收](FIX-2026-10-22-ai-task-workspace.md)，AI 镜像保持不变。

用户明确要求将原 6010 更新到已验证的版本以查看效果。沿用项目 `foliopath-ai-original-review`，
使用独立 `compose.ai.yaml`（不再叠加单服务 Compose）和 Git 忽略的私有 `ai.env`。

- 主应用：`sha256:c4dae9d22bf1ac2efcb63308344a2d752eb550e35236ca6f3fe111aca42772ea`。
- AI：`sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`。
- 地址仍为 `http://127.0.0.1:6010`；媒体与数据库使用原挂载，媒体只读；AI 仅挂载只读模型缓存，
  无媒体/数据库挂载或公开端口，资源限额保持 1024/2304 MiB。
- 接入已验证的 CLIP/buffalo_l/JoyTag 缓存；从固定 AI 镜像离线读取兼容身份，保留原有模型与人物阈值配置。
  runtime 身份更新后搜索索引使用新代次，旧完成数不冒充新代次完成数。
- 原数据副本和旧部署配置保存在本机扩展盘 `backup-6010-autotags-20261022-3tWgew`。
  备份主数据库复制时间 19:52:41，早于新应用首次启动 19:53:01；副本 `PRAGMA integrity_check` 为 `ok`。
- 更新后实际核对：4 个库、8,719 项媒体、0 个已有标签关联、0 个自动标签任务；与备份计数相同。
  当前库自动标签保持关闭，未在原库发起打标；不会填入合成测试结果。
- 双服务 health 为 healthy，`GET /health/ready` 返回 ready；活动库 `PRAGMA quick_check` 为 `ok`。
  浏览器原会话正常，正式设置页显示 JoyTag、按库开关及人工保护说明。
- 记录一次启动期间 `media-worker` 退出，Docker 自动重启后恢复；最终主应用 restart=1、AI restart=0，
  两者 OOMKilled=false，后续连续检查均 healthy。本次部署不把该记录删除或写成零重启。

本次为已验证候选版本的本地原环境更新，不关闭标注质量、大容量、AMD64 和发行门禁。
页面证据：`docs/evidence/ui-audits/automatic-tags-2026-10-22/6010-updated-settings.png`。
