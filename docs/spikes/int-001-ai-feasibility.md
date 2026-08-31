# INT-001：本地智能检索与人脸聚类可行性 spike

## 状态

- 状态：Local exploration closed / S0 对 revision 1 A+B 为 Go
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

## 2026-08-25 初步执行记录

- 已增加隔离标准入口 `spikes/int001-ai`，不进入生产 import graph；包含严格 JSON manifest 校验、
  内存/SQLite BLOB 精确向量基线，以及 Linux `/models:ro` 安全发现实验。
- dataset manifest v1 已限制为仅合成 fixture，所有非合成数据必须使用 v2；人脸 ground truth 的可执行
  intake 门槛只接受列举的评测用途和
  访问角色，要求保留截止日、固定删除动作、禁止再分发、不透明授权与隐私评审引用；公开图片许可不能
  代替生物特征处理授权。仓库只含占位模板和拒绝测试，合法真实数据、签署、受控存储及删除演练仍缺失。
- macOS arm64 开发基线见 [`docs/evidence/int-001`](../evidence/int-001/README.md)。100k × 512 维时，
  内存精确扫描 P95 为 34.365 ms，SQLite 逐行精确扫描 P95 为 202.204 ms；这不是 Linux Gate 结果。
- SQLite 100k × 512 维数据库已达 410,619,904 bytes，尚未计入视频帧、人脸向量和 WAL，接近 500 MiB
  总派生预算。当前证据不支持直接冻结 float32/512 维方案，必须评估降维/量化/分表预算或正式缩小范围。
- 官方来源初审保留 ONNX Runtime、SigLIP 2、YuNet 和 SFace 作为 spike 候选；Chinese-CLIP 权重许可未
  形成明确证据，暂不得批准；InsightFace 公共预训练包存在非商用研究限制，不作为默认可分发候选。
- `/models` scanner 在非 Linux 平台失败关闭；Linux 原生 read-only、symlink、特殊文件、bind mount、
  hash mismatch、消失/替换和双架构矩阵仍 Pending。
- 已固定 OpenCV Zoo revision 下的 YuNet/SFace 候选 hash，并在 macOS arm64 ONNX Runtime CPU 单线程
  跑 1000 次随机张量：两者 P95 均约 16 ms，组合进程最终 RSS 约 172 MiB。SFace 循环 RSS 单调增加约
  0.56 MiB，尚不能区分 Python allocator 与 native leak，必须由 Linux C API 长循环复核。
- 临时许可示例图已跑通 `检测 → landmark 对齐 → 128 维 embedding`，但无身份/consent ground truth，
  检测数量不得写成 recall 或聚类质量。示例图不进入仓库。
- 100k × 512 随机向量量化：float16 payload 102,400,000 bytes、Recall@20 1.0；per-vector int8 payload
  51,600,000 bytes、平均 Recall@20 0.9975、最差 0.95。随机分布不能替代真实模型 embedding，因此只把
  float16 列为优先验证方向，不接受 int8 选型。
- 人脸评分工具已覆盖 pair threshold/ROC、cluster pair precision/recall、核心簇 99.5% 门槛、cannot-link
  与 manual assignment；合成正确 fixture 通过，注入跨身份核心簇错误后失败。这只证明 scorer，不是模型成绩。
- 隔离人脸状态机进一步覆盖后台匿名 core/edge、core 创建人物、另一匿名 core 并入人物、单脸确认、
  cannot-link 和换代重聚类。实现中拒绝让 edge 随批量操作自动成为人物成员，并让所有 manual assignment
  排除在后续自动聚类之外。它只证明用户关系可以有单一 owner，不证明真实 detector/embedding/cluster
  质量或生产事务。详见
  [`face-cluster-state-machine-2026-08-26.md`](../evidence/int-001/face-cluster-state-machine-2026-08-26.md)。
- `coder/hnsw` 10k × 128 随机向量比较中，M16/ef64 平均 Recall@20 仅 0.485；提高到 M32/ef512
  才达到 1.0，但单次 build 已 51.7 秒。截断索引可拒绝并全量重建，同一 graph 两次 export 不同 hash，
  importer 还缺 hostile length 分配边界。鉴于 exact 100k 本机已远低于延迟预算，当前不选择该 ANN，
  不浪费时间扩到 50k/100k；只有 Linux 并发 exact 失败才重开 ANN 选型。
