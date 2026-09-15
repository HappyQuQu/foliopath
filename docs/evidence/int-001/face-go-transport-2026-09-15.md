# Go 人脸传输接入子片

CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005，POST-MVP-5 S2。
Owner：remoteml 私有 HTTP 适配；face 仍拥有质量/任务/聚类；app 组合尚未完成。
合同沿用 [face-v1](face-service-contract-2026-09-15.md)，无公开 API 或数据库变化。

## 已实现

- `Client.BindFace` 绑定部署独立提供的 buffalo_l 身份；拒绝 CLIP 身份以及与同一服务 CLIP 缓存/运行时 hash 不一致的配置。
- 共享既有 HTTP 连接池和 2 槽并发预算，不另开人脸连接预算；请求前检查 face identity，响应再次验证 hash。
- 固定 image multipart、8 MiB 输入、1 MiB 响应和 30 秒上下文；沿用禁代理、禁重定向设置。
- 只接收最多 64 张脸、合法归一化框、检测分数与 512 维有限非零向量，拒绝缺失/null 字段，不把空响应当成功零人脸。只返回传输结果，不编造 Quality 或 generation。

## 本地 Docker 已执行

- `remote-ml-image-probe` 阶段构建成功，包括 Go 主程序编译、带 race 的传输测试二进制。
- `/out/remote-image-probe.test -test.v -test.skip 'Probe$' -test.timeout=2m`：13 项顶层测试通过（包含新增 2 项人脸合同测试及原 CLIP/身份/故障回归）。
- 真实 `foliopath-inference:face-candidate` 以 `--network none`、只读根/模型缓存启动，无宿主端口；Go 测试容器共享其网络命名空间，独立给定预期身份，`TestFaceServiceProbe` 成功（0.35 秒）。仅 320×240 合成纯色 PNG，返回零人脸。
- `make arch-check` 通过；修改的 Go 文件已 gofmt。
- 首次未排除显式探针，原两个 CLIP 探针因没有提供服务地址失败；随后按上面命令运行无外部服务单元测试，真实人脸探针独立执行成功。未宣称原 CLIP 真实探针本轮通过。

测试用服务容器已停止并删除；未改当前运行 app 镜像或数据库，未访问 server222/真实媒体。

## 未完成

`run.go` 尚未实例化 FaceClient。安全图片解码到远程输入、独立人脸代次/阈值配置准入、JobProcessor/队列及人物 HTTP 组合仍待实现。
非空真实识别、持久化/聚类/人物闭环、完整七项验证、容量和发行仍未完成。
