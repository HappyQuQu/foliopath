# 生产 govips adapter 语义输入对照 — 2026-08-26

状态：原生 Linux/arm64 + Linux/amd64 QEMU 开发证据；**不是原生双架构、ONNX 或完整 4 GiB 进程 Gate 证据**。

## 环境与执行边界

- Docker 29.6.2，宿主 macOS arm64；容器镜像
  `sha256:01ab8caf0eb816fa7e4336aed7cfdfaf57fc2ee5530d6b3e754ac7d395949eb9`，
  `linux/amd64` 通过 QEMU 运行。
- 容器 Go 1.26.5、libvips 8.16.1、govips 2.18.0。当前仓库只读挂载，宿主 Go module cache 只读
  挂载，`GOPROXY=off`；公开许可原图只读挂载，输出只写临时目录。
- 当前生产 `internal/media/imagevips` 的 libvips-tag 测试在该环境通过。新增隔离工具
  `spikes/int001-vips-input` 直接调用同一 `imagevips.Processor`，没有复制 resize/codec 规则。
- 完整 Dockerfile 构建先后在 Docker Hub 的 Dockerfile frontend 和 Node manifest 遇到 `EOF`；使用
  本地缓存镜像与 module cache 才完成离线运行。这不是代码失败，但保留为境外供应链可用性证据。
- 随后从 `mirror.gcr.io` 取得与 Dockerfile 完全相同 digest 的 Node 基础镜像，本地补标签后重新执行
  未修改的 `make test-libvips`。Dockerfile 在宿主原生构建 Linux/arm64 的 Expat、修补 GLib、libvips
  8.16.1、Go 应用与测试 stage，`internal/media/imagevips` 测试通过；输出 build manifest 为
  `sha256:3686d08675dc8f5a20a34635d30ad90cbe81e8a51a1836dae98526752fd73502`。

## 派生输入

生产 adapter 将 10 张、33,216,272 bytes 原图生成 10 个最长边 512px、WebP quality 82 的临时文件，
合计 265,516 bytes。所有输出尺寸与 Pillow surrogate 一致；逐像素 RGB 对照的平均 normalized MAE
为 0.008644、平均 normalized RMSE 为 0.013862。文件 byte hash 不相同，说明不能把 Pillow 产物当作
生产变换的 byte-equivalent fixture。

同一容器重复生成的 manifest/文件 hash 一致只用于当前缓存环境的开发复现；QEMU 不能证明原生
amd64/arm64 byte determinism 或 embedding 容差。两次 manifest SHA-256 均为
`42be241529815f3e9a9fd7ad4aa254f0083c6452cd30a6b26996a9bdd327ee92`。

相同工具随后在新构建的原生 Linux/arm64 stage 运行。arm64 native 与 amd64 QEMU 的 10 个 WebP
逐文件 byte-for-byte 相同，尺寸、字节数和每项 SHA-256 全部一致；规范 item 摘要 SHA-256 都是
`b79f8d1dac3905fa4db785482a5abf1b4d9289056b989c28519db87596c05dea`。这完成了当前 fixture 的
跨编译架构输出一致性开发证据，但 amd64 仍是 QEMU，不能声称原生 amd64 性能/内存通过。

## 检索结果

| 候选 | 中文 R@1 / R@3 | 英文 R@1 / R@3 | RSS after images | 图片 P95 |
| --- | --- | --- | --- | --- |
| SigLIP 2 base patch16 224 | 1.000 / 1.000 | 1.000 / 1.000 | 1,177,174,016 B | 59.110 ms |
| SigLIP base patch16 224 | 0.917 / 1.000 | 1.000 / 1.000 | 728,334,336 B | 60.123 ms |

对两个模型，24 条查询的 Top-1 和 first-relevant rank 相对 direct-original 与 Pillow surrogate 均未
改变。SigLIP 1 的中文红金盔甲查询仍为 rank 2。SigLIP 1 英文 mean relevant Recall@3 从 Pillow 的
0.931 变为 0.972，但样本过小，不能解释为模型或 transform 改善。

## 当前决定

1. “模型只消费有界派生输入”得到生产 adapter 的开发支持；直接原图路径不应进入生产设计。
2. 不能直接冻结复用 grid thumbnail：当前 arm64 native 与 amd64 QEMU 输出已 byte-identical，但仍需
   原生 amd64，以及 ONNX runtime 的跨架构 embedding 容差；还要决定 grid cache eviction、transform
   version 与 semantic backfill 的所有权耦合是否可接受。
3. 两个 SigLIP 候选继续保留。10 图结果不支持选型；至少 1,000 图、100 视频、ONNX Linux runtime、
   完整 FolioPath 4 GiB/browse concurrency 和 SBOM/许可签署仍阻断。
4. Docker Hub `EOF` 不证明必须运营项目镜像，但证明在线安装不能只有单一境外 origin；离线
   `/models:ro` 与部署者提供包仍是必需路径。

机器结果：

- [SigLIP 2 + govips input](semantic-vips-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 + govips input](semantic-vips-siglip1-darwin-arm64-2026-08-26.json)
- [arm64 native vs amd64 QEMU govips output comparison](semantic-vips-cross-arch-2026-08-26.json)

公开原图、派生 WebP 和生成 manifest 均只在临时目录，不随 Git 提交。
