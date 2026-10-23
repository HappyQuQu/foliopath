# FIX-2026-10-23：搜索方式提示与中文索引续验

- 范围：POST-MVP-5 / CR-2026-032、既有 INT-S3 Consumer/UI Ready；FR-INT-002/003、FR-UI-001/002/003/006/010。
- Owner：search feature 拥有当前 URL 模式及提示；GlobalHeader/AppShell 只接收本地化 label/placeholder；MediaCollection 复用既有 contextLabel。
- 状态：搜索说明与手机 Header 修复已通过本地 Docker 回归并更新 6010，原任务续跑；不是全 AI 完成或发行批准。
- 合同：沿用 OpenAPI 的普通 assets 与 semantic/assets、semantic/videos；不增加 API、迁移、模型或持久状态。

## 问题与修复边界

6010 进入画面语义后，Header 的可访问名称仍为“搜索文件名或路径”，空状态仍指导输入路径；
结果没有说明自身检索方式。这会让中文语义使用方式与字面搜索混淆。
当前已接受原型涉及多来源搜索，但本轮只修复已有两个独立检索模式的可解释性，
不声称完成混合排序、逐图多来源归因、文件名与路径的细分命中或 OCR。

| 可见项 | 合同与语义 |
| --- | --- |
| 搜索框、空状态 | feature 按规范化 URL mode 提供中英文提示；共享 Header 不自行解析业务模式 |
| 卡片检索方式 | 当前实际使用的普通或语义 endpoint；不是另算置信度或猜测命中字段 |
| 视频命中帧 | 保留已有 semanticVideoMatches 时间/故事板信息 |
| 语义结果说明 | 画面相似度排序，不是精确关键词；明确 OCR 尚未接入 |
| 模式切换/重新输入 | 复用唯一 URL codec/query key；保留查询、库/目录范围与递归，不额外混发另一个端点 |

## 只读现库核对（2026-10-23 17:50 前后 +08:00）

在主容器 Linux 内以 query_only、mode=ro、15 秒超时和单只读快照查询；临时工具已移除。
未从宿主打开活跃 SQLite，未重启/重复提交/全量重建或清理任务，原件和人工整理未修改。

| 当前 XLM 640 代次 | 成功 embedding | 失败 | 原 operation 状态 |
| --- | ---: | ---: | --- |
| pixiv | 999/999 | 0 | succeeded |
| 8k | 785/792 | 7 | succeeded（处理结束不代表全成功） |
| 4k | 368/707 | 0 | queued，原断点保留 |
| coser | 112/6202 | 0 | running |

8k 七项均是此前人物也记录失败的相同源指纹资产：六张 0 B 和一张非空无效 PNG，
参见[逐图源诊断](FIX-2026-10-23-face-source-diagnosis.md)。这是缺 embedding 与既有诊断的交叉核对，
不是本轮重新解码或为历史语义失败伪造原因。
coser 标签当前旧 92 类：ready 3180、failed 15；本轮 operation 已处理 2311，仍 running，
包含先前任务保留成功，因此不得把本任务 completed 与累计 ready 混用。153 类策略未切换。

正式页面 pixiv 输入“蓝色头发”保留 mode=semantic 和当前库，覆盖 999/999、失败 0，返回第一页 50 项。
这是中文查询链路实测，不是蓝发召回/准确率质量验收。

## 18:21 当前原库续跑快照

同一只读方法复核：pixiv 999/999、失败 0；8k 785/792、失败 7，均为终态；
coser 语义 running，416/6202、失败 0；4k 原任务仍 queued 368/707。
coser 标签累计 ready 3478、failed 15，本任务 processed 2609、checkpoint 3510，仍 running，
原 operation/模型身份不变，没有另建任务。主/AI healthy、自动 RestartCount=0、RAM/swap cap=0；
主容器内 database-check 再次通过。15 个标签失败只有结果状态，无逐图持久原因，不能伪造新诊断。

## 验证记录与修正

- 首次新增组件测试把递归 URL 写成 `recursive=true`；规范合同是 `recursive=1`，修正 fixture 后 21 项通过。
- 随后类型检查发现测试误用 Playwright 的 `exact` 选项；删除后重跑通过。
- 首次 Docker E2E 缺少只读合成图片挂载，未发现测试；补充 fixture 挂载后重跑。
- 第二轮 24 项因未模拟 Header 消息中心的 libraries/diagnostics/releases 请求失败，保留“不允许未模拟请求”
  断言，按合同补齐只读 fixtures 后 24 项通过，没有放宽断言或把真实 API 作为 fallback。
