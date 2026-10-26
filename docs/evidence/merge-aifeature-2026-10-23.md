# aifeature 与 main 合并验证（2026-10-23）

范围：用户明确要求将 aifeature 全部现有工作（含未提交修改）整理、验证后合并到 main。
Owner：仓库集成维护；既有 POST-MVP-5 / FR-INT-009、FR-INT-016 与
FR-BRW-014、FR-BRW-015、NFR-COMP-001 的集成维护，不新增产品能力或改变发布 Gate。

## 输入和冲突处理

- 主线基线：`fe42406ceb53bba886930b5edf96ae0ba1740442`。
- AI 分支原已提交基线：`b63c4d25867f8629242f5248599a17ea355ea914`。
- 全部既有工作先保存为 `2d4b437`（414 个文件），再合并；原历史保留。
- 保留主线无缝瀑布流、隐藏文件名、缩略图大小滑杆及偏好记忆，合并 AI 来源说明、
  标签语言和任务偏好。补充无缝布局可访问描述及不同偏好互不覆盖的回归。
- 保留主线较新的依赖版本，以 Node 24.22.1 / npm 11.19.1 重新生成 lockfile。
- Node 工具链 fitness check 使用同时检查 npm、两个 Dockerfile、CI 与浏览器镜像的完整版本。
- 同号 CR-2026-033 与 ADR-0026 的不同主题通过完整链接消歧，历史编号与证据不改写。
- `spikes/int001-remote-ml/facedecode/main.go` 增加 `libvips` build tag，与同目录
  facememory 诊断一致。可选 native 诊断不再使常规无 libvips 的开发检查强制依赖 govips。
- 本聊天生成的媒体浏览静态设计图未实现为生产界面，不属于此次合并的实现内容。

## 已执行检查

| 检查 | 结果与边界 |
| --- | --- |
| `make fmt` | 通过 |
| `make arch-check` | 通过，由 `make lint` 实际执行 |
| `make generate-check` | SQL 与 OpenAPI 生成一致性通过 |
| `make lint` | 架构、合同和 Go vet 通过 |
| Linux `make test` | 44 个含测试包通过，另 10 个包无测试 |
| Linux `make test-integration` | 通过，复用紧前全量测试的缓存 |
| `make test-e2e` | Linux 容器启动、健康、鉴权边界、重启及只读原件 hash 烟测通过 |
| `npm ci`、`npm run check` | 高危审计、生成、架构、视觉清单、类型、423 项测试、Storybook 构建通过 |
| 前端补充组合回归 | 2 个文件、30 项测试通过 |
| `npm run build` | 通过；保留大 chunk 提示 |
| AI 服务单元/合成合同 | 8 个模块、36 项通过，0 跳过；包含 TCP 断连回收 |
| Chromium 真实后端浏览器回归 | 首轮 48 通过、12 跳过、1 失败；修复旧断言后失败项定向复验 1 通过 |
| 浏览器测试修复后 `npm run check:types` | 通过 |
| `git diff --check` | 通过 |

首次 macOS 全量 Go 检查未通过：未加 build tag 的 native 诊断要求本机缺少的
`pkg-config`；模型下载集成测试因宿主机约 12 GiB / 460 GiB 的可用空间低于生产
10% 保留线而正确返回 `insufficient_space`。前者已修复并复验 lint，后者未降低安全阈值，
改在 Go 1.26.5 linux/arm64、独立 3 GiB `/tmp` tmpfs 中执行全量与集成测试通过。
Linux 镜像没有 `rg`，Makefile 未使用的 GO_FILES 展开有警告，两个测试目标仍实际执行成功。

AI 测试复用本机现有 Python 3.11.15 镜像，在无网络、只读容器中测试当前源码；
没有下载模型或访问真实媒体/数据库。前端审计仍有 2 个 moderate，无适用 high 项。
本次工程验证不替代 AI 模型准确性、容量、完整 native/发行矩阵或新镜像发布验收。

浏览器首轮执行 `FOLIOPATH_BROWSER_BACKEND_PORT=18143 FOLIOPATH_BROWSER_WEB_PORT=4179 make test-web-e2e`。
原 `auth.spec.ts` 仍断言旧 AI capabilities / Advanced maintenance 导航，改为已接受的
Feature settings / Processing jobs / Service and models，保留禁止下载/导入和按库作用域断言。
首次定向复验继续发现 Masonry 模糊选择器同时命中 Seamless masonry，改为精确匹配。
最终通过同一 harness 的临时副本加 `--grep`，在新建隔离后端中完整重跑该条
auth/library/browse/search/settings 纵向链通过（50.4 秒），测试结束媒体 hash 和目录快照比较通过。
没有重新运行已通过的 48 项；12 项跳过中 9 项需专用真实 AI/故事板环境，3 项属于桌面/移动
输入条件排除。跳过不视为通过。所有临时后端和端口已清理，原 6010 未被访问或重启。

本机执行日志位于 `/tmp/foliopath-merge-{fmt,lint,generate,linux-go,runtime-e2e,frontend,inference,web-e2e}.log`，
不作为仓库源文件提交；上述表格记录此次实际运行结论。
