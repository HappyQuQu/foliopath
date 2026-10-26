# FIX-2026-08-31：S2C 人脸真实质量证据校验入口

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-250`、`INT-251`
- 影响不变量：公开媒体许可不能替代生物特征授权；anonymous core precision 不得下调；真实 ground truth
  不进入 git 或普通 CI

## 问题

既有合成 `face-score` 能验证公式和状态机，C/D 的质量入口只接受 ordinary-media。两者都不能证明最终
detector/embedder/cluster 在经授权真人 ground truth 上满足质量、错误合并和偏差门槛，也不能把 E 证据绑定
到最终双架构与供应链汇总。

## 修复

- 新增 `make verify-intelligent-media-face-quality`，严格读取逐项结果和 governed dataset manifest。
- manifest 必须是 schema v2、`biometric-ground-truth`、书面授权、禁止再分发，并同时允许 detection、
  verification、clustering evaluation；要求 privacy review、授权引用、受控角色、保留和删除程序。
- 至少 50 个 opaque identity、每个至少 20 张图片；结果必须完整覆盖 manifest，pair label 必须与 manifest
  identity 一致。
- 验证器重算 detection recall、verification recall/FPR、core/edge precision、肤色呈现/年龄呈现/光照/
  遮挡/多人图片 slice 和 Wilson 95% 区间。core precision 低于 99.5% 时失败关闭。
- S2 聚合器新增必需 `FACE_QUALITY_SUMMARY`，要求它与 C/D quality、strict native 和 supply-chain 使用
  同一 source commit、model package 和最终双架构镜像绑定。
- 原生 identity/outcomes/final-model evidence、供应链 manifest 和最终四份 summary 均按不可信外部证据处理：
  共享 release-evidence JSON owner 拒绝重复 key、未知字段和 trailing JSON value；文件边界继续拒绝 symlink/
  非普通文件，聚合器再计算原始字节 SHA-256。不能用重复 `gate_pass`/`result`、拼接第二个对象或未定义字段
  制造歧义。
- release-evidence JSON 文档统一限制为 16 MiB，并在 open 前后复核 non-symlink regular-file identity；native
  identity/outcomes/model evidence、supply manifest 与 aggregate summary 不再直接用 `os.ReadFile` 跟随输入
  symlink，也不能用超大 JSON 消耗无界内存。
- S2B/S2C quality input 与 governed dataset/model manifest 的现有 strict decoder 也新增递归 duplicate-key
  拒绝；嵌套 approvals、gate、items 或结果数组中的重复字段不能靠 Go JSON 的 last-value-wins 语义绕过。
  这些可能包含逐项质量结果的文档使用独立 256 MiB 上限，并同样要求 non-symlink regular file、在 open
  前后复核文件身份和 bounded read。

## 回归证据

- 通过 fixture 固定 50×20 intake、全部指标重算、confidence interval 和 commit/digest summary。
- 失败 fixture 覆盖 core false merge、缺偏差 slice、identity label 不一致、公开许可和 group assignment 未授权。
- 架构 fitness 固定两个 Make 入口及四份 summary 的聚合绑定。
- 共享 decoder、native outcomes、supply-chain manifest 和最终聚合器回归均覆盖 unknown field、duplicate key
  与 trailing value 的失败关闭。
- native identity 与 supply-chain manifest 的入口级回归证明 symlink document 被拒绝；共享 reader 另有
  regular-file boundary 测试。
- quality/face-quality 共用 manifest decoder 的回归覆盖嵌套 duplicate JSON key、symlink 和超限文件拒绝。

该入口本身不是现实质量证据；在真实私有输入和五方批准到位前 Gate 继续 No-Go。
