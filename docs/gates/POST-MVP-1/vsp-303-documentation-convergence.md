# VSP-303 发布文档与追踪收敛

## 当前结论

**Prepared, blocked by VSP-302 — 本 Gate 尚未签署。**

本文是 `VSP-303` 的收敛审计，不是 Integrated Done 或发布批准。依赖
[VSP-302 目标平台与资源复验](vsp-302-target-platform.md)仍为 Pending，因此已完成的
文档只能记录当前候选事实、证据和残余限制；必须在原生双架构成对证据成功后复核并签署。

## 收敛矩阵

| 文档 / 合同 | 当前收敛内容 | 状态 |
| --- | --- | --- |
| `README.md` | 区分 MVP 候选与 Post-MVP/1 storyboard 候选；说明交互、VSP-301 与剩余 Gate | 已准备 |
| `POST-MVP-1-scope.md` | 冻结 FR、非目标、验收和继承安全约束 | 已冻结，不原地改写 |
| `product-requirements.md` | `FR-MED-009～011`、`FR-UI-008` 当前交付阶段 | 已同步 |
| `user-flows.md` | hover intent、播放、停止、fallback 与非模态预览关系 | 已同步 |
| `ui-design.md` | fine pointer、reduced-motion、共享 controller、a11y 与视觉矩阵 | 已同步 |
| `api/openapi.yaml` / `api-design.md` | `variant=storyboard`、状态、布局、错误与生成 client | 已实现且权威合同已同步 |
| `data-model.md` | migration 11、variant 身份、layout、优先级、LRU 与恢复 | 已同步 |
| `security.md` | 锚定只读 FD、subprocess/resource cap、原子发布与错误脱敏 | 已同步 |
| `deployment.md` | 无新增 mount/服务/配置；生产镜像包含 FFmpeg；候选平台阻断 | 已同步 |
| `testing-strategy.md` | 分层证据、VSP-301、原生成对 artifact 校验和未执行边界 | 已同步 |
| `risk-register.md` | R-018 已有缓解与 VSP-302/VSP-S4 残余阻断 | 已同步 |
| `feasibility-study.md` / `VSP-002` | 算法、资源和运行时可行性；原生最终候选不由模拟替代 | 已同步 |
| `architecture/traceability.md` | owner、合同、数据、风险、证据和 Gate 链 | 已同步 |
| feature task list | VSP-301 完成；VSP-302～304 顺序与阻断 | 已同步 |
| release notes | 候选行为、运维影响、限制与未完成条件 | 草案 |

## 签署前必须完成

- [ ] VSP-302 同一 commit/run 的原生 `linux/amd64`、`linux/arm64` artifacts 都为 passed。
- [ ] `make verify-storyboard-evidence` 对成对 artifacts 成功。
- [ ] 同源码 browser job 与适用的 Go、migration、race、integration/E2E 检查成功。
- [ ] 把发布说明中的 `DRAFT / NOT RELEASED` 替换为真实版本标识、commit、镜像 digest
  和完整验证链接；未发布时不得删除该警示。
- [ ] 复核本文矩阵没有“计划实现”与真实实现相冲突，也没有把未跑检查写成通过。
- [ ] 勾选 feature task list 的 `VSP-303`，再交给 `VSP-304` 做
  `VSP-AC-001～008` 最终聚合与签署。

## 当前阻断

GitHub Actions 已正常解析并创建原生 storyboard jobs，但仓库账户付款失败或 spending
limit 使 jobs 在 runner 分配前终止；没有 test step，因此没有原生证据。本机 arm64 预检
成功；跨架构 amd64 模拟在 media-root 安全初始化处按设计失败关闭。二者都不能单独关闭
VSP-302，也不能提前关闭本 Gate。