- 2026-08-26 原生 Linux/arm64 4 CPU/4 GiB/no-network 三次联合负载中，100k×512 float32 exact search
  P95 为 229.843～258.371 ms，取消/重启正确；但 keyset browse proxy 相对退化 16.25～23.22 倍，未通过
  20% Gate。该 proxy 的微秒级基线不能代替生产 browse 测试，同时 410.6 MB 已排除 float32 combined
  layout。后续只继续真实 embedding 的 float16/降维容差与生产 browse path；失败则缩减视频/人脸范围。
- 同一受限矩阵的 float16 三次 DB 均为 136,880,128 bytes，exact search P95 138.854～156.912 ms，
  cancel/restart 正确；SQLite page layout 令文件缩减大于 raw 2:1。它是当前唯一值得继续的 exact 容量
  路径，但随机向量不能批准真实 embedding Recall，browse 相对 Gate 也仍失败，不能据此冻结选型。
- SigLIP 2 base patch16 224 已固定 revision，并在完全离线的 macOS arm64 PyTorch/F32 管线跑通 8 张
  程序绘制图片和 16 条配对中英文查询：图片推理 P95 58.414 ms，观测 RSS 1,170,636,800 bytes，英文
  Recall@1 1.0、中文 0.875。该小集合只证明多语图文管线可工作，不代表真人/Coser 质量，也不替代
  ONNX/Linux、并发浏览、泄漏和 100k backfill。完整模型包 1,539,458,338 bytes，且包含权重、tokenizer、
  preprocessing/config；现有单文件 catalog 不足以保证整包原子校验和激活。
- 较小的 SigLIP 1 base patch16 224 candidate B 也已固定 revision/hash：排除重复 PyTorch bin 后模型包
  815,877,562 bytes，观测 RSS 723,795,968 bytes，图片 P95 60.373 ms；同一合成集英文 Recall@1 1.0、
  中文 0.875。它降低常驻内存但没有提高吞吐，并引入 SentencePiece native tokenizer 依赖；过小夹具无法
  区分两者质量，不能据此选 SigLIP 1。
- 模型 catalog spike 已增加兼容的 multi-file schema v2：固定 package directory、1～128 个直接文件、
  每文件 size/hash 和规范化 package digest；两套 SigLIP 运行文件均可严格校验。Linux scanner 已实现整包
  缺失/额外/篡改/symlink 失败关闭，amd64/arm64 可交叉编译；由于本机没有可用 Linux 容器，只能保留
  native 执行 Pending。macOS no-replace 原子发布测试已证明完整 staging 一次可见、已有 generation 不覆盖、
  失败保留 staging；它尚未连接下载流程，也不等于 signed manifest 或数据库 generation 激活。
- 使用已有图片 embedding 组合 4 帧视频故事板代理：归一化均值在 8 条查询上中英文 Recall@1 均为 1.0，
  最大帧相似度仅为中文 0.5/英文 0.75，暴露共享帧导致并列误命中的问题。它只支持下一轮优先评估 mean
  pooling，不证明真实视频抽帧质量、10 帧路径或“无重复 FFmpeg admission”。
- 未执行代表性语义质量、生产格式下的双候选比较、人脸 ROC/聚类、生产 Go/C API runtime、并发浏览、取消/恢复和
  受限 Linux 容量，不能据此勾选任何 INT-S0 Go 条件。

## 2026-08-26 公开许可语义 pilot

- 从 Wikimedia Commons 固定 10 张 Cosplay/人物照片的 page ID、revision、作者、CC BY/CC BY-SA
  许可、尺寸、字节数、原始 SHA-1 和下载 SHA-256；原图仅下载到临时目录，不在 Git 再分发。下载器会
  复查当前 Commons 元数据并校验内容，且明确公开许可不消除肖像权与隐私义务。
- 建立 12 组中英文配对查询（24 条），只标注可见人物、服装、颜色、人数、场景和道具，不标注 Coser、
  模特或角色身份。scorer 新增多相关项 Top-3 recall，避免“至少命中一张”掩盖宽泛查询漏召回。
- 串行离线 PyTorch/F32 结果：SigLIP 2 中英文 Recall@1 均 1.0；SigLIP 1 中文 0.917、英文 1.0，
  中文红金盔甲查询的相关图排第 2。两者在宽泛彩色头发查询都漏掉部分相关项，10 张小样本不能选型。
