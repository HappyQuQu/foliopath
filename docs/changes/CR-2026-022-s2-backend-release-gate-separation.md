# CR-2026-022：S2 后端完成与最终发布 Gate 分离

- 日期：2026-09-01
- 目标版本：`POST-MVP-5` revision 2
- 需求：`FR-INT-001～020`、`NFR-INT-001～010`
- Owner：产品用户
- 影响阶段：S2A、S2B、S2C、S4
- 架构影响：无部署单元、技术栈、信任边界、持久化边界或模块依赖方向变化；无需新 ADR

## 决定

产品用户明确本轮目标是验证 FolioPath 后端功能是否实现，不训练、不发布模型或人脸结果，并要求最终发布
继续 fail-closed。由此把此前混入 S2 Backend Evidence Ready 的最终发行输入恢复到 S4/Release Gate：

- S2 验收 production capability、SQLite/API/app composition、默认不可用行为、取消/恢复/损坏/offline/
  原媒体不变矩阵、可重复本地容量，以及获授权媒体的私有功能验证；
- S4 继续独占最终审核模型/catalog、governed semantic/tag/video/face 质量与偏差、native Linux
  amd64+arm64 final-image pairing、最终联合 RSS/容量、SBOM/VEX/notices/provenance 和 privacy/compliance/
  security/release owner 签署；
- S2 Backend Ready 不填充 production reviewed catalog，不启用 face runtime/route，不批准候选权重或阈值，
  不等于功能可发布；所有 AI 设置继续默认关闭，缺少最终输入时稳定 unavailable；
- S3 可以按已接受 OpenAPI 开发消费者界面，但 release build 必须继续隐藏或禁用没有 S4 Go 的能力，不能
  以 mock、candidate package 或本机私有数据制造可用状态。

## 证据与任务归属

- `INT-209/215` 以现有 synthetic/candidate production-boundary 故障矩阵验收；最终模型完整容器强杀、
  损坏/恢复与供应链身份继续由 `INT-401/403/404/405/407` 验收。
- `INT-210` 以本机强制 100k/10k 查询及 100k×512 聚类容量验收；最终镜像、native 双架构与真实 inference
  联合负载继续由 `INT-402/403` 验收。
- `INT-227` 的 scorer、合法输入 validator、failure semantics 和 tag/video backend ranking contract 在 S2
  验收；最终真实质量结果与三方批准继续由 `INT-403/411` 验收。
- `INT-250` 以获授权 `coser` 私有功能证据、错误合并、source/offline/kill 和合成 100k×512 矩阵验收；
  50×20 governed identity、五维偏差、99.5% core precision 和阈值批准继续由 `INT-403/406/411` 验收。
- `INT-251` 签署的是 **Backend Ready / Release No-Go**：确认敏感数据边界、默认关闭、清除/备份、API/
  日志脱敏和 production fail-closed。它不是最终隐私/合规发布签署。

## 发布影响与回退

本变更不降低或删除任何发布门槛，只纠正阶段所有权。S4 任一最终证据缺失或失败时，reviewed catalog 保持
为空、face composition 保持缺席、消费者入口不得发布；核心媒体浏览、人工策展和现有故事板不受影响。
