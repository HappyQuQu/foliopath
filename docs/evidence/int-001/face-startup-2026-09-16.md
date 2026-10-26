# 人脸部署配置和主进程启动

CR-2026-030 / FR-INT-009 / NFR-INT-002、004；POST-MVP-5 S2，Owner app/face/aimodel。

## 实现

- `FOLIOPATH_AI_DEPLOYMENT.face` 可选；省略时行为不变。配置包含独立 face-v1 身份、两个组件 hash、显式阈值 profile、模型字节和许可标识；没有隐式默认 profile。
- 配置解析不访问网络，拒绝错身份、与 CLIP 不同的缓存/运行时、缺组件 hash、无效 profile、无效大小/许可。显式格式错误导致配置错误；省略人脸或运行时 AI 离线不妨碍核心应用启动。
- `run.go` 装配 face 激活适配与 bootstrap，复用原激活 worker 和共享客户端；数据库代理接身份绑定事务。
- bootstrap 复用同一重试逻辑，以 ActiveFaceModelID 判定人脸，独立更新可用性。每个可用性 revision 最多三次持久化激活尝试；本地 ErrBusy 不错误标记离线。主客户端生命周期仍由原 bootstrap 唯一关闭。
- 尚未装配人脸分析 worker/人物业务 API；模型 active 不等于图片已分析或人物功能完成。

## 验证

Docker `app.test -test.run Deployment -test.v -test.timeout=3m` 全通过，包含配置拒绝、适配和 semantic/face 两种离线→注册→重复核对→忙→离线→恢复；`aimodel.test` 全包通过。
真实候选 `foliopath-app:face-startup` 已构建，探针 `spikes/int001-remote-ml/face-startup-probe.mjs` 使用独立临时网络/数据卷、无媒体库和测试用未校准 profile。AI 只读缓存，不开放宿主端口。
探针首次发现 internal-only 应用网络不能被宿主访问，添加应用侧 bridge；随后 setup 请求缺必填 displayName，修正后重跑。两次探针问题不登记为成功。
真实探针最终 exit 0：首次离线核心 ready 且无未核验模型；服务上线后两个模型均 active/available；停服后两个记录均 unavailable 且保留；恢复后 available；重启主应用后仍为相同两个模型 ID。全部使用临时数据卷和测试 profile，不证明阈值质量或人脸分析入库。
应用候选镜像 config digest：`sha256:1107b87ded77ec002438a8db5d2062d0e24cad1caf5c6494fb386d7be0ec226c`。
探针已停止并删除本轮测试容器、网络和仅含合成管理员/模型记录的临时卷；这些测试状态不保留。原缓存只读，未触碰用户运行库或 server222。
`make arch-check`、`git diff --check`、探针 `node --check` 通过；未执行完整七项验证或浏览器人物流程。