- 本轮直接解码最高 5,388×3,592 原图，SigLIP 2/SigLIP 1 推理后 RSS 分别为 1,709,178,880 与
  1,268,580,352 bytes。该路径不是生产方案，数字只证明任意原图直解码不安全；正式比较必须复用
  受控 libvips 缩放输入，并在完整 4 GiB 进程与浏览并发下重测。
- 详细来源、结果和局限见
  [`semantic-commons-pilot-2026-08-26.md`](../evidence/int-001/semantic-commons-pilot-2026-08-26.md)。
  至少 1,000 张代表性合法质量集仍 Pending，并已归入 Slice B Backend Evidence Gate；不再扩张 S0 pilot。
- 同一 pilot 又以对齐现有 grid 合同的 512px WebP/quality 82 Pillow surrogate 串行运行三次。24 条
  查询的 Top-1 与 first-relevant rank 均未改变；SigLIP 2/SigLIP 1 模型进程 RSS 中位数相对原图
  直解码约下降 31%/43%。本机没有 libvips CLI，该结果只支持“必须复用有界派生输入”的方向，不能
  证明生产像素等价或关闭 4 GiB Gate。详见
  [`semantic-bounded-input-2026-08-26.md`](../evidence/int-001/semantic-bounded-input-2026-08-26.md)。
- 当前生产 `imagevips.Processor` 随后在缓存的 Linux/amd64、libvips 8.16.1 QEMU 镜像生成同一批
  512px WebP，生产 adapter 测试通过。输出尺寸与 Pillow 一致，平均 normalized pixel MAE 0.008644；
  两候选全部 Top-1/首个相关项排名相对原图和 Pillow 均未改变。完整 Dockerfile 因 Docker Hub
  manifest `EOF` 无法重建，只能用本地缓存和离线 module cache；这既不是 native 双架构，也不是发布
  镜像证据。详见 [`semantic-vips-input-2026-08-26.md`](../evidence/int-001/semantic-vips-input-2026-08-26.md)。
- 通过同 digest mirror 缓存解除 Node manifest 下载阻断后，未修改 Dockerfile 的原生 Linux/arm64
  `make test-libvips` 完整构建/测试通过。同一生产 adapter 在 arm64 native 生成的 10 个 WebP 与
  amd64 QEMU 逐文件 byte-identical；这补齐 transform 的 arm64 原生开发证据，但原生 amd64、ONNX
  embedding 容差和完整进程仍 Pending。
- 固定 Google source revision、源权重 hash、PyTorch/Transformers/Optimum ONNX/Accelerate/ONNX 版本、
  opset 18 与 224×224/length-64 后，SigLIP 2 自导出三次产生相同的 1,501,208,026-byte ONNX 与参考
  fixture。macOS arm64 ORT 1.29.0 和原生 Linux/arm64、4 CPU/4 GiB/no-network ORT 1.28.0 的四个输出
  都匹配 PyTorch `1e-4` 容差，Linux 三次峰值约 2.19 GiB。该结果只关闭 arm64 provenance/兼容性子问题：
  tracer warning 限制为固定输入形状。后续三次 Linux 100-call smoke 拒绝四类错误输入，执行中取消后
  可恢复，warm-up 后 RSS 增长 9,740～10,508 KiB；但原生 amd64、统一 production runtime、真实检索
  质量、C/Go adapter、hostile graph、重复加载/长循环和完整进程容量仍 Pending。详见
  [`semantic-onnx-export-arm64-2026-08-26.md`](../evidence/int-001/semantic-onnx-export-arm64-2026-08-26.md)。
- 同一 ONNX 又读取生产 govips 原生 arm64 生成的 10 张公开 pilot WebP；macOS/Linux arm64 的
  float32 与 float16-stored 图片向量均给出相同 Top-3 记录，24 条查询无 ranking 变化。图片 P95 分别
  约 38.0/128.1 ms，24 条文本批次约 192.1/763.1 ms。但十图不能批准真实 float16 Recall，而且当前
  monolithic graph 每次同时执行图文 encoder，存在明显无效计算；必须比较 split image/text export，
  当前单体产物不进入生产候选。
- 固定 split graph 虽在 Linux arm64 将同形状 image/text P95 相对单体降低约 20.7%/78.1%，但 30 轮
  image session load/infer/close → text session load/infer/close 后，完整 cycle RSS 相对首轮增加
  213,004 KiB，超过 128 MiB smoke 门槛；峰值 2,305,768 KiB。曲线早期跃升后在高位波动，不足以声称
  无限 leak，但足以否定“close 后可按零残留预算”的调度假设。该 lifecycle 当前 No-Go。
