# ADR-0027：计划停机交回分析任务

- 状态：Accepted（2026-10-23，用户要求修复安全升级阻碍）；Owner jobs / 各分析能力 / SQLite / app。
- 范围：POST-MVP-5，FR-INT-016、FR-AUTOTAG-001/002、NFR-OPS-001；不增加部署单元或浏览器 API。

## 背景

现有自动标签、图片语义、人物任务每次 claim 消耗一次恢复机会，上限为 3。
正常 SIGTERM 只留下 running 租约，第三次运行时升级会被后续 crash recovery 判为失败。
不能以清零计数、替换任务 ID 或切换模型身份来隐藏问题。

## 决定

1. app 只把正常 SIGINT/SIGTERM 标为 `jobs.ErrPlannedShutdown`；组件异常退出和普通取消不携带此原因。
2. 支持交接的 WorkerPool 等 Process 返回且 heartbeat 退出后，才用独立最多 5 秒上下文交回该 claim。
   仍在推理的处理器、强杀、OOM 不会得到交接资格；持久化失败必须上报，不伪装成功。
3. SQLite 在单个有界事务中校验 job/operation/claim revision 和 running 状态，保存交接审计，
   交回同一任务与 operation 为 queued，撤销本次 claim 消耗的一次机会（不是清零），推进 fence。
   保留 checkpoint、completed_items、结果、错误样本、模型/代次和人工整理。下一次 claim 正常加一。
   已取消、终态或被新 claim 替代时不重新排队；接受的取消优先。
4. 原 3 次异常恢复上限不变。审计记录交接前 attempt/checkpoint/completed，重复/过时交接无效；
   旧 claim 不能再提交结果或再次撤销机会。
5. 首次升级旧二进制时，不假定它已支持新停机逻辑。仅在 Docker owner 已确认停止、完整停写备份后，
   在 Linux 离线维护容器使用同一 canonical store 方法交接清单中的精确旧 claim。
   清单必须来自停写快照，且核对 operation、claim、attempt、checkpoint；未匹配即整次事务回滚。
   离线工具不进入生产启动图或浏览器 API；不得对活跃数据从 macOS 打开 SQLite。
6. 正在执行旧标签身份时继续保留旧 AI/部署身份；153 类策略另行切换，不能影响原任务。

## 替代与后果

- 等待所有任务结束：可安全操作，但不能解决今后任意长任务的计划升级。
- 提高/清零重试次数：隐藏崩溃风险，拒绝。
- 将 SIGKILL/过期租约也当计划停机：没有处理器退出证据，拒绝。
- 仅修改 WorkerPool 不做首次离线交接：旧二进制仍可能把第三次任务终结，拒绝。

增加一个迁移和受限审计表；回退到旧二进制必须使用停写备份，不修改已发布迁移。
先以第三次 claim、取消竞争、旧 fence、真实 SIGTERM/SIGKILL 的合成 Docker 回归验证，再更新 6010。
