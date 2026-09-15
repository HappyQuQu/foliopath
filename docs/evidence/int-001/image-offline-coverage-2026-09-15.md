# 图片模型离线不清零覆盖率

CR-2026-030 / POST-MVP-5 / FR-INT-005/015/016。Owner SQLite/semantic，既有设置读取回归。

## 原因与修复

原 GetSemanticLibrarySettings 用 model.state=available 筛选整个进度查询；不可用时跳过进度并返回默认 0/0。
现在单条 SELECT 在同一 SQLite 快照读取设置、活动代次和已记录进度，模型不可用不丢弃计数。
同时不再吞掉代次查询的数据库错误。

重要兼容边界：图片 activeGenerationId 仍在模型不可用时为 null，保留 Service 请求任务的既有门禁。
视频只读状态中的代次仅标识记录，不参与任务准入。开启分析仍要求 available 模型，未修改写入政策。
没有新增表、迁移、后台任务或媒体读写。OpenAPI 补充说明并同步生成与摘要。

## 验证

- 本地 Docker 5 项仓储竞态回归通过：默认/可用模型要求、缺库、时钟回拨、源变化失效、新增离线保留用例。
  新用例包含 available→unavailable→available、失败/过期计数、覆盖率 revision 不变、离线库关闭分析、拒绝不可用模型开启。
- Docker OpenAPI 合同测试、make arch-check、generate:check、lint:openapi 通过。
  OpenAPI 仍有两个既有 health 端点无 4XX 警告，Vite 保留既有大 chunk 警告。
- 最终实际镜像 manifest-list：`sha256:0c222e6e0e0048d83ca9b8ab991441f459ba6f7c5047cdb82a1d6f0775d930fd`。
- 实际本地停止 AI 容器：图片 eligible=1/completed=1/revision=11 保留，activeGenerationId=null；
  视频 eligible=2/completed=1/degraded=1/revision=18 保留。浏览器同页仅模型文案变化，图片不再变为 0/0。
- 实际离线请求图片与视频 missing job 均为 409 model_unavailable；未创建任务。
- 恢复后探针检查原计数和代次；仅使用 `/tmp/foliopath-app-fixtures.wDId08` 合成媒体、隔离数据卷。

## 边界

本轮未运行全量七项验证、全浏览器 E2E、容量/并发矩阵。人物接入、视频操作按钮等仍未完成。
尚无活动代次/从未产生进度时的默认覆盖率语义未在本轮重设计；本修复只解决已记录图片覆盖率在模型离线时消失。
无 server222、无真实媒体、无提交推送；测试后停止本地两个容器，保留数据与镜像。
