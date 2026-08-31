# ADR-0014：SigLIP SentencePiece tokenizer runtime 与模型包 v2

## 状态

**提议（2026-08-27；2026-08-29 接受审计仍阻断）**。本记录未接受前，只允许隔离 spike、参考夹具和不依赖 SentencePiece 的纯 Go
canonicalization；不得把 SentencePiece 链入生产镜像、填写真实审核 catalog 或宣称模型可激活。

当前逐门槛判断见
[ADR-0014 接受审计](../gates/POST-MVP-5/adr-0014-acceptance-audit-2026-08-29.md)：
仅特殊 token 失败关闭合同已通过，其余门槛仍为 Partial/Blocked；不得把 QEMU amd64、incomplete SBOM
或动态 tripwire 提升为 native、final signed provenance 或 VEX。

## 背景

ADR-0013 选择 SigLIP 1 float16-internal split graph，但没有选择生产 tokenizer runtime。S1 模型包
format v1 示例只含 `image_encoder`、`text_encoder` 和一个 `tokenizer.json`。对固定 revision
`7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed` 的真实文件复核表明：

- `tokenizer.json` 是 32,000 词 Unigram tokenizer，启用 byte fallback、Metaspace 和预编译 Unicode
  normalization charmap；它不是可以用字符串切分近似替代的普通词表；
- 官方 slow tokenizer 使用 `spiece.model`，先执行 Unicode lowercase、ASCII punctuation 删除和 Unicode
  whitespace collapse，再由 SentencePiece 编码；EOS、padding 都是 ID 1，固定长度为 64；
- 对中文、英文、全角字符、组合重音、emoji、Unicode whitespace、空字符串和截断输入的固定参考 ID
  可以离线生成；直接用原始 `spiece.model` 编码 canonical text 与官方 slow tokenizer 在已测夹具一致；
- 完整复刻 `tokenizer.json` 的预编译 charmap 和 Unigram/byte-fallback 算法会形成项目自维护 tokenizer
  实现，正确性与长期供应链成本高于复用官方 SentencePiece。

当前生产代码已经拥有纯 Go canonicalization 和 embedding codec，但 activation 不能仅凭打开
`tokenizer.json`、检查文件大小或成功加载两张 ONNX graph 就发布 generation。

## 备选方案

1. **项目自行实现 `tokenizer.json`**：没有额外 native library，但必须重写预编译 Unicode normalizer、
   Unigram Viterbi、Metaspace、byte fallback、special token、截断和 padding。边缘 Unicode 静默偏差风险高。
2. **Hugging Face Rust `tokenizers` + C wrapper**：可直接消费 JSON，但增加 Rust/static-library 构建链和
   更大的未验证 native 闭包；上游没有 FolioPath 当前依赖的稳定官方 Go API。
3. **官方 Google SentencePiece C++ + FolioPath 窄 C wrapper**：需要一个额外 native library，但输入模型
   较小、接口窄、与官方 SigLIP slow tokenizer 路径一致。
4. **Python/Transformers sidecar 或 subprocess**：改变单进程/单容器运行闭包，启动与资源成本高，违反
   revision 1 边界。
5. **近似分词或只支持英文**：会让中文搜索质量与跨代 embedding 合同不可解释，直接违反冻结需求。

## 提议决定

若后续证据通过，采用方案 3：

- 使用固定版本的官方 SentencePiece C++，通过 `internal/inference/sentencepiece` 的最小 C wrapper 调用；
- adapter 只接收来源 owner 已打开的 Linux FD，通过 `/proc/self/fd/<fd>` 加载审核后的
  `spiece.model`，不接收任意路径、URL 或 package code；
- 生产实现使用显式 `sentencepiece` build tag；未带 tag、非 Linux、版本不匹配或 library 缺失时返回
  stable tokenizer-unavailable，核心浏览不受影响；
- canonicalization 由 `internal/semantic` 单一 owner 执行：最多 512 Unicode code points、按
  Transformers 4.56.2 non-greedy regex 行为逐 code point 做 Unicode lowercase（不得改成有上下文的
  whole-string lowercase）、移除 ASCII `string.punctuation`、Unicode whitespace collapse/trim；空结果拒绝；
  lowercase 后、标点移除前以大小写不敏感语义拒绝 literal registered control token `</s>` 与 `<unk>`，
  不允许用户文本注入 EOS/unknown 控制语义；
