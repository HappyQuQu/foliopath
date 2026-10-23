# 计划升级时保留分析任务与恢复机会

- 范围：FR-INT-016、FR-AUTOTAG-001/002、NFR-OPS-001 / POST-MVP-5；Owner app/jobs/各分析能力/SQLite。
- 决策：[ADR-0027](../adr/0027-planned-analysis-job-handoff.md)；用户已要求修复并继续保留原数据更新 6010。
- 当前状态：实现、Go/定向 race、真实合成 Docker 联验通过；16:47 停写备份后已更新原 6010，原任务按同 ID/断点续跑。153 类识别扩充仍未在原库启用。

## 进度与验收边界

| 项目 | 状态 |
| --- | --- |
| 计划停机与异常分离，处理器/heartbeat 回收后交接 | 已实现并本地 Docker 验证 |
| 同任务/断点、审计、取消优先、旧 fence、原崩溃上限 | Go 专项通过 |
| 旧二进制首次精确离线交接 | 合成验证通过，原 6010 两个精确 claim 已交接 |
| 真实 SIGTERM/SIGKILL、重复升级、源 hash、完整性 | 通过；3 次正常重启续跑，最终强杀仍按旧上限失败 |
| 停写全备份、6010 更新/页面/任务续跑 | 已完成；双服务 healthy、原 ID/断点继续增长 |
| 原库 153 类策略、长任务终态、质量/容量/发行 | 不由本次维护关闭 |

## 根因与修复

正常退出原先只留下 running 租约，claim 已计费；第三次运行的任务下一次启动被 crash recovery 终结。
此外 AI worker component 原来脱离启动上下文，且只等待停止不传递 worker 错误，无法辨识停机原因或报告交接失败。
app 现在只在 SIGINT/SIGTERM 设置计划原因，AI component 在有序 stop 时传递；组件异常退出不冒充计划升级。
jobs 等处理器返回及 heartbeat 退出后，最多 5 秒调用能力队列的 Handoff，失败传播到生命周期日志/退出状态。

canonical SQLite 事务校验 operation/job/claim 的 running fence，先写最多 10 条/operation 的审计，
再交回原队列并撤销本次 claim 消耗的一次机会；历史 crash 消耗保留，checkpoint/completed/身份不改。
重复、过时、取消或终态交接不复活任务。旧二进制首升必须先验证 Docker owner 停止并完整备份，
再在 Linux 离线容器对精确 claim/attempt/checkpoint/completed 清单调用同一事务；任一不符则整批回滚。
维护工具的 Open 可能升级 schema 44，失败时不能假定 schema 也回滚；回退使用完整停写备份。

## 已执行与未执行

首轮 `CGO_ENABLED=0 GOTOOLCHAIN=go1.26.5 go test -count=1`：jobs、autotag、semantic、face、app、
store/sqlite、tests/contract、tests/architecture 全部通过；后加架构检查及 AI component 原因/失败传播测试亦已定向通过。
前端源码未改；候选构建中的 TypeScript/Vite 通过，存在既有大 chunk 提示。
早轮编译暴露 databaseStore 缺少两项新适配器方法，已补齐后重新通过，不把失败轮算通过。

- `go test -race`：jobs/autotag/sqlite 的 handoff/worker/processor 定向项，以及 app 的停机原因/失败传播项通过。
- `go vet`：jobs/autotag/semantic/face/app/sqlite/离线维护工具通过。
- `make fmt-check arch-check generate-check` 实际通过，收尾再次执行通过；`git diff --check` 通过。
- 集成首轮在内部盘临时目录触发既有 `insufficient_space` 保护，未通过；改用扩展盘独立 TMPDIR 后
  `CGO_ENABLED=0 go test -count=1 ./tests/integration` 通过。不是删除断言或关闭磁盘保护。
- 最终合成项目 `fp-handoff-03cbab36`，80 张程序合成 PNG、固定真实 JoyTag 权重：
  旧主镜像 `1806724d` 两次真实 SIGKILL 恢复至 attempt=3 → 完整停写备份/精确交接 →
  新主镜像 `e10d3bef` 三次真实 SIGTERM/启动均保留 operation `autotag_6dd45b1d152e02a58857f8f0aa7afe7e`、
  断点从已提交结果继续（末次记录 checkpoint=5、completed=5）→ 最后强杀仍终结 `operation_interrupted`。
  交接前 attempt 审计均为 3；人工中文标签和收藏（非空 fixture）、源 SHA256、容器内 database-check 通过。
  为加速只在停止的合成库、Linux 容器内推进租约到期时间，不直接写 attempt，不对原库注入。
