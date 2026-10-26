# 独立 ML 图片编码入口子片

2026-09-14；CR-2026-030 / POST-MVP-5；FR-INT-017/019/020、NFR-INT-007/009/010。
Owner：semantic 接口与预算，imagevips 实现。合同见
[图片入口](../../architecture/independent-ml-image-contract.md)。

## 实现

- 新 `EncodedImagePreprocessor`，只接受已打开的 stream，不接收路径。
- 与旧 SigLIP tensor 方法并列，未更换旧处理器或写入旧代次。
- 限制源字节/像素；实际格式核对；转正、sRGB、去 alpha、保持比例缩小、输出去元数据 PNG。
- 传输与图片入口共用 semantic 定义的 8 MiB 输出预算，避免两个 owner。
- 非 native 构建失败关闭；调用方关闭源文件，适配器释放 image。

## 已执行验证

- `go test -race ./internal/semantic ./internal/media/imagevips ./internal/inference/remoteml` 通过。
- `go vet ./internal/semantic ./internal/media/imagevips ./internal/inference/remoteml` 通过。
- `make arch-check`、`git diff --check` 通过。
- 本机无 pkg-config/libvips；改用仓库原有 `make test-libvips` Docker target，构建及 imagevips native 测试通过。
- 新 native 合成测试：2048×1024 PNG 输出 1024×512、格式不符、视频拒绝、取消、损坏输入；补充源字节/像素超限测试。
- 未使用真实媒体，未访问 server222，未改部署实例。

## 不代表完成的部分

本片未接入 internal/app，未执行新编码入口到 ML 服务的完整链路，未改 SQLite 代次或公开 API。
新入口的 JPEG/WebP/GIF、EXIF/ICC/alpha 精确语义和容器资源峰值还需专项矩阵，不能把包级测试当作全部质量验收。
阶段取消不是 native 强制中止；完整回填/查询准入、有界组合与新代次迁移仍待完成。