- SentencePiece 输出最多保留 63 个 token，末尾追加 EOS ID 1，再用 ID 1 padding 到 64；无
  attention mask 输入；
- 模型包升级为 format v2，角色严格为 `image_encoder`、`text_encoder`、`sentencepiece_model`，并固定
  `siglip-rgb224-bicubic-v1`、`siglip-transformers-4.56.2-v1`、
  `sentencepiece-32k-unk2-eos1-pad1-seq64-v1`、`siglip-768-l2-f16le-v1` 四个 contract ID。v1 不会被
  偷偷重新解释，format 不从文件名猜测；当前 catalog 为空，因此没有已发布包兼容负担；
- runtime 原始错误、query、token pieces、模型文件名和路径不得进入日志/API；只允许稳定错误码和计数；
- activation 必须同时通过 graph ABI、tokenizer metadata、固定双语/Unicode token ID fixture 和固定
  text embedding fixture，之后才允许 generation commit。

## 接受前门槛

- 固定 SentencePiece source revision/archive hash、许可证、notices、双架构 SBOM/provenance 和 final
  image SONAME/ABI；
- native linux/amd64 与 linux/arm64 的 token ID 全量 fixture 一致，覆盖中英文、组合字符、全角、emoji、
  Unicode whitespace、空/全标点、未知字节替代和 63-token 截断；
- malformed/truncated/oversized model、版本不匹配、FD 替换、取消、并发、反复 load/close 和 RSS 证据；
- 与固定 Python Transformers/SentencePiece 参考输出逐项一致；reference 工具只用于证据生成，不进入
  生产镜像；
- 用户输入中 literal registered special token `</s>` 与 `<unk>` 的拒绝合同必须由 unit/service/HTTP
  fixture 覆盖；不得让用户文本静默注入 tokenizer control token；
- 更新 package contract、catalog validator、deployment/security/testing/data-model 和供应链 Gate，并由
  架构、推理、安全/发布负责人接受本 ADR。

截至 2026-08-28，arm64 已完成固定 token/text 参考、FD/lifecycle、组合 distroless SONAME 和受限运行
子证据，format v2 也有隔离可执行 validator，固定 archive 内容已逐文件证明等价于官方 tag commit。
但 native amd64、exact distribution archive provenance、exact
final-image 漏洞处置、final signed SBOM/provenance、production composition 和负责人签署仍缺，故本
ADR 状态保持“提议”，生产不得链接 SentencePiece。隔离合并器已为当前 arm64 与 QEMU amd64 镜像生成
可重复但明确 incomplete 的 SBOM；固定 Grype 扫描也已确认两边相同的 glibc 1 Critical/2 High、无 fixed
version。扩展 ELF 闭包检查进一步发现两架构 libstdc++ 均直接导入受影响的 `ungetwc`；虽未证明应用
触发特殊编码条件，但不能签署“无直接导入”VEX。这些结果关闭工具链子证明，同时直接维持发布 No-Go，
不等于上述接受门槛完成。
双架构 `ungetwc` tripwire 随后证明固定 31-case 文本推理路径在 default locale 下不调用该函数，但这
不覆盖未来生产 composition、其他 locale/编码或 native amd64，故同样只作为安全评审输入。

同日的 QEMU amd64 组合镜像已通过 package/SONAME 和 31-case text parity 预检，说明同一 Dockerfile
可以消费固定 x64 ORT archive 与 amd64 distroless child manifests。该结果不改变上述 native amd64 门槛。

任一门槛失败时，不回退到近似 tokenizer、Python sidecar 或云 API；保持 catalog 为空并删除/延期
Slice B。

## 后果

正面：复用候选模型的官方 tokenizer 路径，中文与 Unicode 行为可由固定夹具验证；不需要在 Go 中维护
Unigram/normalizer 实现；仍保持单进程和无网络运行。

代价：最终镜像增加第二个 native ML 组件和 C++ 供应链；模型包 format 必须升级；模型 activation 和
release Gate 增加双架构 ABI、许可证、漏洞和资源证据。即使本 ADR 后续接受，模型权重合规和 1,000 图
质量 Gate 仍独立保持开放。
