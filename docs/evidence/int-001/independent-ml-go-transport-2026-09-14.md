# 独立 ML：Go 传输子片证据

范围：POST-MVP-5 / CR-2026-030；[内部合同](../../architecture/independent-ml-transport-contract.md)。
Owner：inference；本片不改变主应用、数据库、公有 API、UI 或生产部署。

## 已执行

- `go test -race ./internal/inference/remoteml ./internal/semantic`：通过。
  合成测试覆盖图片/文本 multipart、原文保留、归一化、768 维误用、零向量、null、溢出、
  JSON 损坏/尾随内容、64 KiB 响应上限、输入限制、错误脱敏、重定向拒绝、无环境代理、
  并发满额、取消/截止释放及连接失败。
- Go 测试二进制交叉编译 Linux/arm64，置于 `/tmp/foliopath-go-ml-probe.oLgHxx/probe.test`。
  使用 spike 的内部网络及相同固定 Immich 镜像运行 `TestOfflineServiceProbe`：**PASS，0.68s**。
  图片为内存生成 224×224 红色 PNG；文本 `a red square`。图片/文本各 512 维、单位范数、
  重复文本误差小于 1e-5。只读复用已复制模型缓存，HF 离线，无媒体/数据库挂载或公开端口。
  可复现命令见 [spike README](../../../spikes/int001-remote-ml/README.md#go-adapter-probe)。
- `make arch-check`：通过。
- `make fmt-check`、`git diff --check`、`go vet ./internal/inference/remoteml`：通过。
- 联调后已停止本地 ML 测试容器，保留模型缓存和测试二进制，便于继续验证。

## 本地补充回归与 UI 设计复核

用户要求此后仅在本地测试；本轮无 SSH/server222 操作。

- `go test -race ./internal/inference/remoteml ./internal/semantic ./internal/face` 通过。
- 新增正文超时、截断后同客户端恢复测试；`go test -race -count=5 ./internal/inference/remoteml` 通过。
- 本地已有 Go probe 二进制在同一固定镜像/离线缓存复测通过（0.69s）。
- Python probe 再次通过：图像/文本 512 维、重复稳定、纯色图人脸数量 0。
- `npm run test -- src/features/ai/pages/AISettingsPage.test.tsx src/features/ai/pages/AIModelProxyPanel.test.tsx`
  在 web 执行，2 文件 / 11 项通过。仅为旧 UI 回归，不是新 UI 实现证据。
- 本地 Storybook 6007 已恢复，浏览器截图检查旧模型页；生成三个待用户选择的设计候选。
  未改生产页面。后续实现仍需独立服务主应用合同，不把示例状态接进生产。

## 尚未完成的边界

无模型身份/完整缓存指纹准入；无新 generation、搜索/回填/重启集成、人脸 Go adapter、
自动下载、UI/正式 Compose 或 amd64 实机证据。未读取 coser 媒体，未修改 server222。
未执行全量 lint/test/integration/e2e；不将历史 Gate 迁移成新架构验收。