- 截图人工复核又发现共享 MediaCollection 的固定两行页脚把第三行 contextLabel 压缩裁切；
  改为基于既有 token 的最小高度/自然三行，保留虚拟测量并调整有上下文时的估算。
  新增三行各自不重叠/不裁切的真实 DOM 几何断言；最终固定镜像三浏览器 24 项再次通过。

实际通过：

- 本机 Node 24.22.1：最终 `npm run check:types`、`npm test` **67 文件 / 416 项**。
- Docker Node 24：完整 **67 / 416**；合成原型图片以只读挂载提供，不读取真实库。
- `npm run check:architecture`、`generate:check`、`check:visual-references`、`build:storybook`；
  清单检查不等于像素全矩阵；Vite 保留既有 chunk-size 警告。
- `GOTOOLCHAIN=go1.26.5 CGO_ENABLED=0 make fmt-check arch-check generate-check`、`git diff --check`。
- native 候选构建源 `spikes/automatic-tags/Dockerfile.localized-app`，
  `sha256:ae58003f251cf1d9f10b984805a21d8831b5de16e14f0b61db70137915f570cf`；
  只是本地增量构建，不是双架构/供应链发行构建。
- 候选 native 应用独立 `/library` 只读空合成目录、独立 data，1 CPU/1 GiB，
  healthy/restart=0、容器内 database-check 通过；**没有运行第二个 AI 模型服务争抢原任务资源**。
- Docker Playwright 1.61.1：对该 native 镜像的真实生产 bundle 使用合成只读 API，
  Chromium/Firefox/WebKit × 四宽度（390/768/1440/1920）× 中浅/英深 = **24/24**，
  含中文重提、范围/递归、单一请求端点、来源文案、页脚三行、无异常/未模拟请求/横向溢出。
  这些是 UI/路由测试，不冒称真实模型推理或中文准确率校准。
- 稳定 Story `Features/Search/Mode guidance` 复用生产 SearchPage，
  `Patterns/MediaCollection/Source and context` 覆盖共享三行页脚；没有另造生产页面。

完整截图/合成 data 保留于 `/Volumes/extended/foliopath/fp-search-ui-MECPT3/`。
收尾已核对归属并停止/移除本轮独立 UI 测试容器；合成数据、截图、候选镜像保留，可重建。
此前 18:21 的 6010 原 UI 仍是 e10d3bef；当时只读测试中文查询，没有替换主应用或 92 类 AI，
未改 ai.env、凭据、人工整理、原件或任务，不连接 server222。候选待后续集中安全更新，
不为这些界面文案反复重启在途工作。完整 make lint/test/integration/E2E、真实模型烟测、
混合检索/OCR、质量/容量/AMD64/发行没有本轮新增通过，现有 npm 高危审计也未关闭。

## 续作：原 6010 更新与手机可用性（18:34–19:13 +08:00）

### 实际发现与修复

18:38 首次安全更新为 ae58003f 后，在原登录会话实测：中文“蓝色头发”返回第一页 50 项，
切换文件名、查 `saihate` 返回 2 项；输入、空状态、卡片说明对应真实端点，不是 OCR/混合召回。
实际 390px 截图发现旧 Header 虽无横向溢出，但输入框被工具按钮挤至仅能显示很少文字。

- <=480px 的 canonical GlobalHeader 改成“品牌/搜索”与“人物/工具”两行；保持 DOM 与视觉顺序相同。
- 搜索按钮可见；高度统一由 `--size-header: 7rem` 提供，侧栏/固定页头/主内容不复制偏移。
- 人物移动图标增加真实可访问名称；未删除工具或缩小到不可用尺寸。
- 新稳定 `Patterns/AppShell/Full header` 使用正式 AppShell、合成身份/消息按钮，不另造页面。
- 需求、流程、UI、traceability、testing strategy 同步；原语义 API/模型/代次/工作限额均未改。

加严浏览器验证第一轮 **18 通过 / 12 失败**，定位隐藏“人物”文本后丢失可访问名称；
补充 `aria-label` 后保留原断言并重建，最终 **30/30** 通过：三浏览器 × 五宽度
320/390/768/1440/1920 × 中文浅色/英文深色。增加输入可用宽度、标题不被遮挡、
手机搜索按钮可见与“输入 → 提交 → 人物”焦点顺序，原三行页脚与路由断言保留。

最终前端本机 `check:types`、`check:architecture`、`npm test` **67 文件 / 416 项**、
`build:storybook`、`check:visual-references` 通过；根目录 fmt-check/arch-check/generate-check 通过。
最终 native 镜像构建（TypeScript/Vite）和独立 Docker 的 30 项浏览器测试通过。
本次手机补丁没有再次运行 Docker 的完整 416 项单测；上方 Docker 416 是前一候选的证据。
独立合成数据/最终截图保留 `/Volumes/extended/foliopath/fp-search-mobile-kUvCwK/`，
首轮失败产物在其 `results-accessible-name-failed/`，最终在 `results/`。

