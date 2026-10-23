# 安全编码到真实 ML 服务的本地链路验证

CR-2026-030 / ADR-0019，POST-MVP-5，FR-INT-017/019/020、NFR-INT-007/009/010。
Owner：inference 隔离测试；复用 semantic 预算及 imagevips 入口。

## 执行

`Dockerfile` 新增 opt-in `remote-ml-image-probe` target，正式 runtime stage 不依赖它。
`TestEncodedImageServiceProbe` 仅在 `libvips && remotemlprobe` 标签下编译。
生成 1200×600 红色图片，编码为 PNG/JPEG/GIF/WebP，分别经过正式
`PrepareRemoteSemanticImage` 得到 1024×512 PNG，再经 Go HTTP client 调用真实离线 ML。
四格式子测试全部通过，每个结果为有限、单位范数的 512 维向量；另有文本向量通过。
总测试耗时 1.07 秒，仅为本次合成 smoke 的观测，不是性能或质量承诺。

- `docker build --target remote-ml-image-probe -t foliopath-remote-image-probe:local .` 成功。
- probe run：1 CPU / 512 MiB / 64 pids、drop ALL capabilities、no-new-privileges、只读根文件系统，
  同机 internal network，无端口发布、无 volume mount。
- ML 使用既有固定 v3.1.0 digest 与只读缓存；测试后已停止。
- 测试前后 49 文件清单指纹均为 `c8be0544ab43ffd0c3f81e550218c833410d4e593a3b282fc7844f811606211b`。
- Go 相关包竞态测试和 `make arch-check` 通过。
- 未访问 server222、未读用户照片、未写 SQLite、未提交或部署。

## 仍未覆盖

并非 internal/files → app → 队列 → 新代次持久化的完整链路；本测试使用内存 stream。
没有 EXIF/ICC、透明度、动图帧选择的精确像素矩阵，没有语义质量、人脸质量或 amd64 证据。
来源认证、模型代次/查询/回填/重启集成仍待完成，不能将本报告计为主应用 AI 功能已可用。