- 2026-08-27 保留上述失败和 128 MiB 门槛，改为关闭 ORT CPU memory arena、保留 memory pattern。
  同一 30 轮的 cycle RSS 固定为 509,360 KiB、增长 0 KiB，峰值降至 2,046,084 KiB；三次 public pilot
  的 image/text P95 为 92.8～99.6/29.8～32.9 ms，float16 Top-3 全部不变。后续 arm64 spike 必须固定
  arena off；仍需 production adapter、amd64、完整进程和长周期，不能据此关闭 `INT-008/013`。
- 同日三次原生 Linux/arm64、4 CPU/4 GiB/no-network 组合负载持续执行真实 split image encoder，
  同时运行 synthetic 100k×512 float16 SQLite backfill/exact search/keyset browse/cancel/restart。容器峰值
  1.29～1.33 GB，search P95 165.443～186.532 ms，三次恢复均到 100,000 行；但 browse 相对基线退化
  5.32～12.74 倍，未通过原 20% Gate。绝对 proxy P95 仍小于 0.3 ms，说明必须由生产 HTTP/catalog
  测试裁决，不能放宽门槛。该实验也未编码 100k 真实图片、加载人脸 runtime 或覆盖 amd64，因此
  `INT-013` 继续未完成。详见
  [`semantic-vector-combined-load-linux-arm64-2026-08-27.md`](../evidence/int-001/semantic-vector-combined-load-linux-arm64-2026-08-27.md)。
- arena-off 双图轮换随后延长到 100 轮：cycle RSS 从 500,156 KiB 到 500,184 KiB，只增 28 KiB；但
  cgroup `memory.peak` 为 3,719,651,328 bytes，超过既定 3.2 GiB Gate。进程 HWM 2,037,064 KiB 与
  cgroup peak 的差异可能来自重复读取两份大图产生的 file cache，但本轮没有逐轮 `memory.stat`，不能
  把推测写成结论。故保留 arena-off allocator 子证据，同时拒绝反复完整模型 reload 的 lifecycle；
  下一轮必须比较有界常驻 session 或隔离 worker。详见
  [`semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.md`](../evidence/int-001/semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.md)。
- 双 session 常驻对照同样失败：100 轮推理稳定且输出正常，但 cgroup current 在周期末约 3.56 GB，
  peak 4,008,951,808 bytes；约 1.90 GB anon + 1.65 GB file。在 SQLite、HTTP、缩略图和人脸 runtime
  尚未加入前已超过 3.2 GiB 并几乎撞到 4 GiB hard limit，因此不接受该布局。当前 SigLIP 2 的 reload
  与 resident lifecycle 均无可接受闭包，下一步应验证更小模型或正式删除双 encoder 共存需求。详见
  [`semantic-onnx-resident-sessions-linux-arm64-2026-08-27.md`](../evidence/int-001/semantic-onnx-resident-sessions-linux-arm64-2026-08-27.md)。
- 较小 SigLIP 1 候选随后用同一固定形状/opset 18 split 两次导出，image/text graph 371.7/441.2 MB
  且 byte-identical，PyTorch 最大绝对误差约 `3.81e-6/6.91e-6`。生产 govips 10 图/24 查询在
  macOS/Linux arm64 Top-3 一致，float16 全部不变，但中文 Recall@1 仍为 0.917。双 session 100 轮
  cgroup peak 2.18 GB；叠加 synthetic 100k float16 三次 peak 2.364～2.370 GB、search/restart 通过，
  browse 相对退化仍达 4.76～8.70 倍。故它替代 SigLIP 2 成为资源优先候选，不是最终选型；1,000 图
  质量、production browse/full process、native amd64 与合规仍阻断。详见
  [`siglip1-split-combined-linux-arm64-2026-08-27.md`](../evidence/int-001/siglip1-split-combined-linux-arm64-2026-08-27.md)。
