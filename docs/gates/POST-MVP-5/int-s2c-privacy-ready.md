# INT-S2C：人脸隐私与模型准入

- 日期：2026-08-29
- 目标版本：[POST-MVP-5 revision 2](../../releases/POST-MVP-5-scope-r2.md)
- 范围：E 匿名人脸聚类与管理员人物库
- 当前判断：**Backend Ready / Release No-Go**
- Contract input：[INT-S1R2 Contract Ready](int-s1r2-contract-ready.md)（Go）

2026-08-31 产品用户决定当前执行顺序先跳过人脸测试。该决定把 S2C 的实现与测试暂缓，不删除 E、
不降低本 Gate，也不允许用“未执行测试”宣称完成；工程继续推进非人脸 S2A/S2B 和发布证据，直到
下列外部准入输入到位后再恢复 S2C。

同日产品用户随后明确授权对一组本机网络图片做非公开、非训练、非模型发布的功能测试。隔离测试已证明
macOS/arm64 上候选 YuNet → align → SFace 能对普通图片产生有限 embedding，且未复制原图或持久化
人脸派生数据；见[功能冒烟记录](../../evidence/int-001/face-functional-local-arm64-2026-08-31.md)。该有限授权
只恢复隔离功能验证，不是 face-level ground truth、privacy/legal owner 签署、模型发布准入或 production
S2C 开工许可，本 Gate 仍为 No-Go。

2026-08-31 产品用户随后要求继续到全部 S2 完成后再统一汇报，明确授权 S2C 后端实现。该授权允许在
production package 中实现默认不可达、无 UI、无模型自动获取的 fail-closed capability、repository、worker
和 HTTP adapter；不授权发布、默认启用、身份推断或跳过质量/双架构/供应链验收。下列输入因此从“代码
开工前置”调整为“Release Gate 前置”，缺失时 production composition 必须保持不可用。

## Release Gate 前必须同时具备

1. 隐私 owner 接受默认关闭、告知、数据分类、访问、删除、备份/恢复与 incident fallback；
2. 合法真实 face ground truth 的来源、授权、purpose、访问、保留、删除 owner 与删除证明；
3. 可商业分发的 detector/embedding/runtime 候选及精确 hash/license/native 双架构来源；
4. 冻结评测协议能证明 anonymous core precision ≥99.5%，否则合同明确降为逐脸/小组；
5. 数据不进入 git、普通 CI artifact、日志、诊断或公共对象存储的可执行控制。

在这些输入签署前允许 backend-first 实现和测试，但不得注册可用 runtime、启用默认设置、创建消费者 UI
或解释为隐私、质量或模型准入证据。

失败回退：保持 E 全局不可用并不创建/保留新的 biometric 派生数据；核心浏览、人工标签、故事板以及
已独立通过 Gate 的其他切片不受影响。

## 2026-08-31 后端实现复审（历史）

当前仍为 **Implementation Authorized / Release No-Go**，但以下内部实现证据已完成：

- 有界 detector/quality/embedding processor 合同、float16 observation repository、零脸完成 marker、
  source fingerprint 失效和 generation/library 隔离；
- restart-safe analysis/clear durable jobs、lease、协作取消、重试耗尽、checkpoint 和最后可靠状态保留；
- 分析任务在每个候选前复核 cancellation 与 library state；offline 立即停止后续 admission，以稳定错误终止，
  保留已安全提交批次和全部最后可靠派生/人工状态，不重建空 cluster；staged cluster 激活再次绑定
  ready/enabled/active generation，offline/disable/换代竞争保留旧 active build，聚类后的 disable 收敛为
  cancelled 而非 succeeded；
- analyzer 报告 runtime unavailable 时按代次级 `model_unavailable` 立即终止，不推进 checkpoint/覆盖率、
  不继续逐资产失败、不创建部分 cluster build，并将库收敛到 `awaiting_model`；
- 确定性 core/edge 聚类、cannot-link 优先、staged build 原子切换和本机 100k 聚类容量基线；
- core 使用 smallest-ID anchor coherence，拒绝 `A≈B≈C` 在 `A` 与 `C` 未达阈值时传递形成同一 core；
  bridge face 最多作为 edge。该规则由授权 `coser` 扩大样本暴露的 834-member 跨来源巨簇触发，修复后
  同一 986 候选重算为 316 个 pending candidate 且 100k×512 指纹/容量保持；这不是身份质量通过证据；
- people/revision/重名/空人物、assign/exclude/cannot-link/merge/split、audit、alias、幂等和 guarded undo；
- people/assets、active cluster detail 和单资产多人脸安全投影；只返回 opaque ID、状态/角色和整数百分比粗略
  区域，不返回 embedding、crop、精确 bbox、模型分数或内部路径；
