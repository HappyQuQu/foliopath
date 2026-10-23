# ADR-0025：部署选择的多语言语义模型与能力身份隔离

日期：2026-10-23；状态：Accepted direction / implementation in progress。
范围：POST-MVP-5，FR-INT-009/017/019、NFR-INT-007；Owner inference/remoteml/app/semantic。
授权：用户要求使用 Immich 配置中的 `XLM-Roberta-Large-Vit-B-16Plus`，仅本地 Docker 验证/更新。

## 决策

保留两个部署单元，增加固定白名单模型 XLM-Roberta-Large-Vit-B-16Plus（640 维），
不把现有 ViT-B-32__openai 的 512 维结果重标为新结果；部署身份选择对应模型/维度，未知组合拒绝。
模型来源为 immich-app 同名仓库固定 revision f8857d09bd30188c40214543d2181f732e64eab4。
逐文件核对 LFS SHA256 或 Git blob OID，再生成部署固定 SHA256 清单；不是任意模型下载器。

现有 face/tag 的模型、算法与 worker 保持字节不变。新语义入口/worker 的兼容身份覆盖其自身源码、
固定模型及完整缓存清单；旧 face/tag 仍运行原入口/worker，其既有身份不因新语义配置改变而失效。
外层语义路由与旧协议复用同一 scheduler 的准入、26 秒总期限、kill/reap/清理，不另建并发队列。
旧 English 语义模式继续可构建，两个搜索空间绝不混用。固定镜像/受信部署仍是安全前提，不声称远程 attestation。

XLM ONNX 有外部权重文件，总缓存超过旧 2 GiB 合同；新语义私有快照上限明确为 4 GiB、512 文件，
每文件仍不超过 1 GiB。复用旧 FD-anchored snapshot 按 <=2 GiB 批次核验后合并私有快照，
不放宽旧能力缓存合同、不跟随源 symlink、不从网络补缺。
新模式使用专用可丢弃 disk scratch volume `/verified-cache`，不再将多 GiB 私有副本放在 tmpfs 与 ORT 常驻堆叠加；
它不是权重真相或任务持久状态。启动仅清理固定前缀的旧工作目录，未知内容/symlink 失败关闭，
请求/取消后沿用 scheduler 回收。Compose 覆盖仅换临时盘，不偷偷改变 RAM/swap；不得挂数据库/媒体为 scratch。
新模式需单独声明临时盘/资源需求并测量，
默认小模型预算不能被算作大模型通过证据，原 6010 的不限 RAM/swap 操作者选择保留。

## 切换与验收

先合同/适配测试、真实权重的合成 Docker 推理/索引/中文查询回归，再更新本地 6010。
更新前核对在途任务并停写完整备份；不丢弃人物、人工标签、删除抑制或原件。
只补齐新代次图片语义结果，旧代次保留；OCR、自动标签模型与显示语言不在本次范围。
图片索引完成/模型可用/中文检索质量分别记录，不能用运行中或少量合成图冒称全库质量通过。

## 替代方案

仅把查询翻译成英文：不能复用用户选定模型，且引入翻译依赖。
全局替换运行时身份：会让独立 face/tag 结果无故失效，拒绝。
任意模型名/动态下载：扩大信任边界，拒绝；本次仅固定白名单。