- 随后不再使用微秒级 browse proxy，而是交叉编译当前 `tests/performance` 并在原生 Linux/arm64
  4 CPU/4 GiB/no-network 跑真实 10k 目录/100k 文件 scanner/catalog/SQLite/storyboard admission。
  配对中 ordinary recursive browse P95 28.267→31.431 ms（+11.2%）、扫描期 read/search +7.2%/+1.6%，
  但 cgroup peak 1.604→3.590 GB，global search/storyboard browse +25.3%/+20.3%。两轮 search-keyset
  均超过既有 250 ms 预算，只有一组配对也不足以界定方差。它否定 proxy 数倍退化的代表性，同时明确
  full-process 内存仍失败；必须先量化/缩小模型再复测。详见
  [`siglip1-production-catalog-capacity-linux-arm64-2026-08-27.md`](../evidence/int-001/siglip1-production-catalog-capacity-linux-arm64-2026-08-27.md)。
- dynamic-QInt8 MatMul/Gemm 权重量化两次 byte-identical，将 image/text graph 降至 95.8/184.8 MB，
  native Linux cgroup peak 约 811 MB且推理更快；但 macOS/Linux 只有 8/24 Top-3 一致，Linux 中文
  Recall@1/3 从 0.917/1.0 跌至 0.25/0.5。不同 ORT minor 是混杂因素而不是放行理由；该配置已在容量
  测试前拒绝。详见
  [`siglip1-dynamic-int8-rejection-2026-08-27.md`](../evidence/int-001/siglip1-dynamic-int8-rejection-2026-08-27.md)。
- float16-internal/float32-I/O 转换的 image graph 初次因插入 Cast 后拓扑无效被 checker 拒绝；加入
  runtime 自带确定性 topological sort 后两次转换 byte-identical，image/text 为 186.0/220.7 MB。
  macOS/Linux 与 float32 的 24/24 pilot Top-3 均一致。双 session 100 轮 peak 1.614 GB；生产
  10k/100k capacity peak 2.906 GB、recursive browse +14.5%，但 global search 单次 +41.6%，且
  baseline/AI keyset 都失败。它成为资源优先候选，不是选型完成。详见
  [`siglip1-float16-production-capacity-linux-arm64-2026-08-27.md`](../evidence/int-001/siglip1-float16-production-capacity-linux-arm64-2026-08-27.md)。
- 追加两组 fresh no-AI/float16-AI production capacity 后，三次 AI cgroup peak 为 2.860～2.951 GB，
  ordinary recursive browse 退化 6.0%～14.5%、中位 +10.4%；首轮 global search +41.6% 在后两轮为
  -4.5%/-3.3%，未复现。storyboard browse 一轮 +20.34% 仍超限，且六次 baseline/AI search-keyset
  P95 352～383 ms，全部失败既有 250 ms 绝对预算。详见
  [`siglip1-float16-production-capacity-repeated-linux-arm64-2026-08-27.md`](../evidence/int-001/siglip1-float16-production-capacity-repeated-linux-arm64-2026-08-27.md)。
- 对无 AI 的生产 10k/100k 基线追加 component timing 后，两页 search-keyset P95 为 358.623 ms；
  第一/二页服务调用为 167.755/190.937 ms，repository count 为 66.987 ms，list 为
  106.750/130.316 ms。既有 250 ms 门槛保持不变。结果证明重复 count 和列表查询都占实质成本，
  但尚未用 query plan 证明 broad FTS、派生目录排序或 keyset predicate 中哪一项主导；该既有搜索
  债务必须在独立维护 Gate 处理，不能归因给模型或借 `INT-S0` 偷改生产合同。详见
  [`catalog-search-keyset-components-linux-arm64-2026-08-27.md`](../evidence/int-001/catalog-search-keyset-components-linux-arm64-2026-08-27.md)。
