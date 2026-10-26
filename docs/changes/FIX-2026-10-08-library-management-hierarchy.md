# FIX-2026-10-08：媒体库管理页信息层级与响应式布局

- 范围：MVP `FR-LIB-001～008`、`FR-SCN-003`、`FR-UI-001/009` 的既有正式界面修复。
- 目标：让媒体库页采用已确认的紧凑管理信息层级，并保持需求、稳定 Storybook 状态、生产组件与回归测试一致。
- 所有者：Web / libraries。

## 问题

旧媒体库页把每个库渲染成独立卡片，只在一条副文案中混合路径和媒体数。宽屏缺少可比较的信息列，
扫描时间、目录数和状态的层级不清；窄屏依赖卡片动作换行，难以快速判断哪些库需要处理。

## 决定与实现

沿用智能分析任务页已确认的视觉语言，但不复制其业务字段：正文先展示媒体库、媒体与健康状况三项
真实摘要，再使用一个分隔列表承载所有媒体库。每行展示安全容器路径、媒体/目录数量、最近成功扫描、
库状态和自动发现状态；浏览、状态展开、改名和安全移除继续调用既有合同。

媒体库列表是 keyset 分页。尚有下一页时，摘要明确标记为“已载入”，避免把客户端当前页聚合误报为
实例全量。该变更不新增 API、持久化字段或状态迁移，也不改变原始媒体只读边界。

生产组件的稳定 Storybook 状态使用明确标注的合成合同 fixture，作为设计与实现一致性证据。响应式
规则覆盖窄屏纵向布局、中等宽度两行布局、普通桌面和至少 1920px 宽屏的流式工作区。

## 验证

- `npm run check:types` 通过。
- `npm test -- --run src/features/libraries/pages/LibrariesPage.test.tsx`：6/6 通过。
- `npm test -- --run`：51 个文件、237 项测试通过。
- `npm run check:architecture`、`make arch-check`、`make generate-check` 通过。
- `make fmt-check`、`npm run check:visual-references` 通过。
- `npm run build:storybook` 通过，稳定 `Features/Libraries/Management` 状态已生成。
- 最终本地镜像 `sha256:44d7a6749f1f3700f898053a2b3cf33142ae09f09c07091f438aea4394c2c023`
  （122,574,735 bytes）；应用和 AI 容器均 healthy、重启计数 0。
- 390、768、1440、1920 四档、浅色/深色、手机行内状态展开均无横向溢出，浏览器 console error 为 0。
  截图位于 `docs/evidence/ui-audits/library-management-2026-10-08/`，最终设计复核见根目录
  `design-qa.md`，结果为 passed。

本地旧测试库在重启时暴露出既有零头损坏；损坏文件已保留为
`foliopath-data/foliopath.db.failed-20261008-1559`，未删除。依据测试数据库可重建的既定授权，新库已
重新创建并完成只读 `coser` 索引（6,221 项媒体、234 个目录）；原始媒体未修改。该运行态事故不属于
本次前端变更，也没有被写成测试通过。

`npm run audit:high` 未通过：当前锁文件仍报告 `brace-expansion`、`source-map-js` 和 `undici` 的既有
高危/中危公告。本次没有修改依赖，故未把该检查写成通过，也没有在界面修复中顺带升级依赖。
