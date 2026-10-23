# 独立 ML 编码图片入口子合同 v1

2026-09-14；CR-2026-030 / ADR-0019；POST-MVP-5；FR-INT-017/019/020、NFR-INT-007/009/010。
Owner：semantic 定义输入预算/接口，imagevips 实现编码。仅实现独立适配及测试，
不接入现有 SigLIP 张量处理器，不激活代次，不改 API 或原媒体。

输入只接受调用方经 internal/files 打开的 io.ReadSeeker；调用方持有关闭责任。
JPEG/PNG/WebP/GIF，编码源最多 32 MiB，最多 16 百万像素；不支持超大 JPEG 特殊回退。
核对声明格式与实际格式，首帧、自动转正、sRGB，丢弃 alpha，保持长宽比缩小到最长边
1024（不放大），Lanczos3，8-bit RGB PNG，移除元数据，输出最多 8 MiB。
CLIP crop/normalization/tokenizer 仍由固定 ML runtime 持有，本入口不冒用 SigLIP 预处理。
此变换必须作为未来预处理身份的一部分，不能复用未处理原图探针的空间身份。

错误返回现有 semantic 安全错误，不透传 native/IO 正文。阶段间检查取消；不能中断正在
执行的 native 解码/导出，不能中断阻塞的调用方 ReadSeeker。app 未来必须通过已有有界
worker 与容器资源限制运行，不能把阶段检查视为 native 强制超时。当前尚未组合 app。
非 libvips 构建失败关闭。合成测试之外，不读真实照片；完整主应用/Linux 边界验证另行完成。
