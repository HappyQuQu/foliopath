# FIX-2026-09-23：分析任务页视觉层级

- 范围：POST-MVP-5 / CR-2026-030，D01/D04 既有正式界面修复。
- 目标：让需求、确认原型、生产代码和测试保持一致；不改变 operation API 或任务状态机。
- 所有者：Web / AI settings。

## 问题

旧页面把“返回智能分析”和“任务”标题悬置在内容区右侧，任务为空时使用近乎整页宽的大横幅；有任务
时每个 operation 又成为独立大卡片。宽屏信息松散，任务之间难以比较，窄屏也缺少稳定的信息顺序。

## 决定与实现

采用用户确认的方案 1：页头内显示返回路径与“分析任务”标题，正文先显示进行中、已完成、失败三项
真实计数，再用一个分隔列表展示最近任务。列表字段来自现有服务端 operation：名称、更新时间、状态、
阶段、进度、处理数量、错误与取消。空状态缩为普通内容块并返回智能功能。

确认原型中的媒体库名称与失败重试没有进入生产实现：当前公开 operation 不返回 library ID/name，
也没有足够参数可靠重放失败任务。伪造值或猜测重试会违反真实状态和单一状态机所有权规则。

## 验证

- `npm --prefix web run check:types`
- `NODE_OPTIONS=--localstorage-file=/tmp/foliopath-ai-tasks-localstorage.json npm --prefix web test -- --run src/features/ai/pages/AISettingsPage.test.tsx`
- `npm --prefix web run check:architecture`
- `make arch-check`
- 干净 Node 22.22.0 容器中的全前端回归：51 个文件、236 项测试通过。

最终本地镜像为 `sha256:dadb472c28d7062766e49c8609de531c9550cc2a03efd47d2ce96580f837e210`
（122,506,025 bytes）。应用与 AI 容器均 healthy，应用重启计数 0。浏览器在 1145px 桌面与
390px 窄屏完成真实任务状态复验，均无横向溢出，console error 为 0；浅色、深色和对比截图位于
`docs/evidence/ui-audits/ai-tasks-2026-09-23/`。设计复核见仓库根目录 `design-qa.md`，最终结果为 passed。
