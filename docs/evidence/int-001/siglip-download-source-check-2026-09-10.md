# SigLIP 1 下载源核对（2026-09-10）

范围：POST-MVP-5 / CR-2026-027；核对首个预设下载包，不宣称下载/激活闭环完成。

## 已执行

- 本机 `go test ./tests/integration -run TestAIModelDownload -count=1` 通过。
  可用磁盘空间约 92 GiB，先前约 21 GiB 时触发的模型存储 10% 安全余量限制不再触发；
  未修改余量策略、未删除数据。测试仍使用合成文件，不是真实权重或 native inference 验证。
- 读取 HF 的 Immich SigLIP 1 仓库元数据，固定当前 revision：
  `c3b037a9ef99ed5f116440790f1e098fb388db49`。
- 对固定 revision 的两个 ONNX 文件各读取末尾 16 KiB，检查 protobuf 输入输出描述。
  这不是整文件 hash 验证，也不能证明 ONNX 整图安全或推理输出正确。
- server222 用 `curl -IL --max-time 20` 连接该固定 visual URL，约 10 秒后连接超时。
  仅说明本次从该主机直连失败，不推断永久不可达或其他网络路径状态。

## 文件与接口差异

来源：[Immich SigLIP 1 固定仓库](https://huggingface.co/immich-app/ViT-B-16-SigLIP__webli/tree/c3b037a9ef99ed5f116440790f1e098fb388db49)。
以下大小与 SHA-256 来自 HF LFS 元数据，尚未下载全文件复核：

| 文件 | 大小（bytes） | SHA-256 |
| --- | ---: | --- |
| visual/model.onnx | 371674292 | 0981d03d194baa6153106d41b5a5ca73faa514e8d9a007a3332eecb63be2c74a |
| textual/model.onnx | 441295409 | 3983f96348daba5024f5e3085d2ea8cc67c11ce553e4061f193e8ea37aabd792 |

| 接口 | Immich 文件末尾描述 | FolioPath 当前 production 合同 |
| --- | --- | --- |
| 图像输入 | image / float32 / [1,3,224,224] | pixel_values / float32 / [1,3,224,224] |
| 文本输入 | text / int32 / [1,64] | input_ids / int64 / [1,64] |
| 输出 | embedding / float32 / [1,768] | image_embeds 或 text_embeds / float32 / [1,768] |

仅换下载地址或 manifest 不会让该包兼容；当前 activation 会拒绝它。文本目录提供 tokenizer.json，
也不能据此认为与当前 SentencePiece canonicalization 合同相同。不能只改 tensor 名字就宣称兼容。

## 未完成与下一步

1. 使用现有确定性导出流程准备符合当前合同的包，或接受并验证新的 Immich runtime/tokenizer 合同。
2. 若使用项目转换包，需明确固定 HTTPS 托管位置；公开上传权重/创建外部模型仓库须另获用户授权。
3. 验证 server222 可达的 HTTPS 来源；未绕过下载器的私网/重定向安全策略，未添加不明镜像。
4. 之后仍需整文件 hash、native runtime、生产下载/启用/合成搜图验证。

本轮没有发布模型、修改 server222 配置、切换镜像或激活模型。

## 后续推进：完整文件、候选转换与 native 验证

以上为首次读取片段时的记录。后续在本机下载了完整 image/text/spiece 文件，实测大小与上述固定
SHA-256 一致。Immich tokenizer.json 为 2,421,038 bytes，实测 SHA-256
`ec6fe74c37f2a65be3857ef5e8fbca7662ebb15fcfa569a845a59958db969f76`。
可复现脚本：`spikes/int001-ai/siglip_immich_probe.py`，依赖记录在同目录
`requirements-siglip-immich-probe.txt`。所有输入先验证大小与 hash，再调用 native parser。

原图保留在本机 `/tmp/foliopath-siglip-conversion.cxRbSk`，转换候选在其 `candidate` 子目录；
重复转换结果在 `repeated-candidate`。不在仓库提交权重，不修改源权重，不把候选加入 production Catalog。
图像仅重命名输入输出，文本增加 int64 → int32 Cast 并重命名外部输入输出；没有训练或改权重。

| 转换结果 | bytes | SHA-256 |
| --- | ---: | --- |
| image_encoder.onnx | 371674312 | 941bc2649234f3dbbc4ae3c77059bb9969f2aa47ca57d6b7a4e081bb05157a2e |
| text_encoder.onnx | 441295454 | 0b9cdf435028e23ddee07c0943c881ab3b017203a409c9918742f63a866ed91f |

- 两次独立输出的 hash 相同；macOS/arm64 ONNX Runtime 1.29.0 各一组固定合成输入，
  转换前后输出 max absolute error 均为 0。不是语义质量或全输入等价证明。
- Immich Unigram 的前 32,000 个 piece ID 与官方 SentencePiece 相同（JSON 另有 100 个条目）。
  官方 SentencePiece 0.2.1 对仓库 31 个 **已 canonicalize** 样例的 64 个 token ID 全部匹配。
  这不证明 Immich tokenizer 的归一化/特殊 token 与 production 在所有输入上等价。
- Linux/amd64 原生测试在 server222 的隔离临时容器中运行，限制 2 CPU / 2 GiB，无媒体挂载，
  使用既有开发 toolchain、只读源码/vendor/模型挂载。测试目录 `/tmp/foliopath-siglip-native.sg3NjD`。
- ORT 官方 v1.28.0 x64 archive 9,125,960 bytes，GitHub release digest 与实测 SHA-256 均为
  `a3e1b79d7bb1bf09696ce675f49e4064e6c81f6202b8225624fff0e93f8d6407`。
- 通过 `go test -p=2 -mod=vendor -tags="onnxruntime inferencekill" ./internal/inference/onnx
  -run TestNativeImageInferenceRecoversAfterProcessKill -count=1 -v`：真实图像推理进程中断后恢复。
- 通过 `go test -p=2 -mod=vendor -tags="onnxruntime inferencecandidate" ./internal/inference/onnx
  -run TestNativeSemanticConvertedCandidate -count=1 -v`：图像/文本各一个合成输入，非零有限 768 维输出。
- 通过 `go test -p=2 -mod=vendor -tags="onnxruntime sentencepiece inferencecandidate" ./internal/inference/onnx
  -run TestNativeSemanticCandidateRuntimeValidation -count=1 -v`：production `LoadAndValidate` 的联合加载、
  固定分词和文本推理校验。此开发容器使用 Debian `libsentencepiece-dev/libsentencepiece0=0.2.0-1+b3`，
  不等同于固定最终发行闭包或 native 全量分词验收；没有 activation worker 事务或 generation commit。
- 本地通过 `make fmt-check`、`make arch-check`、`git diff --check`、Python probe 实跑/语法编译，
  以及 `go test ./internal/inference/onnx ./internal/aimodel ./tests/integration
  -run 'TestDownload|TestAIModelDownload|TestOpenModelFiles|TestBuiltinPresets' -count=1`。

首次 native Docker 启动因只读 source 下缺 vendor 挂载点失败；在隔离临时目录创建挂载点后重跑通过。
SentencePiece 依赖只安装在自动移除的临时测试容器内；正式 FolioPath/Immich 容器和宿主机依赖未改。

## 网络复核与剩余边界

server222 系统 DNS（192.168.2.1）当次把 huggingface.co 解析为 69.171.228.74，与通过 DNS-over-HTTPS
读取的公共解析结果不同。普通访问超时；单次 curl --resolve 使用公共结果 3.164.110.128、保留 HTTPS
主机名和证书校验，仍返回 TLS connection reset。GitHub 首页当次返回 200。没有写 hosts、改 DNS、
设置代理或使用未知镜像；不能据此声称 server222 的 HF 大文件下载已修复。

**目前完成的是可重复的开发候选和 native runtime 检查，不是默认一键下载或实际搜图交付。**
仍缺：固定可达的转换包托管源、正式 runtime 闭包、完整安装/激活事务与实际搜图验证、质量/供应链审核。
公开托管尚未获得用户明确授权，因此未上传或创建外部仓库。此次所有测试没有访问真实媒体。

## 后续推进：真实候选的托管安装与激活事务（2026-09-10）

新增 opt-in 集成测试：
`tests/integration/semantic_candidate_activation_test.go` /
`TestSemanticCandidateManagedInstallAndNativeActivation`。
范围为 CR-2026-026/027、FR-INT-017/019/020 的候选验收增量，不新增生产行为或下载授权。
测试临时 Catalog 只承认上表三份固定文件；manifest 的合同声明是测试输入，不是正式质量批准。

在 server222 临时容器实跑 **PASS（12.14 秒）**：

- 通过真实 managed publisher 流式读取并校验三个候选文件，原子写入临时托管目录，注册 SQLite。
- 再次安装复用同一模型 ID，不重复注册。
- 真实 activation admission → worker → ONNX/SentencePiece LoadAndValidate → SQLite commit。
- 验证激活请求幂等重放、active model 指向正确模型，重开数据库仍保留。

命令核心：

```sh
go test -p=2 -mod=vendor -tags=onnxruntime,sentencepiece,inferencecandidate \
  ./tests/integration -run TestSemanticCandidateManagedInstallAndNativeActivation -count=1 -v
```

环境：既有 toolchain image `foliopath:ai-deploy-toolchain-20260909`，ORT 1.28.0，
Debian SentencePiece 0.2.0-1+b3；2 CPU / 2 GiB / 256 pids。
隔离源码目录 `/tmp/foliopath-native-activation.9h5fYI`，候选目录
`/tmp/foliopath-siglip-native.sg3NjD`，均只读挂载；应用数据库和模型输出由 t.TempDir 创建。
依赖只安装于自动移除的测试容器。正式 FolioPath/Immich 镜像、配置和数据库没有修改。
本地 `make fmt-check`、`make arch-check`、`go test ./tests/integration -count=1`、
`git diff --check` 通过；本机默认构建不执行带 native tags 的候选测试。

此项将“完整安装/激活事务”推进为**隔离候选集成已通过**，不等同于生产下载、正式镜像、
质量/供应链审核或实际搜图。固定可达的转换包 HTTPS 来源仍缺，未公开上传任何模型。
