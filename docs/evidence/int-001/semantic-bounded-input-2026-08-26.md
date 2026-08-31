# 512px 有界语义输入 surrogate — 2026-08-26

状态：macOS 开发方向证据；**不是生产 libvips、Linux 或 4 GiB Gate 证据**。

## 问题与方法

上一轮直接让 Pillow 解码最高 5,388×3,592 的公开许可原图，测到的 RSS 混合了模型与无界原图解码
成本。FolioPath 当前正式 grid thumbnail 合同是最长边 512px、WebP quality 82。因此本轮新增隔离脚本
`semantic_input_prepare.py`，先检查 256 MiB、100 MP、32,768px 上限，再用 Pillow JPEG
shrink-on-load、EXIF 方向和 512px WebP/quality 82 生成临时输入。

这只是参数对齐的 surrogate：本机没有 `vips` CLI 或 `pkg-config`，没有执行生产 govips/libvips。
Pillow 11.3.0/libwebp 1.5.0 与生产 libvips 的像素、色彩、重采样、内存和错误语义可能不同。

10 张来源图从 33,216,272 bytes 变为 262,630 bytes。相同环境重复生成的临时 manifest SHA-256
均为 `3f351ce4abad3e0bfdb1456e1f10a0981308b7164d7e608d63a0ea5c1961e3fb`；这只证明当前本机版本重复
执行一致，不承诺跨平台 byte determinism。

## 三次独立模型进程结果

| 候选 | bounded RSS after images | bounded image P95 | 质量稳定 | 相对 direct-original 的查询排名 |
| --- | --- | --- | --- | --- |
| SigLIP 2 base patch16 224 | 1,178,173,440 / 1,177,714,688 / 1,040,252,928 B | 71.286 / 59.669 / 59.606 ms | 是 | 24 条 Top-1 与 first-relevant rank 均未改变 |
| SigLIP base patch16 224 | 725,565,440 / 727,580,672 / 729,464,832 B | 60.227 / 59.924 / 61.338 ms | 是 | 24 条 Top-1 与 first-relevant rank 均未改变 |

RSS 中位数相对 direct-original 单次诊断由 1,709,178,880 降到 1,177,714,688 bytes（SigLIP 2，
约 -31%），由 1,268,580,352 降到 727,580,672 bytes（SigLIP 1，约 -43%）。两个候选的汇总
Recall@1/R@3/MRR 结论未改变；SigLIP 1 的中文红金盔甲查询仍是 rank 2。SigLIP 2 第一次 P95 较高，
后两次约 59.6 ms，说明 10 图单次计时噪声很大，不能将单个 P95 当作吞吐结论。

## 决定

1. 后续模型输入应复用受控派生图，不允许任意原图直接进入 inference runtime。
2. 尚不能冻结“直接复用 grid thumbnail”还是建立独立 semantic transform。grid 的 512px/quality 82
   可能满足当前模型，但模型升级、色彩处理、缓存清除和 transform version 所有权需要 S1 决定。
3. 下一项必需证据是在生产 govips/libvips 生成同一 fixture，比较像素/embedding/排名，再放入完整
   FolioPath Linux 进程进行 4 CPU/4 GiB、浏览并发和 100k backfill 测试。

机器结果：

- [SigLIP 2 bounded run](semantic-bounded-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 bounded run](semantic-bounded-siglip1-darwin-arm64-2026-08-26.json)

只有第一轮完整 JSON 进入仓库；三轮汇总列出全部 RSS/P95，并确认三轮 metrics 完全一致。临时 WebP
和派生 manifest 不随 Git 提交。
