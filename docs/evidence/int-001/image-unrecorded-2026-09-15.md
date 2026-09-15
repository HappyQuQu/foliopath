# 图片未记录覆盖率显式化

CR-2026-030 / POST-MVP-5 / FR-INT-005/015/016，Owner semantic/SQLite/API/前端 ai。

为既有设置响应补充 coverageRecorded（新服务总返回，旧服务可省略），无需迁移。
SQLite 在同一查询中判断 progress 行存在；HTTP 未记录时强制 complete=false。
已记录空进度与缺记录不再等价。总览与媒体库管理页隐藏未知的数值与进度条，显示说明；Storybook 同步。
新字段只影响设置读模型，不改变搜索结果和任务准入；旧服务省略字段时保留兼容渲染。

本地 Docker：5 项 SQLite 竞态、3 项语义 HTTP/序列化回归与 OpenAPI 合同测试通过。
Docker 前端 35 项（页面 24、adapter 8、query 3）通过；web 构建含类型检查通过。
make arch-check、generate:check、lint:openapi 通过；保留已有两个 health 文档警告及大 chunk 警告。

此增量尚未重建最终 app 镜像做“新代次无记录”的浏览器验收；不要将前一图片离线实际镜像
验证混同为此字段的实际进程验收。人物、视频操作按钮和完整发行验证仍未完成。
本轮两个本地测试容器已停止，保留合成夹具和数据卷，未接触真实媒体/server222。
