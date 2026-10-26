# ADR-0015：人脸组合模型包与独立 generation 激活

## 状态

**Accepted for fail-closed backend implementation（2026-09-01）**。产品方向已明确授权 S2C 进入后端实现，
同时要求发布保持 fail-closed。本 ADR 因而接受方案 3 的模型生命周期、事务归属和 generation 边界，允许实现
production parser、purpose-aware catalog/schema 和独立 face generation commit；它不批准任何候选模型字节、
不允许把未审核模型加入内建 catalog，也不允许在 S2C Release Gate 前把 face runtime、route 或 worker 注入
production composition。

架构接受与发布准入是两个独立状态。最终模型许可/质量、原生双架构、隐私/合规和供应链签署仍是激活与发布
硬门槛；缺少任一输入时 catalog 保持无可用 face entry，application composition 继续失败关闭。

## 背景

POST-MVP-5 revision 2 已接受匿名人脸聚类的 capability、data 与 OpenAPI 合同，并已实现默认不可达的
repository、worker、HTTP adapter 和 YuNet/AuraFace candidate runtime 边界。当前模型生命周期仍只拥有
`semantic_image_text`：

- `internal/aimodel` format v1 与 `ai_models` schema 只接受 semantic package；
- `ai_model_state` 与 activation commit 只拥有一个 semantic active pointer，并创建 `semantic_generations`；
- `face_generations` 能绑定 detector/embedder component identity，但没有 production 创建/激活 owner；
- 直接复用现有 activation commit 会错误退休 semantic generation，或产生只能由测试 seed 的 face generation。

因此“把候选文件加入 catalog”不是例行配置；它改变模型 purpose、激活事务和 generation ownership，必须先有
ADR。

## 备选方案

1. **让 face 复用 semantic singleton**：实现最少，但激活人脸会替换图片/文本模型，违反两个 capability
   独立可用和失败回退合同。
2. **detector/embedder 各自作为独立可激活模型**：组件升级灵活，但会暴露两次激活之间的混合代次，并要求
   跨 operation 补偿事务。
3. **一个审核组合包、一次 face generation 激活**：包内分别绑定 detector/embedder 文件 hash，激活同时
   校验两张图、预处理/后处理 contract 和固定 pipeline fixture，再原子创建 face generation。
4. **绕过 aimodel，在 `internal/face` 自建扫描/复制/catalog**：会复制来源锚定、hash、quota、availability、
   operation 和供应链规则，违反 canonical owner。

## 决定

采用方案 3：

- 扩展一个新的、精确版本的组合 package format；`purpose=face_detection_embedding`，只允许唯一
  `face_detector` 与 `face_embedder` role，并固定 detector input/output/NMS、decode/resize、alignment、
  embedding preprocess/storage 和 threshold-profile contract ID；不复用或重新解释 semantic v1/v2；
- `internal/aimodel` 继续唯一拥有 package discovery、来源锚定、托管复制、direct availability、quota、
  operation 与 catalog；公共 API 仍只接受 catalog 产生的 opaque model ID；
- activation worker 按已审核 purpose 分派。semantic commit 继续唯一更新 `ai_model_state`/
  `semantic_generations`；face commit 在单一短事务内创建并激活 `face_generations`，不得更新 semantic pointer；
- face generation 同时记录组合 package/model identity、detector/embedder 文件 hash、512 维、transform version
  与冻结 threshold profile；任何 component 或 contract 变化产生新 generation；
- 激活前同时验证两个 kernel-handle 文件、精确 tensor ABI、固定 pipeline fixture、取消边界和最终模型来源；
  任一步失败保留旧 active face generation 与所有最后可靠 observation/cluster/manual state；
- 新 face generation 激活不自动切换已启用库的可见 cluster。各库通过现有 durable rebuild job 建新
  observation/cluster build，只有完整成功后原子切换 active build；失败/取消/offline 保留旧代；
- 模型包仍不随 FolioPath 镜像捆绑。最终 catalog、notices、SBOM/VEX/provenance 与再分发批准必须按精确
  detector/embedder bytes 共同签署。

## 激活与发布门槛

- 最终 detector/embedder 精确来源、hash、许可、训练数据/biometric 合规与再分发签署；
- governed face quality 通过 detector recall、verification、偏差与 core precision 99.5% 下界；
- native Linux amd64/arm64 的完整 pipeline、取消、泄漏、畸形输入、RSS 与联合容量；
- migration/transaction、旧代回退、direct source 失效、升级/恢复和并发激活证据；
- security、privacy、compliance、inference、release owner 接受，并同步 package contract、OpenAPI、data model、
  deployment、testing 与 release verifier。

## 后果

正面：semantic 与 face 生命周期互不覆盖；detector/embedder 永不以半组合代次可见；复用现有安全模型来源
边界，不增加部署单元或网络获取。

代价：aimodel catalog/parser、SQLite migration、activation repository/worker 和供应链 evidence schema 都需要
purpose-aware 扩展；这些 fail-closed 后端边界已经实现，但最终模型未获批准前，内建 catalog 和 application
composition 必须保持无可用 face entry/runtime。拒绝具体模型时，S2C 保持全局不可用，不回退到测试 seed、
任意路径模型、OpenCV subprocess 或云服务。

## 先行可执行证据

`spikes/int001-model-package-v2.ParseFaceV3` 已固定组合包的三项唯一 role、七项 transform contract、逐组件
SPDX-safe license ID、ASCII package/version token、阈值 profile 文件 hash、64 KiB manifest 与 4 GiB package 上限，并拒绝 semantic
version/purpose 混淆、缺失/重复 role、嵌套路径、未知字段/contract、duplicate key 和 trailing JSON。
`make spike-ai` 会执行这些测试。production 实现已重新落入 `internal/aimodel` 的 canonical owner，没有
导入 spike；本 ADR 接受和后端实现都不构成具体模型准入。
