# P5 交付与安全收口证据

2026-09-17；POST-MVP-5 / CR-2026-030 / ADR-0019、0020 / F03、F04、G02。
状态：**F03、F04、G02 验收通过，P5 5/5 完成**。最终完整探针项目 `fp-p5-a550603c`，退出 0，测试容器、网络及自产临时数据清理完成。没有删除用户媒体、配置或原有镜像。

## 固定本地镜像

- 新主应用 `foliopath-app:p5-delivery`：`sha256:5629c6119b72c02876734b2ba46d673a4821f223ff3916860d815c8f0c6fac99`，arm64。
- 新 AI `foliopath-inference:p5-delivery`：`sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689`，arm64，face 构建。
- 配对回退旧主应用 `foliopath-app:p4-oom-fix`：`sha256:0421fb761cbf2d00c847038f566dfdebb031f8b59ca4cf02630995ac86503b64`。
- AI runtime/cache 身份仍为 `6e5aadf096b691ce0f1e9b6c99a5ddc13fa01a1a326fb3a39b32f4e63b4c42a4` / `e6d671f9d9fada323fd77a02d217349f46af8e103eccb297b1ebcac10112efe8`；新 OCI digest 不冒称新模型空间。

两个镜像均从本地当前工作树用正式 Dockerfile 构建成功。主应用首次 apt 下载网络失败，显式使用用户已提供的 HTTP/HTTPS 构建代理后成功；未关闭证书校验，检查两个运行镜像 Env 不含该代理。构建使用网络，离线运行单独验证。没有推送、公开发布或更新 server222。

## F01/F02 在新镜像复验

显式设置上述新镜像后执行 `make test-ai-compose`，项目 `fp-p5-9ef65c72`，退出 0。
实际根 Compose 初始化、管理员登录、只读挂载和资源/网络检查、两个模型分析、缺失缓存/等长损坏/错身份三类故障及逐项恢复、原代次和结果保留、原件 hash、重启登录全部通过。不以早前 P4 镜像替代新构建结果。

## F03/G02 最终运行

命令（均为本地测试镜像，不读真实媒体）：

```sh
FOLIOPATH_TEST_APP_IMAGE=foliopath-app:p4-oom-fix \
FOLIOPATH_TEST_UPGRADE_IMAGE=foliopath-app:p5-delivery \
FOLIOPATH_TEST_AI_IMAGE=foliopath-inference:p5-delivery \
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
make test-ai-delivery
```

使用实际 `compose.ai.yaml`，仅测试覆盖将 web 网络也设为 internal；HTTP 控制器从内部网络访问，不打开出网。独立临时数据/备份/恢复目录、两张合成媒体，不影响已有实例。

退出条件：停写备份 → 新镜像零继承旧语义完成数 → 重建及人工人物/标签保留 → 重启 → 旧镜像配对备份恢复 → 返回新镜像 → 鉴权/CSRF/路径/日志诊断 → 两类人脸清除及数据库记录检查 → 重启，全部断言通过且清理完成。

最终实测全部满足：旧语义代次 `aig_UJeJEwqKmH2VK_LDmzb5XXqK`，新代次 `aig_h1Qv-ovoBNyFwK-1H-RuQjsm`；`deliveryAndSecurityPassed=true`，外层重启登录及 `passed=true`。派生清除后观察行 0、人工锚点/人物各 1；重建后人工归属恢复。人工清除后观察行仍大于 0、锚点及本库审核事件/撤销快照为 0、人物仍 1；人工标签和原件 hash 保留。返回新镜像完成最终检查，不在旧镜像上代验新版本。

## F04 文档与配置

新增私有部署 env 模板、构建与离线镜像传递步骤、安装、配对备份/升级/回退、模型来源/限制、恢复排障和残留边界。未填写的 env 模板实际执行 Compose config 失败关闭。
README、部署说明、AI 镜像说明、用户流程和 UI 说明已对齐当前双服务入口；旧模型商店/代理明确归兼容维护，保留已有路由和人工状态，不恢复为主线。核对当前正式 UI 文案已有“兼容维护”与独立服务状态区分，本轮未修改前端功能。模型来源仅有字节基线，许可/供应链不冒称已批准。

## 安全专项与验证范围

- 当前源码在本地 Docker、无网络、`-race -tags=libvips` 下执行 API/files/SQLite 选定安全回归通过，包含认证/CSRF、只读路径、两类清除与 secure_delete 删除 canary 后活动 DB/WAL/SHM 检查。
- 另运行 Linux openat2 两个顶层测试及错误子案例，既有 mount 边界和失败关闭通过。
- 新 AI 镜像 Python 合同/快照测试 13/13 通过。
- 本地 Docker、无网络：当前 AISettingsPage 组件专项 31/31 通过；不称本轮全浏览器 E2E。
- `make arch-check` 通过，人脸隐私静态规则现覆盖 remoteml；脚本语法和 `git diff --check` 通过。

清除不承诺擦除用户备份、存储快照、文件系统日志、人工实体或普通图库缩略图。派生清除与人工关系清除分别检查，不写成“所有人物信息已擦除”。经认证的人物 API 返回人工姓名属于业务合同；禁止姓名/查询进入日志诊断，禁止公共 API 暴露向量。

## 失败与修正记录

1. 测试误期望 Cookie SameSite=Lax；真实合同为更严格的 Strict，按实际合同修正。
2. 测试把业务端点“持有正确 CSRF token 但 Origin 不同”误当应拒绝；既有业务合同是 session + token，初始化/登录额外同源校验。改为缺失/错误 token 拒绝，初始化/登录跨源拒绝；不改生产鉴权政策。
3. internal-only 网络不产生宿主发布端口，离线测试不能继续用宿主随机端口；改用内部测试控制器，未放开服务出网。
4. 两类清除后人脸 API 按合同返回 409 face_not_ready，而非 200 空列表或旧结果；分别修正断言，以数据库检查区分派生清除删除观察/向量、人工清除保留观察。不能用同一个 HTTP 状态推断数据是否存在。
5. 只读挂载的 WAL 库查询因无法建立 SHM 而失败；检查器改为在停写后复制完整 DB/WAL 到私有 tmpfs，以只读连接查询副本，不忽略 WAL，不修改原测试库。

上述失败轮次不计完整通过；最终退出结果另在本页状态中确认。本轮不修改生产 API 或质量规则来迎合测试。

## 不包含

不执行 P6 的完整七项检查或发行批准，不证明当前新镜像的 native amd64、100k AI 联合容量或模型准确率。镜像 save/load 步骤提供给部署者，本轮没有在另一台全新 Docker 主机进行离线归档导入验收。P5 的本地工程交付与 P6 发行状态分开。
