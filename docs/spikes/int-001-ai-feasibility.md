# INT-001：本地智能检索与人脸聚类可行性 spike

## 状态

- 状态：Planned / 未执行
- 对应：FTR-INT-001、CR-2026-021、ADR-0013、R-024～R-030
- 目标：以最小隔离原型回答能否在 FolioPath 的单容器、4 CPU/4 GiB、100k 媒体目标下安全交付
- 限制：原型不得进入生产 import graph，不修改 OpenAPI/migration，不读取开发者真实媒体库

## 要回答的问题

1. 哪个许可清晰的视觉-文本模型能在 CPU 上提供可接受的中英文套图语义结果？
2. 10 万图片/视频帧向量使用精确扫描是否足够；不足时哪个 ANN 方案满足恢复和双架构要求？
3. 人脸检测/embedding 在人物套图中能否形成足够高 precision 的匿名核心簇？
4. 全流程会不会让 4 GiB 实例 OOM、拖慢浏览，或使镜像/备份/升级不可接受？
5. 审核源下载和 `/models:ro` 离线发现能否在不开放任意 URL/路径的情况下可靠安装、直接读取并恢复？

## 数据集

- 全部使用 `tests/fixtures` 下的合成、公开许可或专门授权数据；绝不读取用户真实媒体库。
- 语义集至少 1,000 图片、100 视频，每项有中英文查询与 relevance judgment；覆盖人物全身/半身/
  特写、多人、服装、室内外、舞台、道具、光照、背影、遮挡和相似套图。
- 人脸集至少 50 个身份、每人 20 张，包含妆容/假发/角度/遮挡/多人/相似服装；按身份划分 ground truth。
- 100k 容量集可重复生成向量和合成媒体元数据，但质量指标必须用真实授权图，不能用随机向量冒充。
- 仓库只提交生成器、manifest、哈希、标签结构和汇总；不得提交无再分发许可的人脸图片。

## Workstream A：模型与运行时

候选仅供比较：多语视觉-文本 ONNX 模型；许可清晰的人脸检测与 embedding ONNX 模型。逐项记录
代码许可、权重许可、来源、哈希、模型大小、opset、预处理和输出维度。

执行矩阵：

- 原生 linux/amd64、linux/arm64；4 CPU/4 GiB；冷/热加载；单张与小批量；
- 256px～100 MP 来源经现有安全缩放链；损坏、截断、超大和取消；
- session 创建/销毁、1000 次循环 RSS、超时、线程数、退出和模型 hash 不匹配；
- 中英文 query、图文归一化、一致输入的跨架构数值容差；
- 无网络环境、只读根、`/library:ro` 与 `/app/data` 可写。
- 模型获取矩阵：签名发行源、部署者镜像、无外网 `/models:ro`；下载取消/续传/重定向/错哈希/
  磁盘满；目录 symlink/特殊文件/嵌套 mount/可写/替换/消失；托管复制与直接读取重启恢复。

通过：整进程 RSS ≤ 3.2 GiB、无持续泄漏、AI 并发 1、浏览 P95 退化 ≤ 20%、100k 图片估算/实测
≤ 24 小时、质量达到 feature 暂定门槛，且权重可在目标分发方式下合法使用。

模型获取额外通过条件：公开请求不含 URL/路径；未知文件不进入 runtime；下载/复制失败不替换现用模型；
直接来源变化后失败关闭且普通浏览继续可用；没有已运营的国内镜像证据时结论必须写“不承诺镜像”。

## Workstream B：向量存储与检索

比较：SQLite blob + 应用内精确扫描、许可/部署可接受的 SQLite 向量扩展、独立可重建 ANN 文件。

对 10k/50k/100k image vectors 以及额外 video-frame/face vectors 测量：

- build/incremental build/atomic activate 时间和峰值 RSS；
- 冷/热查询 P50/P95/P99、Top-K recall、过滤后补充候选行为；
- DB、WAL、索引和备份体积；并发 browse/search/backfill 的尾延迟；
- 强杀、截断索引、缺文件、版本不匹配、重建、旧 generation 回滚；
- amd64/arm64 构建、许可证、漏洞和 distroless 运行闭包。

通过：P95 ≤ 750 ms、P99 ≤ 1.5 s；索引损坏不损坏 DB 且可自动重建；100k 派生增量初始预算
≤ 500 MiB；结果排序有稳定 tie-breaker；没有隐藏网络或不可审计动态 extension 加载。

## Workstream C：人脸聚类与人工约束

测量 detector recall、错误框、embedding pair ROC、聚类 precision/recall、核心/边缘成员、人物间
错误合并、同一人在妆容/假发/背影下的拆散，以及多人图 face 定位。

至少验证：

- 高阈值核心簇、低阈值候选边分层；
- 用户从匿名组建人物、整组并入人物、单 face 归入、移动、拆分和 named-person merge；
- cannot-link 与 manual assignment 经重聚类、模型升级和重启仍生效；
- 源 fingerprint 变化使旧 face observation 失效；库删除/AI 清除不触碰原媒体；
- 不根据名称、目录、服装或网络信息做身份推断。

通过：核心簇 precision ≥ 99.5%，人工排除复发为 0。若 precision 不达标但 pair 建议可用，结论必须是
“降级为建议”，不能把 recall 换成错误身份合并。

## 产物

- `docs/evidence/int-001/`：环境、命令、manifest、摘要、原始机器可读指标、失败样本分类；
- 候选原型/benchmark：隔离于 production import graph；
- 模型/权重/运行时/vector 引擎许可证与 SBOM；
- 资源、质量、隐私和供应链结论；
- 更新 ADR-0013、风险登记与 INT-S0 Gate。

## Go / No-Go

- 三个 workstream 均通过且无未处置严重风险：建议 S0 Go。
- 语义通过、人脸不通过：只提议图片语义/标签/视频范围，人脸留后续版本。
- 运行时或索引资源不通过：整个 feature No-Go，除非正式缩小容量/平台承诺。
- 权重许可不通过：对应能力 No-Go，不允许用下载脚本规避分发责任。
