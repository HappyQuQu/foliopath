# 独立 ML 代次兼容性子合同 v1

2026-09-15 兼容边界修补：旧 SigLIP 图片和文本工厂必须共同调用 semantic 的
`RequireLegacySigLIPGeneration`，仅允许 active、768 维且无空间绑定的旧代次。
非空绑定即拒绝，不以相同维度推断运行时兼容；这不改变新服务的准入、配置或激活 Gate。
回归包括策略状态/维度/绑定矩阵及两个实际工厂的已绑定拒绝、未绑定继续模型检查。

2026-09-14；POST-MVP-5 / CR-2026-030 / ADR-0019；FR-INT-017/019/020、
NFR-INT-007/009/010。Owner：semantic。本子片接受纯兼容性检查及合成测试，
不接受生产激活、数据库迁移或服务身份认证；app 组合仍保持关闭。

## 身份描述

一个向量空间由协议、模型标识、维度、运行时摘要、模型缓存清单摘要与预处理合同摘要
共同确定。所有摘要为小写 SHA-256 十六进制；描述版本固定为 1。
规范编码为固定字段结构 JSON，指纹为该编码的 SHA-256。字段顺序固定，禁止散列 map。
同名模型或相同维度均不是充分的兼容性证据；任一字段改变都需要不同代次。

描述只消费已由调用方核验的摘要，不接收主机路径，不读取文件，不执行网络请求。
缓存摘要应覆盖权重、tokenizer、配置；运行时摘要和预处理摘要必须覆盖真正执行的
镜像/算法合同，不能仅散列用户填写的模型名。验证这些事实是后续部署准入 owner 的责任。
计算指纹本身不证明远端确实运行该模型，也不构成签名或 attestation。

## 失败语义

无指纹的历史代次、格式错误指纹、描述无效、指纹不同都失败关闭，不得按维度回退。
错误不含输入、路径或配置。兼容检查只决定能否继续使用对应向量空间；不自行创建、
清除、激活代次。现有 768 维数据不回填伪造指纹，不改写既有 migration。

## 后续 Gate

2026-09-15 候选启动组合：部署 JSON 环境变量 FOLIOPATH_AI_DEPLOYMENT 提供固定 origin、
镜像独立预期身份、模型字节量与许可证说明；未知字段/缺失/重复变量拒绝，不在解析阶段联网。
app 组合一个预期身份绑定客户端，预测发送输入前握手且响应继续检查身份头。后台每 30 秒
核验服务、注册/更新可用性并复用原激活队列；失败不退出图库进程、不删除索引。
每个可用性 revision 最多三个持久幂等激活尝试，重启不重置；已有相同活动模型不重复激活。
配置存在时文本/图片/视频选择新 BoundEncoder，不回落旧空间；移除配置也不自动重激活旧模型。
预处理摘要固定为版本化算法合同（并非主镜像 OCI 证明），发行仍需固定主/AI 镜像及供应链验证。

2026-09-15 共享调度子片：aimodel 增加内部服务激活准入，复用原请求 hash、幂等表、信号、
单一 worker 和取消/失败结算。worker 在启动前可组合一次 DeploymentActivationRuntime，
service 分支不进入本地 manifest/file runtime；未组合则失败关闭。app 适配共享已绑定 HTTP
客户端，加载前重新核对服务身份，提交委托原服务空间事务。该适配已具备独立测试入口，
尚未由 run.go/部署配置创建；旧浏览器激活仍拒绝 service，不改变外部 API。

2026-09-15 服务激活事务子片：semantic.DeploymentActivationRepository 接受已核验的
SpaceIdentity 和既有 aimodel.ActivationCommit。SQLite 的原激活事务为唯一提交 owner；
不复制激活状态机。服务类型必须有匹配 model content_hash 的身份、相同维度、当前模型
可用性 revision、running 的 model_activate 操作。新代次在同一事务内 building → 绑定 → active，
旧 active 退役、活动指针和操作成功一并提交。任一步失败整体回滚，保留旧向量。
旧文件提交接口拒绝 service；服务提交接口拒绝文件模型。绑定复用原有事务内 helper，
不在历史代次补绑。这不开放旧浏览器激活接口，也不代替部署身份核验；app 仍未调用。

2026-09-15 服务模型存储子片（ADR-0020 / CR-2026-030）：ai_models 增加 service 类型，
不将远端模型描述伪装为 direct 目录。保留已有模型字段，package content hash 与
source identity 均为已核验空间指纹，package size 为部署提供的固定模型字节量而非本机占用。
只允许语义用途、available/unavailable；普通安装入口不允许注册 service。旧文件可用性
刷新跳过 service，旧激活入口拒绝 service。无浏览器注册/地址输入，独立内部注册由 aimodel
拥有，必须在部署核验后调用；本子片不连接 app、不激活。migration 39 沿用 migration 35
的关闭外键/legacy-alter/事务表重建模式，保留全部既有列与引用；存在 service 数据时降级
因约束失败而拒绝，不删除服务模型或相关索引以求降级成功。恢复旧版本采用升级前备份。

2026-09-15 视频流子片：VideoProcessor 复用同一个完整 storyboard 计划循环，
新增 NewStreamVideoProcessor 消费 StreamImageEncoder/BoundEncoder；旧构造器通过
既有 legacy bridge 保持行为。逐帧失败/取消丢弃完整计划并关闭所有流，持久化事务
仍由既有仓储/队列拥有，不新增 worker。此片不激活新代次，也不修改 app 组合。

### 查询/图片共同准入子片

2026-09-15 回填接口子片：BackfillProcessor 内部统一 stream 编码接口；旧构造器使用
SigLIP bridge 保留预处理语义，新构造器消费 BoundEncoder。任务循环、分页、源指纹、
流关闭、进度与事务所有权不变，不创建第二个队列/处理器。仅独立 SQLite 组合验证，
app 仍使用旧构造器。提交继续复用事务内的代次可写状态/维度检查。

semantic BoundEncoder 组合代次仓储、编码图片预处理和窄传输接口；部署方传入已经核验的
SpaceIdentity（本构造器不认证部署）。文本只接受 active，图片接受 building/ready/active。
推理前后均核对代次 ID、状态、维度与空间指纹；图片在预处理后、发送前也核对一次。
退役/删除/空间改变时丢弃结果，不返回部分向量。最终提交仍由存储事务检查代次状态，
前后读取不能替代提交原子性。复用 TextEncoder；新图片方法接受 stream 而非旧 SigLIP tensor。
只允许独立组合测试；尚不更换 app/旧 processor、激活配置或 worker。

### 持久化子片 v1（2026-09-14）

CR-2026-030 / ADR-0019 接受 migration 38 的附加向量空间绑定表及 SQLite 适配，
不接受自动激活或 app 组合。semantic 拥有绑定接口；SQLite 通过既有 write transaction
只为 building、维度相同、未有图片/视频/标签 embedding 的代次写入一次指纹；相同绑定可幂等重试，
不同绑定失败关闭。绑定记录不可 UPDATE，删除代次级联删除。旧代次不填充指纹，读取空值。
这不验证模型来源或部署身份；调用方仍须通过部署准入，且不得先写向量再绑定。
历史 active/ready 代次不允许补绑。绑定后模型、维度、变换/索引版本由数据库禁止原地修改。
此片不把绑定查询接到旧 ONNX runtime。

部署缓存清单与镜像事实核验、描述与 HTTP 实例绑定、原图安全编码、索引持久化迁移、
查询/回填共同准入、重启/切换集成仍未完成。人脸有独立代次，不借用语义身份。