- 人物跨库资产页绑定所有关联库的状态 revision；offline/not-ready 不静默过滤为不完整空页，恢复后旧 cursor
  按 stale 失败，最后可靠人物/anchor 状态不删除；
- derived/manual clear 分离、真实 library removal cascade、SQLite 一致备份恢复、孤立人物保留及合成原媒体
  sentinel SHA-256/mtime 不变；
- HTTP read/mutation adapter 与 OpenAPI/生成客户端已完成，但 production composition 仍无 face route、runtime
  或 session 注册，继续失败关闭。
- 架构 fitness test 锁定 face HTTP 禁止字段、capability/SQLite 无直接日志及现有诊断/系统日志不消费 face
  状态；未来 production diagnostics/support package 接入必须显式扩展并重新通过隐私 Gate。

新增 `make verify-intelligent-media-face-quality`。它要求书面授权的 schema v2 biometric ground truth、至少
50 个 opaque identity 且每个至少 20 张图片、privacy/compliance/product/ML/QA 引用，并从逐项结果重算
detection recall、verification recall/FPR、core/edge precision、五类偏差切片与 Wilson 95% 区间。它不接受
公开许可替代生物特征授权，不信任输入 pass 字段，anonymous core precision 低于 99.5% 时失败。最终
`verify-intelligent-media-s2-evidence` 现同时绑定 C/D quality、E face quality、严格双架构和供应链 summary。

本轮未提供上述真实 face ground truth、最终审核模型包、native Linux amd64/arm64 成对运行、最终联合容量或
privacy/compliance 签署，因此不得把内部测试和本机 macOS/arm64 功能数据解释为 Release Gate Go。

## 2026-09-01 Backend Ready 签署

[CR-2026-022](../../changes/CR-2026-022-s2-backend-release-gate-separation.md) 将 formal biometric quality、
native paired final image、最终供应链和 privacy/compliance/security/release 批准恢复到 S4。S2C 已完成
production package/generation、repository/worker/API、人物与人工约束、清除/备份、source/offline/cancel/
kill/retry failure matrix、100k×512 容量，以及用户授权 `coser` 的 2,347 图/1,515 face 私有功能验证。

安全复审确认：默认关闭且 production face route/runtime 不注册；不推断身份或敏感属性；原媒体只读；
crop/vector/path/name 不进入 API/日志/诊断；私有复核输出不进入 Git/CI；`groupAssignmentAllowed=false`。
据此 `INT-241～251` 签署 **Backend Ready**。50×20 identity、五维偏差、99.5% core precision、最终模型
许可和 owner 发布批准仍缺，因此本 Gate 的发布侧继续 **Release No-Go**。
同一 closeout 已成功执行完整仓库验证、100k×512 capacity 和私有复核工具 14 个 Python 回归；命令和
结果集中记录于 [S2 最终完成审计](int-s2-final-blocker-audit-2026-09-01.md)。

2026-09-01 对授权 `coser` 根扩大到 1,547 张只读样本后，单链 core 语义经少量 bridge face 把八个来源
传递串成 834-member 巨簇。production 和私有复核工具已统一改为 smallest-ID anchor coherence；同一
986 候选重算为 316 个簇、9 个 `≥20` pending candidate，巨簇消失。定向 bridge 回归连续 20 次、race、
100k×512 容量及整仓验证通过。该修复关闭算法传播缺口，但 pending candidate 不是身份 ground truth，
不改变本 Gate。见[core bridge 防护](../../changes/FIX-2026-09-01-face-core-bridge-prevention.md)。

同一授权根随后把每来源采样上限提高到 500：2,347 张全部解码，1,496 张图产生 1,515 个唯一、有限的
512 维候选。anchor-coherent 阈值从 0.60 提高到 0.80 时，`≥20` 候选簇从 17 降到 0；0.70 仅剩两个，
且逐脸拼图复核仍发现其中一个含可疑异人。该运行证明 `coser` 足以做只读 pipeline、阈值退化和错误合并
回归，但不足以产生 50 个已确认身份、每身份 20 张及五维偏差切片，故不改变本 Gate。

2026-09-01 又筛选了 fal 官方 AuraFace v1 `glintr100.onnx`：固定仓库 revision、260,694,151-byte 工件及
SHA-256，官方模型卡/Apache-2.0 许可明确称其面向商业场景并使用 commercial dataset；精确图在 ORT 1.28
Darwin/arm64 直接运行，并在授权 135 图中产生 79/79 有限 embedding。相比 held SFace/InsightFace-derived
候选，这提供了更清晰的 provider permission statement，但模型卡未公开训练数据集身份、许可方、同意与
删除链，也没有 FolioPath compliance 签署。因此它仅进入候选复审，不进入 production catalog，不解除
真实 ground truth、99.5% core precision、双架构、联合容量或四方签署要求；Gate 判断不变。证据见
[AuraFace v1 candidate](../../evidence/int-001/auraface-v1-candidate-darwin-arm64-2026-09-01.md)。

