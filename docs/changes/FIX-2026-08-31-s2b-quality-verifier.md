# S2B 合法质量评分入口

- 日期：2026-08-31
- 状态：**Implemented locally / approved dataset evidence pending**
- 类型：已批准 S2B 切片的 evidence tooling
- Requirement：`FR-INT-006～007`、`NFR-INT-003`、`INT-227`
- 目标版本与阶段：`POST-MVP-5` revision 2，S2B Backend Evidence Ready
- Owner：product / ML / QA quality evidence
- 关联 Gate：[INT-S2B](../gates/POST-MVP-5/int-s2b-backend-evidence-ready.md)

## 变更

在隔离 `spikes/int001-ai` module 增加 `quality-score`，并提供根级
`make verify-intelligent-media-quality`。输入由严格 JSON 解码，绑定：

- governed dataset manifest 的实际 SHA-256 与最终 model package SHA-256；
- product、ML、QA 三个不透明批准引用；
- 经批准的 tag 宏平均 precision、recall 和人工接受率门槛，以及逐 tag TP/FP/FN/review counts；
- 至少 100 个 MP4/MOV/MKV，2 秒～2 小时，4/10 帧、motion/static、indoor/outdoor 覆盖；
- 中英文 query、相关视频集合和最多 20 个实际结果 ID。

评分器重新计算逐 tag precision/recall、宏平均、人工接受率和视频 Top-20 query success；任一 tag
批准门槛未达或视频成功率低于冻结的 80% 时 `gate_pass=false`，CLI 返回失败。结果 ID 必须唯一且属于
已登记视频，不能提交自报的布尔 success。

`DATASET_MANIFEST` 必须是非 synthetic 的 schema v2 `ordinary-media` manifest，已经过现有治理 validator，
同时授权 `semantic-evaluation` 和 `video-evaluation`。质量输入的 100 个视频 ID 必须全部存在于该 manifest；
hash 不符、许可/治理字段缺失或仅有合成数据均失败关闭。

## 使用

```sh
make verify-intelligent-media-quality \
  QUALITY_INPUT=/secure/evidence/s2b-quality-results.json \
  DATASET_MANIFEST=/secure/evidence/dataset-manifest.json
```

真实媒体和标注不进入仓库。命令输出结构化 report，但只有退出码 0 且 `gate_pass=true` 才能进入
`INT-227` 复审。

## 验证与边界

首次从根模块执行 `go test ./spikes/int001-ai` 按 Go nested-module 边界失败；随后改为在独立 module
目录执行，并修正 Makefile。实际通过：

- `cd spikes/int001-ai && go test ./... -run 'S2BQuality|QualityScore' -count=1`；
- `go test ./tests/architecture -count=1`。

测试覆盖合格报告、tag precision 失败、视频 70% 失败、少于 100 视频、缺审批、中英文不足、未知结果
ID、manifest hash 不符和 synthetic manifest 拒绝。仓库仍没有真实审核集、真实模型结果或三方批准，
所以本工具不完成 `INT-227`，也不改变 S2B No-Go。