### 两次停写备份与任务交接

不是一次部署：先部署搜索提示候选，实际手机验收发现遗漏后修补，再部署最终版本。
两次均正常 SIGTERM、ExitCode=0、无 OOM，**只替换主应用，未重启或切换 AI**。

| 主应用更新时间 | 完整停写备份（约 1.1 GiB data，另含 env/三份 compose） | 镜像 |
| --- | --- | --- |
| 18:38:25 | `/Volumes/extended/foliopath/backup-6010-search-guidance-20261023-l64RAi` | ae58003f |
| 19:11:58 | `/Volumes/extended/foliopath/backup-6010-search-mobile-20261023-abkY7L` | 81887939 |

首份停止态备份初次以完全只读挂载检查，因没有 WAL/shm 且不能创建辅助文件报无法打开；
改为仅在脱离原 owner 的备份副本允许创建辅助文件，查询仍只读后 database-check 通过。
第二份备份和每次新主容器的原生 database-check 均通过。没有从宿主打开活跃 SQLite。

最终固定镜像：`sha256:81887939a60990d794a167922c2eef235eafb5de4efbc3c60480a95a8f427239`。
每次仅修改 ai.env 的 FOLIOPATH_IMAGE；排除此行后的配置与备份逐字相同。
三 Compose 覆盖仍为 base + unlimited-memory + multilingual，AI `3631b432`、旧 92 类身份不变。
两服务 healthy、自动 RestartCount=0、OOMKilled=false、RAM/swap cap=0；媒体挂载仍只读。

| 最后一次自动交接 | operation | 停止时 checkpoint / processed / attempt | 19:13 恢复后 |
| --- | --- | --- | --- |
| coser 语义 | `aio_08d77ae32263d0ada16a64c6117051e6` | 908 / 896 / 1 → queued 0 | 同 ID running：924 / 912 / 1 |
| coser 标签 | `autotag_0e6072a5243a9a0c7ed8947e2fce74ed` | 3950 / 3049 / 3 → queued 2 | 同 ID running：3964 / 3063 / 3 |

ADR-0027 正常交接审计与 checkpoint 保留，未人工改 attempts、重复提交或清理任务。
升级前后 4 库 / 8,719 资产 ID 与源指纹、10,623 行跨代次人物结果、语义/人物 generation 摘要相同。
真实库人工标签/收藏/命名人物/抑制均为 0，不能拿摘要相同冒充非空人工保护试验。
比对 JSON 和停写/恢复进度保存在各备份目录。

最终原页面手机输入框实测 197px，Header 112px；1920px 下 Header 68px，页面宽度/scrollWidth
同为 1905（滚动条占宽），无横向溢出。手机键盘顺序实测正确；原会话保留，文件名与中文语义
查询再次通过。最终截图名 `6010-semantic-mobile-fixed-390.png` / `6010-semantic-final-1920.png`，
位于 `docs/evidence/ui-audits/2026-10-23-search-guidance/`；旧 `6010-semantic-390.png` 是修复前证据。

### 本轮发现的未解决项

19:13 当前 XLM：pixiv 999/999 成功，8k 785/792 成功、7 源失败；4k 368/707 仍 queued；
coser 899/6202 成功、13 失败，仍 running。coser 标签累计 3932 ready、15 failed；
本 operation processed=3063，仍 running，153 类未切换。

新增 13 个 coser PNG 缺 embedding，尺寸约 16.77 MP，超出语义 16 MP 上限；其中 3 个还超过
32 MiB。已明确定位策略阻挡，**尚未修复大 PNG 兼容性**，不能当作坏图恢复要求或 OOM。
详见 [大 PNG 输入限制调查及修复顺序](INV-2026-10-23-semantic-png-limits.md)。
容器内合成预算测试通过只证明当前拒绝规则存在，不证明 13 张已成功。
不连接 server222，不改原件/人工整理，不宣称全库补齐、153 类、混合检索/OCR 或发行完成。

收尾复查：最终主/AI 仍 healthy、自动重启 0、数据库检查通过；临时诊断可执行文件已从原 data 移除。
已核验标签/固定镜像后停止并移除本轮 `fp-search-mobile-ui-20261023` 合成 UI 容器，
其独立数据、失败/通过截图、镜像及两份原库备份全部保留，可重建；没有清理其他项目。
最终 `make fmt-check arch-check generate-check` 与 `git diff --check` 再次通过；未提交/推送。
