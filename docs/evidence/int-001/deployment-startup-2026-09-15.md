# 部署启动组合验证（2026-09-15）

范围：CR-2026-030 / POST-MVP-5 / FR-INT-017/020 / NFR-INT-007/009/010。
Owner app，沿用 aimodel 激活队列、semantic 空间合同与 SQLite 原子提交。

## 实现

- 严格的可选 FOLIOPATH_AI_DEPLOYMENT JSON；解析不联网，不合法时拒绝启动。
- run.go 将共享的身份绑定客户端接入图片、文本、视频和激活适配器。
- 后台每 30 秒核验，登记 service 模型、按既有队列激活；每个可用性版本最多三次。
- AI 离线只标不可用，保留活动模型/代次；恢复不重新激活同一活动模型。
- 每次推理发送输入前核验身份；本地请求预算满不标离线。

## 已执行

- 本地 Docker remote-ml-regression 镜像构建成功（Linux/arm64，libvips）。
- Docker race：app 的 TestDeployment / TestSemantic / TestConfiguration 匹配测试，
  aimodel 全包、remoteml 非 Probe 测试。首次通过后加入并发满保护并复跑通过。
- make arch-check 通过。未运行完整七项仓库检查，也未声称完整 E2E 通过。
- 最终镜像重新构建后上述三个 Docker 测试命令再次全部 PASS，包含新增的
  TestConfiguredClientChecksIdentityBeforeSendingInput；git diff --check 通过。
- 真实 /out/foliopath 使用 synthetic tests/fixtures/media 只读挂载与独立测试数据卷。
  AI 停止后重建主容器，/health/ready 返回 ready，管理员设置返回 201。
- 专用 AI 镜像启动后，原队列自动完成服务模型注册/激活；API 显示 service / active。
  停止 AI 后模型 unavailable / availabilityRevision 2，活动 ID 保留；重启 AI 后
  同一 ID available / availabilityRevision 3，active 仍为 true。
- 首次宿主访问失败：internal-only 网络未发布主端口；修正为主应用另外加入
  candidate-web，6091 仅回环发布。AI 仍只在内部网络，没有发布端口。

身份从候选镜像 image_identity.py 离线读取，缓存清单大小 610841665 bytes。
许可字段明确 unreviewed-local-cache，不代表来源/许可审核完成。
测试后两个服务已停止，保留 candidate-app-data 测试卷，未删除数据。
未访问 server222，未扫描真实照片；仅缓存只读用于推理服务。

## 仍未完成

本轮未通过真实主进程创建库、回填、搜索全链路，也未完成 UI 操作验收。
此前独立真实模型持久化图片/视频测试不能替代该验收。人脸、正式 Compose、双架构
发行和完整回归仍待完成。候选主应用使用开发回归镜像，不是发行运行镜像。