- 测试脚本首轮漏了建库幂等键，已纠正重跑。第二轮误从宿主读取停止后的原挂载 fixture，出现
  `database disk image is malformed`，同一 fixture 的 Linux database-check 实际通过；读写/断言已全部
  改到容器 Linux 锁域后全轮重跑通过。旧失败轮不计通过，原 6010 未参与故障注入或跨域 SQLite 读取。
- 未执行完整 `make lint/test/test-e2e`、原生 AMD64、长期/完整资源预算和发行矩阵；前端未改源码，
  本轮构建的 types/Vite 与原 6010 页面验证不冒称完整前端套件重跑。

## 原 6010 部署与保留证据（16:47–16:51 +08:00）

1. 原主应用正常停止，ExitCode=0、OOMKilled=false；AI 未停止/替换。
2. 停写完整备份（data 约 1.1 GiB，包含数据库及全部缓存/整理），另存 ai.env 和三个 Compose 文件：
   `/Volumes/extended/foliopath/backup-6010-localized-handoff-20261023-DTt5e9`。
   备份在 Linux 维护容器执行原生 database-check 通过，不从宿主打开原库。
3. 停写新快照精确交接两个 running claim，同一事务写入 schema 44 审计：

   | 能力 / 库 | operation | 交接前 attempt / checkpoint / completed | 更新后 16:51 |
   | --- | --- | --- | --- |
   | 自动标签 / coser | `autotag_0e6072a5243a9a0c7ed8947e2fce74ed` | 3 / 2571 / 1672 | 同 ID running；3 / 2603 / 1704 |
   | 语义 / pixiv | `aio_479074a1d4d99ad663dd86f2087b1d8d` | 1 / 8184 / 464 | 同 ID running；1 / 8216 / 496 |

   此处 completed 是本任务已处理数，不是全库成功数。4k 语义仍按原 ID 排队 368/707，coser 语义 0/6202；
   8k 语义原 ID 为 succeeded，已处理 792/792，不等于全部图片成功或人工质量通过。
4. 仅把 ai.env 的 `FOLIOPATH_IMAGE` 改为
   `sha256:e10d3bef357b6902e13bfd3551b645d3ac79cb395f7998f15bb12b136491c004`，
   通过 base + unlimited-memory + multilingual 覆盖，`up -d --no-deps foliopath` 单次替换主应用。
   其他配置去除该行后的 SHA256 与备份相同；AI 保持 `3631b432`，旧 92 类策略与 tag identity 不变。
5. 两服务 healthy、自动 RestartCount=0、RAM/swap cap=0；原 `/library` 只读挂载、data 路径不变。
   新主应用容器内 database-check 通过。4 库 / 8,719 资产的 ID/源指纹、跨代次 10,623 行人物结果及
   搜索/人物 generation 摘要前后一致；原库人工关联/收藏/人物整理/抑制均为 0，未伪装为非空保护试验。
   非空人工保护证据来自前述合成库，未往真实库添加测试标签。
6. 原浏览器会话仍有效，任务历史保留；正式标签云中文“汽车”返回 `tag_33`（5 张图片），切为 English
   显示 `Car`、双语显示 `汽车 / Car`，仍同 ID。测试后恢复跟随界面语言并清除筛选。
   截图：`docs/evidence/ui-audits/localized-tags-2026-10-23/6010-bilingual-search.png`。

没有新提交/清理原库任务，没有更换搜索/人物/tag 身份，也未连接 server222。
**中文/双语 UI 已上线，但原 6010 的 153 类识别扩充尚未切换；原有源错误与全库补齐仍未清零。**

## 16:59 收尾复核

原 coser 标签 operation 继续到 completed=1784、checkpoint=2683，pixiv 语义继续到 560/999、checkpoint=8280；
两者同 ID、任务错误均为空，attempt 仍为原 3/1，没有额外恢复消耗或清零。其余排队/8k 终态与上表一致。
两服务仍 healthy、RestartCount=0、RAM/swap cap=0，容器内 database-check 再次通过。
原库数据/人物/generation/整理摘要再次与停写快照一致；临时诊断可执行文件已从 `/app/data` 移除。
三个本轮合成项目容器/网络及其可重建 scratch volume 已核验归属后清理，合成数据/备份/镜像仍保留，
没有清理原 6010 或其他项目。未提交/推送 Git。