同日 production `internal/inference/onnx` 增加严格、独立的 `face_detector` / `face_embedder` session，并在
Linux/arm64 用精确 YuNet/AuraFace 候选通过原生 C API 编译、12-output detector 与 embedding 图合同校验、
有界 detector decode/NMS 和单次 512 维有限非零输出。随后冻结 libvips 与 production adapter 在固定公开
JPEG 上完成 decode/resize、BGR detector input、五点 alignment、AuraFace preprocess 与 candidate embedding。
模型和输入仍只读外置，且没有注册 production catalog、generation activation、route 或 worker。该证据只关闭
候选原生 pipeline 边界
子项，不替代 native amd64、正式质量/偏差、联合容量、训练数据/许可复核与四方签署；Gate 判断不变。见
[Linux/arm64 production-boundary smoke](../../evidence/int-001/auraface-production-boundary-linux-arm64-2026-09-01.md)。

同一边界随后以固定 x64 ORT archive 和仓库固定源码 libvips 在 Docker Desktop foreign-architecture
Linux/amd64 目标通过 direct detector/embedder 与完整 pipeline 预检；运行期断网、rootfs/模型/输入只读。
宿主仍为 arm64，因此这是 compile/ABI/functional preflight，不是 native amd64 证据，不能进入最终 paired
summary 或改变 Gate。见
[emulated Linux/amd64 preflight](../../evidence/int-001/auraface-production-boundary-emulated-amd64-2026-09-01.md)。

2026-09-02 将同一候选边界固化进手动原生双架构 workflow：Linux runner 必须同时匹配 machine、Go arch
和 Docker arch，固定 ORT/YuNet/AuraFace/公开 fixture SHA 后在断网只读有界容器运行。paired verifier 严格
读取每侧 `face-candidate.json`，要求同 source/model/runtime commit/fixture/candidate count，并拒绝任何
production/quality/compliance approval 声明。当前工作树尚未提交到可调度入口，也没有真实 native run；
即使候选步骤未来通过，仍不替代最终模型、真实质量/偏差、联合容量和 owner 签署，故 Gate 不变。见
[人脸候选原生工作流预检](../../changes/FIX-2026-09-02-face-native-workflow-preflight.md)。

100k 聚类基线随后从 32 维升级为候选所需的 512 维，并暴露/修复一个 LSH 桶边界漏比较导致 edge 扫描
退化的缺陷；随后还关闭了 edge 全簇扫描与 edge→edge 误挂。修复后本机 Darwin/arm64 与 4 CPU/4 GiB
Linux/arm64 容器都在约 7.2 秒内连续完成 100,000 paired-core 成员和 100,000 all-singleton 成员两种
极端，非向量结果指纹一致；原生 workflow 现要求两个 runner 都产出严格的
`face-capacity.json`。这是合成算法容量证据，尚无已运行的 native amd64 配对，也不覆盖真实质量、SQLite/
HTTP/浏览并发或最终联合负载，因此 Gate 不变。见
[容量修复记录](../../changes/FIX-2026-09-01-face-lsh-bucket-boundary-capacity.md)。

模型生命周期审计还确认当前 `ai_models`/activation commit 只拥有 semantic purpose/singleton，不能安全创建
face generation。为避免以测试 seed、两次组件激活或误退休 semantic generation 补洞，新增
[ADR-0015](../../adr/0015-face-model-package-and-generation-activation.md) 已接受 fail-closed 后端实现：一个审核组合包、
一次独立 face generation 原子激活、各库完整重建后再切换 cluster。production parser、schema 与 activation
transaction 已实现并通过回归；内建 catalog 和 application composition 仍无 face entry/runtime，Gate 判断不变。

ADR-0015 的隔离 precursor 与 production `internal/aimodel` 都执行 face format v3 hostile-manifest 规则，绑定
detector、embedder、governed threshold-profile 和精确 transform IDs，并拒绝 semantic format 混淆或 manifest
自我授权。架构 fitness 继续禁止 production 导入 spike，并要求 Release No-Go。独立 face generation 事务不会
退休 semantic generation，也不会在新代激活时提前切换各库 cluster。它不提供最终模型、阈值、质量、许可、
双架构或 owner 签署。