- 随后的 benchmark-only query-plan capture 证明 count/list 均以 FTS 虚表候选扫描和 asset rowid 回表
  开始，first/second list 均以 `USE TEMP B-TREE FOR ORDER BY` 结束；第二页 keyset 没改变执行形态，
  既有 folder/name expression index 没有产生搜索结果顺序。这关闭了“根因尚无 plan”的证据缺口，
  随后 index-ordered membership 广匹配候选在同档原生 arm64 的前后两页各 101 个 ID 与生产结果
  完全一致，扩展运行 ID-selection P95 33.990/32.451 ms，完整装配 33.668 ms，稀疏 `asset-099`
  为 110.061 ms 且结果一致，计划均无临时排序。后续 arm64 100k correctness matrix 又覆盖 26 个
  首页面与 22 个第二页 scope/filter/sort/order/cursor 组合，ID 顺序全部一致，候选单页最坏约 203 ms；
  三次现有 repository broad + modified-window 首/次页约 9.44～19.00/9.55～19.00 s，plan 证明
  `assets_modified` 范围扫描驱动、逐行 FTS 探测与临时排序。80k image/10k video/10k animated 及
  image+video 组合已覆盖；
  另一个两个不重叠库各 5k/50k 的 global matrix 加入不同 image/video/animated 比例、image+video、
  选择性日期窗口和约 2% 稀疏词后，11 个首页/11 个第二页全部保持顺序，候选最坏约 83 ms。合成
  跨库 mixed-media/date/sparse 正确性子证明已关闭；其他组合 kind、真实分布/选择性、矩阵
  hydration/P95、混合阈值和 amd64 未证，因此仍不关闭
  性能 Gate；后续须在独立维护
  slice 比较该候选、bounded/materialized candidate 与 count revision/首屏语义，并用原预算复测。详见
  [`catalog-search-query-plan-linux-arm64-2026-08-27.md`](../evidence/int-001/catalog-search-query-plan-linux-arm64-2026-08-27.md)。
  当前最慢 kind、递归名称排序和 broad modified-window 三个代表场景随后各采样 20 次，首页/第二页
  P95 最坏 174.591/135.115 ms，均低于 250 ms；其余矩阵、跨库分布和完整 hydration 仍无重复 P95。
- 同一 float16 split 在 macOS/arm64 ORT 1.29 与原生 Linux/arm64 ORT 1.28 拒绝空、随机、截断模型，
  以及 image/text 各四类错误 shape/dtype；所有损坏模型子进程均在 15 秒内正常退出，错误后正常推理
  恢复。它只关闭基础失败关闭子范围，不覆盖 protobuf-valid hostile graph、native amd64、C/Go adapter
  或 adapter 级取消。详见
  [`siglip1-float16-failure-closed-arm64-2026-08-27.md`](../evidence/int-001/siglip1-float16-failure-closed-arm64-2026-08-27.md)。
- parser-valid hostile fixture 在两套 arm64 runtime 均让同目录 external-data 控制图正常推理，同时拒绝
  `../` external-data、未知算子和循环图，所有子进程无超时/信号退出。该结果只作纵深防御；生产包
  该能力没有产品必要性却扩大路径/TOCTOU 面，因此首版候选收敛为 graph 内嵌全部 initializer，发行
  校验拒绝全部 external-data，runtime 结果只作纵深防御。详见
  [`onnx-hostile-graph-arm64-2026-08-27.md`](../evidence/int-001/onnx-hostile-graph-arm64-2026-08-27.md)。
- 固定请求 6 GB 输出的 parser-valid graph 在原生 Linux/arm64、1/1.5/2 GiB child address-space
  上限下均返回 `RuntimeException`，同期控制图正常，且无 timeout/signal；512 MiB 因控制也失败而排除。
  child RLIMIT 不是 Go 单进程生产方案，正式边界仍是批准图、固定 shape 和 exact hash。详见
  [`onnx-oversized-allocation-linux-arm64-2026-08-27.md`](../evidence/int-001/onnx-oversized-allocation-linux-arm64-2026-08-27.md)。
- 官方 ORT 1.28.0 Linux/aarch64 C 包固定 archive/lib/header/license/notices hash；独立 Go/cgo harness
  使用 C 分配 tensor、arena off，在 100 轮 cancel→recovery 中取消 P95 6.56 ms、RSS 增长 17,404 KiB，
  30 轮 race build 无 Go race 报告且 RSS 增长 80,364 KiB。raw ORT error 含模型路径，harness 只输出
  稳定码。它不进入生产依赖图，仍缺 native amd64、最终镜像 ABI/扫描和 context/admission owner。详见
  [`onnx-go-capi-linux-arm64-2026-08-27.md`](../evidence/int-001/onnx-go-capi-linux-arm64-2026-08-27.md)。
