# 非空审核实际验收：上游标签生成缺口

CR-2026-030 / POST-MVP-5 接入验证 / FR-INT-004 / FR-INT-016。
Owner app/semantic。此次为只读代码核查和本地 HTTP 诊断，无生产实现变更。

## 确认事实

internal/app/run.go 创建并接入词表、建议列表、审核、审核清理。
但未调用 NewTagJobService、NewTagJobProcessor 或 EnableTagJobs，
也未给 api.Dependencies.AITagJobs 赋值；已有 tagJobQueue 适配器不等于 worker 已启动。
internal/api/health.go 仅在 AITagJobs 非 nil 时注册生成任务路由。

本地 foliopath-multilib 两容器健康后，以测试管理员请求 Synthetic gamma：

- GET /api/v1/libraries/{id}/ai/tag-suggestions 返回 200，items=[]，eligible=1，completed=0，complete=false。
- POST /api/v1/libraries/{id}/ai/tag-suggestions/jobs，body={"mode":"missing"}，返回 404 resource_not_found。

没有通过直接注入运行数据库伪装完整正式链路，也没有创建成功的生成任务。
结束后停止两个容器。未访问 server222 或真实媒体，未执行全套测试。

## 对进度的影响

之前“标签审核清理实际镜像”仅证明空审核库的取消/重试链路；
非空清理与正式标签保留仅有 SQLite 组合证据，均不证明自动标签生成功能已交付。
标签建议生成仍未在独立 AI 主应用组合中实现完整生产链路。
非空实际审核验收应先完成标签生成服务、模型能力/代次约束及有界 worker 接入，
再经正式生成→审核→清理验证。此项优先于继续对空审核库重复验收。
人物生产接入、压力容量和发行也仍未完成。
