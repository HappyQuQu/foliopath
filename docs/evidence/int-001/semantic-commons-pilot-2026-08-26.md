# Wikimedia Commons Cosplay 语义检索 pilot — 2026-08-26

状态：公开许可小样本开发证据；**不满足 INT-S0 质量或资源 Gate，不批准模型选型**。

## 数据与复现边界

- 数据源：Wikimedia Commons API 与原图地址；10 张 Cosplay/人物照片，10 个固定 page ID/revision，
  合计 33,216,272 bytes。
- 许可：逐项记录作者、CC BY/CC BY-SA 版本、来源页、原始 SHA-1、下载 SHA-256、尺寸与字节数。
  `semantic_public_fixture_fetch.py` 会在下载前复查当前 Commons 元数据，并在落盘前后校验内容。
- 文件只存在临时目录，未随 Git 再分发。Commons 的公开许可不消除肖像权、隐私或当地法律义务；
  本 pilot 只标注可见服装、颜色、人数、场景和道具，不评估人物或角色身份。
- 24 条查询由 12 组中英文配对构成，含 10 组单一相关图和 2 组多相关图。它用于验证评分链和发现
  候选差异，不是至少 1,000 张的代表性验收集。

## 结果

| 候选 | 中文 R@1 / R@3 | 英文 R@1 / R@3 | 中文 / 英文 mean relevant R@3 | 图片 P95 | 直接原图后 RSS |
| --- | --- | --- | --- | --- | --- |
| SigLIP 2 base patch16 224 | 1.000 / 1.000 | 1.000 / 1.000 | 0.944 / 0.931 | 58.244 ms | 1,709,178,880 B |
| SigLIP base patch16 224 | 0.917 / 1.000 | 1.000 / 1.000 | 0.944 / 0.903 | 59.744 ms | 1,268,580,352 B |

SigLIP 1 的唯一首位未命中是中文“黑色长发穿红色金色盔甲的人”：黑色蝙蝠舞台图排第 1，红金
盔甲图排第 2。两模型在宽泛的“彩色头发人物近景”查询中，Top 3 都只找回部分人工相关项；因此单看
“至少一张相关图命中”的 R@1 会高估质量，后续验收必须同时保留多相关 recall/nDCG 类指标和失败分类。

中英文 Top 1 配对一致率两者都只有 0.917；即使单语言 R@1 为 1，宽泛查询也可能在多个相关项之间
选择不同首位，不能把配对不一致直接算成检索错误。

## 资源数字为何不能验收

本次 `input_pipeline` 明确为 `original-public-file-direct-decode-diagnostic`：Pillow 直接解码最高
5,388×3,592 的原图，进程 allocator 可能保留大图中间内存。这不是计划中的生产输入链。正式 runtime
必须复用受像素/尺寸限制的 libvips 缩放产物，再在完整 FolioPath 进程、并发浏览和 4 GiB cgroup 内测量。
因此上表 RSS 只说明“不能直接解码任意原图”，不能用于认定 SigLIP 1/2 通过或失败 3.2 GiB Gate。

## 当前决定

1. 两个候选都继续保留，SigLIP 2 在本 pilot 的中文首位质量较好，SigLIP 1 的模型包和常驻资源较小；
   样本规模、输入链和运行时都不足以选型。
2. 下一轮先扩到至少 1,000 张合法代表性图片及失败分类，再用同一生产缩放输入跑两个候选；不得用
   这个 10 张 pilot 替代 Gate。
3. 生产 ONNX/Linux 双架构、4 CPU/4 GiB 联合容量、模型许可/SBOM/再分发签署仍全部 Pending。

机器结果：

- [SigLIP 2 JSON](semantic-commons-pilot-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 JSON](semantic-commons-pilot-siglip1-darwin-arm64-2026-08-26.json)
- 数据 manifest 与下载校验器位于 `spikes/int001-ai/testdata/semantic-score-commons-pilot.json` 和
  `spikes/int001-ai/semantic_public_fixture_fetch.py`。