- 隔离 distroless arm64 final-stage 首次因遗漏 ORT 的 `.so.1` SONAME 链接而以 127 失败；补齐归档内
  symlink 后，在 non-root、read-only root、no-network、cap-drop、no-new-privileges 下完成 100 轮
  cancel→recovery。最新 `cc-debian13` 闭包的 Grype 扫描仍为 1 Critical/9 High；改用固定
  `base-nossl` 并只复制带 Debian metadata/license 的 C++ 运行库后，同一运行测试通过且移除 7 个
  OpenSSL High，但 glibc 仍有 1 Critical/2 High。Debian 将三项标记为 trixie vulnerable、minor/
  no-dsa，这只是 VEX 输入而不是自动豁免；CycloneDX 未将裸 ORT `.so` 识别为 package component，
  现已补一个通过官方 1.6 Schema 结构校验、composition=incomplete 的 arm64 显式组件，绑定
  library/archive/source/license/notices hash。因此 arm64 ABI/最小闭包和组件身份可行，但发布安全、
  最终镜像 SBOM 合并、native amd64 与生产 composition 继续阻断。详见
  [`onnx-distroless-runtime-linux-arm64-2026-08-27.md`](../evidence/int-001/onnx-distroless-runtime-linux-arm64-2026-08-27.md)。
- 合规复核固定了 ORT archive/MIT license/6,121 行 notices、SigLIP 1、YuNet 与 SFace 的精确
  revision/hash/上游许可记录，但没有将其误写成批准。Docker Scout 因本机未登录而未产生漏洞报告；
  固定 Grype 0.116.1 后续扫裸 ORT 包虽为 0 match，也只识别 0 component，仍属 inconclusive。
  SFace 目录虽声明 Apache-2.0，精确权重的训练数据与商业/再分发澄清仍无已采信答复，因此 production
  hold。若无法消除该风险，必须换经审查的人脸 embedding 或正式移除/延期人脸 slice，不能用下载能力
  绕过。详见
  [`candidate-compliance-review-2026-08-27.md`](../evidence/int-001/candidate-compliance-review-2026-08-27.md)。
- S0 收口后的 S1 tokenizer 阻断验证没有重开模型探索：官方 SentencePiece 0.2.1 在原生 Linux/arm64
  通过窄 Go/cgo wrapper 读取固定 SigLIP `spiece.model`，中英文/Unicode 固定 token ID、63-piece 截断
  与长度 64 EOS/pad 通过。它证明提议 ADR-0014 的 arm64 技术路径可行，不证明 native amd64、供应链、
  hostile model/resource、FD 加载或端到端 embedding parity；ADR 与 `INT-203` 仍开放。详见
  [`sentencepiece-capi-linux-arm64-2026-08-27.md`](../evidence/int-001/sentencepiece-capi-linux-arm64-2026-08-27.md)。
- 后续 Linux/arm64 生命周期复核补齐 open-FD 同步加载、并发/关闭语义、畸形与有界模型拒绝、预取消，
  并在 100 次 load/close 后观测到 7,602,176 bytes retained RSS 增量，低于 64 MiB spike 门槛。
  SentencePiece 无 mid-call interruption，且 native amd64、长期 soak、完整 reference、text embedding parity、
  format v2 与最终供应链仍缺；因此 ADR 与 `INT-203` 状态不变。详见
  [`sentencepiece-capi-lifecycle-linux-arm64-2026-08-28.md`](../evidence/int-001/sentencepiece-capi-lifecycle-linux-arm64-2026-08-28.md)。
- 同一 tokenizer suite 又在 QEMU 模拟 Linux/amd64 userspace/ABI 编译并通过，100 次 load/close retained
  RSS 增量为 7,655,424 bytes。它是架构 ABI/行为 smoke，不是原生 amd64 性能或稳定性证据，故不关闭
  native gate。详见
  [`sentencepiece-capi-emulated-amd64-2026-08-28.md`](../evidence/int-001/sentencepiece-capi-emulated-amd64-2026-08-28.md)。
- 固定 Python 3.12.13、Transformers 4.56.2 与 SentencePiece 0.2.1 又生成 31 组完整 64-ID reference
  fixture。首轮比较抓到 Go whole-string lowercase 对 Greek final sigma 的真实偏差；改为与 Transformers
  non-greedy regex 一致的逐 code-point lowercase 后，Linux/arm64 全矩阵通过。literal special-token
  后续已冻结为大小写不敏感拒绝 `</s>`/`<unk>`，并在 snapshot/inference 前失败关闭；这仍不替代
  native amd64、text embedding 或供应链 Gate。详见
  [`sentencepiece-transformers-reference-linux-arm64-2026-08-28.md`](../evidence/int-001/sentencepiece-transformers-reference-linux-arm64-2026-08-28.md)。
