# 人脸移出后重新归类修复

> 本文保留当时状态、失败及撤回记录；最新结论见 [P2/P3 全项复查](p2-p3-final-recheck-2026-09-16.md)：修复后最终固定镜像重新验收，P2 6/6、P3 9/9、18/34；不代表质量、容量或发布完成。

最终结论：A05/D03 修复复验后重新关闭，P2 6/6、P3 8/9，当前清单 17/34；D04 仍保留。

范围 CR-2026-030 / FR-INT-011、013 / POST-MVP-5 S2/S3，Owner face / SQLite / web AI。
这是 A05/D03 的常规修复，不新增部署/持久表，不放宽原媒体只读或整组质量门禁。

## 合同与实现

- 保留已移出 lineage 的 anchor ID，在 SQLite 现有审核事务内重新归属，避免删掉历史撤销/约束引用。
- 对仍绑定的人脸，扩展现有 AssignFaceReview 的可选 sourcePersonId / expectedSourceRevision，必须成对提供并匹配当前归属；同时校验目标 revision。不带源版本的旧客户端不能抢占已有归属。
- 已解除绑定的 needs_review lineage 可以不带源字段归入新人物；事务内读取和推进旧人物 revision。带 cannot-link 的转移仍拒绝，约束不删除。
- 保存原 owner/state/current face 与双方版本；撤销校验双方当前版本及 anchor 版本，原子恢复旧归属。旧快照缺少 beforePersonId 时回退到既有 payload.personId，保持兼容。
- 审核、双方版本、归属和撤销快照同一事务提交；后段失败全部回滚，幂等重放保留 affectedPersonIds。

## 证据状态

最初 detached 七场景、扩展直接移动后的十场景 Docker race 回归通过：撤销、源/目标版本冲突、cannot-link、未声明源的绑定转移拒绝、事务后段回滚、直接转移、错误源与幂等重放。

最终应用镜像 `foliopath-app:face-move`：`sha256:c68d7db8594bf988b9a400bad57467390fc209cbdcc93ec3d5b347d6a7a522e2`。
回归镜像 `foliopath-remote-ml-regression:face-move`：`sha256:f5777136121d257daabf615a1af12886dfb448468c089472b44865507b5f01ea`。

- 前端 Docker `generate:check`、`check:types`、`check:architecture` 和 52 文件 / 234 单测通过。
- `make fmt-check`、`make arch-check`、`git diff --check` 通过。
- 最终 Docker race 回归通过：`sqlite.test -test.run "Face|People|Person|CannotLink|GuardedUndo"`、`face.test` 全包、`app.test -test.run Face`、`api.test -test.run Face`、`contract.test` 全包；容量开关未启用，不计入容量验收。
- 最终应用镜像运行 `face-business-probe.mjs` 全部通过：真实模型非空推理、逐脸归类/拆分/合并/撤销、cannot-link、重建/两类清理、禁用恢复、代次切换、取消、库离线、源变化/移除及 Chromium 人物流程。
- 三浏览器首次完整在线运行：22 通过、8 条件跳过、1 失败。三浏览器新人物直接移动/移出后重新归类及标签 52 条分页/冲突重试均通过；失败为 WebKit 设置连续矩阵导航触发正式 `/auth/session` 429（`Retry-After: 1`），不是人脸接口失败。测试已改为有界遵守 Retry-After，未关闭/放宽服务端限流，也不重试其他状态；完整重跑待记录。
- 第二次在线 23 通过 / 8 条件跳过；离线切换前 Node 因同步浏览器运行后复用已关闭的 keep-alive socket 而出现 `UND_ERR_SOCKET`。探针改用 `Connection: close`，不盲目重放写请求；本次不记为整个探针通过。测试脚本 TypeScript 检查通过。
- 最终完整重跑 `FOLIOPATH_P3_APP_IMAGE=foliopath-app:face-move node spikes/int001-remote-ml/p3-browser-probe.mjs` 退出码 0：在线 23 通过 / 8 条件跳过，随后真实停 AI 服务的三浏览器离线 3 通过，并恢复服务。8 跳过为在线阶段 3 个离线案例、桌面项目 3 个移动专属案例、移动项目 2 个桌面专属案例；不隐藏失败。设置矩阵覆盖三浏览器 × 两语言 × 两主题 × 四宽度，但人物/标签全矩阵、真实 AI 触控及视觉基线仍欠缺，不关闭 D04。

以上不是完整七项 make 发布检查，也不证明模型质量、100k 容量、原生双架构或 D04 全矩阵。
没有访问 server222、真实媒体或对外发布。
