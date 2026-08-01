# CR-2026-014：扫描后派生媒体进度

## 状态

- 状态：Confirmed
- 变更等级：C1
- 目标版本：`POST-MVP-1`
- Scope revision / 范围状态：[POST-MVP-1 revision 5](../releases/POST-MVP-1-scope-r5.md)
  Frozen；继承 revision 4
- Change Record ID / 基线事件：CR-2026-014 / 2026-08-01 产品用户确认
- 提出日期：2026-08-01
- 产品负责人：产品用户
- 架构负责人：FolioPath maintainers
- Capability Owner：`internal/thumbnail`

## 用户问题与价值

- 用户 / JTBD：完整扫描结束后，仍能判断缩略图、视频封面和视频悬停预览何时真正可用。
- 当前问题及证据：durable `media_jobs` 已在后台继续执行，但公开合同只提供逐资产状态；扫描
  终态后状态页停止轮询，用户无法区分“索引完成”和“派生媒体完成”。
- 为什么必须进入目标版本：这是既有 thumbnail/storyboard 队列的只读可见性，不新增生成
  工作；没有该状态时，已确认的异步处理会被 UI 错误表达为全部完成。

## 范围

- 新增或改变的 FR/NFR：`FR-MED-013`；复用 `FR-SCN-006`、`FR-MED-001`、
  `FR-MED-009～010`、`FR-UI-008`。
- 明确包含：每库 thumbnail/poster 与 storyboard 两组聚合计数；queued/running/succeeded/failed；
  扫描或派生仍活动时有界轮询；失败数单列；storyboard eligibility 未确定时使用不确定进度。
- 明确不包含：通用任务中心、逐资产任务列表、派生批次历史、取消、重试、补齐或重建操作。
- 被替代/延期的现有范围：N/A；`POST-MVP-3` 任务中心仍为提案，不提前实现其 operation run。
- Scope-budget exception：N/A；小型只读纵向切片，不改变既有 worker 或发布预算。

## 架构影响

- Capability 与依赖方向：`internal/thumbnail` 拥有进度投影语义；SQLite 聚合当前 durable job；
  API 只调用 service；Web 通过生成 client 与单一 query key 消费。
- API / 用户流程：新增认证只读
  `GET /api/v1/libraries/{libraryId}/media-processing`；媒体库状态页把扫描与派生媒体分开显示。
- 数据 / migration / 派生状态：无 migration；只读取现有 `assets` 与 `media_jobs`。
- 安全、隐私与信任边界：不打开媒体文件，不返回路径、stderr 或逐资产信息；继续要求认证。
- 性能、容量与并发：按已有 `media_jobs_library` 索引做每库有界聚合；仅 active 时 1.5 秒轮询。
- 部署、升级、备份、恢复与观测：无新部署单元；重启后投影直接反映 durable 恢复状态。
- 平台、依赖、许可证与供应链：N/A；无新依赖。
- ADR：N/A；不改变任务一致性、所有权、持久化或部署边界。

## 质量属性场景

- 刺激：完整扫描变为 succeeded，而数千个缩略图或 storyboard 仍在队列中。
- 环境：四核/4 GiB、多媒体库、grid 优先且 storyboard 分批 admission。
- 系统响应：扫描进度保持完成；派生媒体区继续轮询并分别显示真实计数，terminal 后停止。
- 可测结果：无伪造 ETA；失败不计为成功但计入 processed；storyboard 总量未稳定时不显示百分比。

## 风险与验证

- 风险 ID / 新风险：复用 R-005、R-013、R-015、R-016；不新增风险。
- Fallback / 回滚：移除只读 route/query/UI，不影响 durable job 或已生成缓存。
- 正常、边界、失败、恢复测试：空库、扫描活动、grid 活动、storyboard 等待/活动、terminal
  failure、missing library、重启后现有队列。
- Fixture 与目标环境：SQLite synthetic catalog；Go HTTP/unit；Web query/UI 与现有浏览器矩阵。
- 验收证据要求：OpenAPI/生成一致性、SQLite/API/unit、Web type/unit、架构与契约检查。

## Gate 影响与决定

- 需要重跑的阶段/切片 Gate：VSP-S2/S3 适用回归；完整发布 Gate 仍独立。
- 产品决定：Confirmed；扫描完成不等于缩略图和视频预览完成，必须继续显示后台进度。
- 架构决定：Go；只读 capability 投影，不复制 job 状态机或引入通用 task owner。
- 安全/数据评审：Go；无新写入、路径或媒体访问边界。
- 最终结论：Go。
