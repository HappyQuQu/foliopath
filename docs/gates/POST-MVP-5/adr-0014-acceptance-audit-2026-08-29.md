# ADR-0014 接受审计（2026-08-29）

- 目标版本：`POST-MVP-5` revision 1 / Slice A+B
- 被审计决策：[ADR-0014](../../adr/0014-siglip-sentencepiece-tokenizer-runtime.md)
- 当前裁决：**保持提议 / Blocked**
- 生产授权：**无**；SentencePiece 不得进入 production import graph，package parser 保持 v1-only，
  reviewed catalog 保持为空，semantic search route 保持不注册

## 结论

方案 3（官方 SentencePiece C++ + 窄 Go/C wrapper）仍是现有证据支持的首选技术方向，但 ADR 自己冻结的
接受门槛尚未满足。当前证据不能把 QEMU amd64 当作 native amd64，不能把 incomplete SBOM 当作 final
signed provenance，也不能由动态 tripwire 自动代替 glibc VEX/修复基座决定。

本次审计不继续增加同类 arm64/QEMU spike。下一次裁决只由下表“精确剩余输入”发生变化触发。

## 接受门槛审计

| ADR 接受门槛 | 当前证据 | 判断 | 精确剩余输入 |
| --- | --- | --- | --- |
| 固定 SentencePiece source、license/notices、双架构 SBOM/provenance、final image SONAME/ABI | 固定 0.2.1 tag commit 与 258/258 blob/mode 相同；arm64 与 QEMU amd64 SONAME、component inventory 已有 | **Partial** |  reviewed distribution archive 或 deterministic reconstruction 的签名 provenance；final native 双架构完整 SBOM；ORT/SentencePiece advisory correlation；glibc 修复基座或正式 VEX |
| native arm64/amd64 token-ID 全量 fixture parity | native arm64 31-case 通过；QEMU amd64 31-case 通过 | **Blocked** | 同一 fixture 在 native Linux/amd64 运行并固定 runner、binary/image digest 与结果 |
| malformed/version/FD/cancel/concurrency/load-close/RSS | native arm64 已覆盖 malformed、FD、32 并发、pre-cancel、100 load/close 与 RSS；组合 text graph 另有有限 lifecycle | **Partial** | native amd64 同矩阵；production FD owner/adapter；明确无 mid-call hard-cancel 承诺；final process long-soak/RSS |
| 固定 Python Transformers/SentencePiece 参考逐项一致 | 31-case reference、native arm64 token/text 23,808 坐标通过；QEMU amd64 数值预检通过 | **Partial** | native amd64 token/text parity；production activation fixture 通过后再允许 generation commit |
| literal `</s>` / `<unk>` 在 unit/service/HTTP 前失败关闭 | canonical unit、search service dependency-short-circuit、HTTP stable error mapping 已覆盖 | **Passed** | 无；保持回归 |
| package contract、catalog/runtime、部署/安全/测试/数据及负责人接受 | format v2 隔离 validator 和设计文档存在；production parser/catalog/runtime 未迁移 | **Blocked** | 先完成上述供应链/native 输入；再由架构、推理、安全/发布负责人签署；同一变更迁移 production v2 parser、FD tokenizer activation 和 fitness check |

## 漏洞处置不能被降格

当前 arm64 与 QEMU amd64 runtime image 的 Grype 结果均为 1 Critical / 2 High，来自 glibc。外部 ELF
闭包未直接导入两类受影响函数，但两架构 `libstdc++` 都直接导入 `ungetwc`。31-case 动态 tripwire 未
触发只能证明该固定默认-locale 路径未调用它，不能覆盖 production HTTP composition、其他 locale、
其他 libstdc++ 路径或 native amd64。因此：

- 不允许写“无直接导入” blanket VEX；
- 不允许把“无 fixed version”解释成自动接受；
- 必须由安全 owner 接受精确 call-path/runtime-configuration VEX，或迁移到修复基座并重扫。

2026-08-29 的[官方状态与当前 distroless 复核](../../evidence/int-001/glibc-security-status-refresh-2026-08-29.md)
确认 Debian trixie 仍把镜像内的 `libc6 2.41-12+deb13u3` 标为三项 vulnerable；最新
`cc-debian13:nonroot` 的 arm64/amd64 子镜像也仍安装该版本。镜像标签变化没有解除 blocker，故未做
无安全收益的 pin churn，裁决保持 Blocked。

## 下一次裁决输入

至少出现以下任一实质输入后才重开本审计：

1. native Linux/amd64 runner 产出同 fixture、lifecycle、RSS 和完整 text parity；
2. final base digest 消除阻断漏洞，或安全 owner 签署精确 VEX；
3. final 双架构完整 SBOM/provenance 和模型/运行时 notices 获签署；
4. production format-v2 parser、FD tokenizer adapter 与 activation fixture 形成待审变更；
5. 架构、推理、安全/发布负责人给出接受或否决签署。

在此之前，ADR-0014 既不接受也不否决；Slice B 保持失败关闭，不以近似 tokenizer、Python sidecar、
云 API 或重复 arm64/QEMU 证据绕过。