- 本地保留的两份 SigLIP 1 split text graph 经 hash 复核均为 441,217,411 bytes / `16eef127...fd664`；
  固定 ORT 1.29/NumPy 2.5.2 对 31 组 token 输入生成的 768-D reference JSON 逐字节一致。Go/C ORT 1.28
  adapter parity 尚未运行，因此只关闭 reference construction。详见
  [`siglip1-text-embedding-reference-2026-08-28.md`](../evidence/int-001/siglip1-text-embedding-reference-2026-08-28.md)。
- 隔离 Go/cgo SentencePiece → Linux/arm64 ORT 1.28 text graph 链已消费该 reference：31×768 全部有限，
  cross-runtime max abs `1.811981201171875e-05`，通过 `1e-4` 门槛。它关闭 arm64 端到端数值路径，
  不批准 production FD composition、native amd64、active cancellation/lifecycle、format v2 或供应链。
  详见 [`siglip1-go-c-text-parity-linux-arm64-2026-08-28.md`](../evidence/int-001/siglip1-go-c-text-parity-linux-arm64-2026-08-28.md)。
- 同一 text harness 的 active-run cancel、取消后复用与 8 caller 串行并发通过。10 warm-up + 10 measured
  load/close 的后段 retained RSS 仅 +28,672 bytes，但 cold-to-stable 扩张为 355,897,344 bytes；结论是
  未见持续逐次泄漏，不是“无内存代价”，完整进程预算必须保留这约 356 MB warm cost。
- 模型包 format v2 已形成隔离 executable contract：三项严格文件角色和四项 pipeline contract ID，且
  v1/v2、generic tokenizer/SentencePiece 不会静默互认。该 matrix 通过但 production parser 未改变；
  ADR-0014 接受、内建 catalog 与 FD activation 证据仍是迁移前置条件。详见
  [`model-package-v2-contract-2026-08-28.md`](../evidence/int-001/model-package-v2-contract-2026-08-28.md)。
- 隔离下载状态机用 `httptest` 验证了 catalog-owned origin/ETag/size/hash、Range/If-Range 稳定及
  传输中取消后续传、变化对象拒绝、跨 origin 重定向拒绝、loopback 默认拒绝、配额、错 hash 不发布和
  no-replace；macOS/arm64 与原生 Linux/arm64 均通过，arm64 受限 tmpfs 还触发了真实 `ENOSPC`，
  candidate 未发布且 active 不变；真实子进程在 partial 后被 `SIGKILL`，新进程以同一 ETag/Range
  恢复并精确发布。catalog transport 只解析一次，特殊/私网/混合 answer 整体拒绝，后续拨号固定到
  已审核 IP 并禁用环境 proxy，两套 arm64 环境均通过；私有测试 CA 的真实 TLS 握手/SNI/未知 CA
  拒绝和固定地址集合内显式首地址失败回退也在两套 arm64 通过；阻塞 dial 又证明每地址 5 秒或更短
  外层 deadline 后会继续下一个已验证地址。它没有解决公网 CA/CDN/DNS TTL、外层 resolver/retry
  策略、其他发布阶段磁盘满/容器强杀、生产 key 托管/
  轮换/撤回与 checkpoint、active generation 或
  原生 Linux/amd64。Ed25519 primitive 已验证 exact payload/metadata、时间窗、rollback/equivocation
  拒绝，但不等于真实签名发布体系。隔离 SQLite registry 又验证 checkpoint 与 active pointer 单事务、
  故障整体回滚、幂等和重启持久化；原生 Linux anchored scan 接入后，精确 orphan 只登记不激活，
  missing/corrupt/restore 只改变 availability、不改变 active/checkpoint。真实只读 bind mount 又验证
  direct source kind 不可变、消失/精确 remount 恢复，不复制/删除/自动切换。它仍未定义生产
  migration/owner、direct 部署/API/UI、备份、配额或 unavailable 查询语义。详见
  [`model-download-failure-matrix-2026-08-26.md`](../evidence/int-001/model-download-failure-matrix-2026-08-26.md)。

## Go / No-Go

- 三个 workstream 均通过且无未处置严重风险：建议 S0 Go。
- 语义通过、人脸不通过：只提议图片语义/标签/视频范围，人脸留后续版本。
- 运行时或索引资源不通过：整个 feature No-Go，除非正式缩小容量/平台承诺。
- 权重许可不通过：对应能力 No-Go，不允许用下载脚本规避分发责任。
