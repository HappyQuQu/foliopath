# 远程人脸激活调度合同

CR-2026-030 / FR-INT-009 / NFR-INT-002、004，POST-MVP-5 S2，Owner aimodel；app 和 SQLite 后续实现身份绑定提交。
沿用 ADR-0019/0020，保持原激活队列、取消/重试机制和单 worker，不新建部署单元。

远程人脸以 service 模型记录，仅内部注册入口允许；浏览器文件激活入口仍拒绝 service。
激活必须使用独立的人脸部署运行时，验证模型/缓存/变换/阈值兼容性后返回组件 hash、维数与阈值 profile。
worker 校验 FaceGeneration 后把提交交给该运行时的身份绑定仓储入口；不得走本地文件提交或 CLIP 提交。
尚未装配该入口时失败关闭，不能仅凭模型存在启用分析。

实现内部 `RegisterFaceDeployment`、独立运行时启用及加载/提交分支。加入 migration 40 扩展 service 模型 purpose，旧 SQLite 人脸提交明确拒绝 service。

首次数据库回归发现 migration 39 的 semantic-only 约束；新增 40 后六项旧测试仍断言版本 39，现改为 40，业务断言未删除。这些首次失败不计作通过。

本子片不代表主进程已配置人脸代次或完成实际入库。尚缺 app 身份/阈值准入适配、身份绑定的 SQLite 人脸代次提交、启动配置、分析 worker 与人物服务装配。测试运行时使用合成 metadata，未证明真实模型激活。

## 已执行验证

本地 Docker `remote-ml-regression` 构建成功，以下带 race 二进制依次运行，最终 exit 0：

```sh
/out/aimodel.test -test.timeout=3m
/out/sqlite.test -test.run 'TestFace(ModelActivation|ServiceCannot)|Migration' -test.v -test.timeout=5m
/out/app.test -test.run Deployment -test.timeout=3m
```

覆盖人脸服务内部注册/幂等准入、独立运行时、无配置拒绝、无效阈值 hash 拒绝、取消透传、可用性 revision 冲突、旧文件提交隔离，以及所选迁移回归。无效代次测试具体变异为清空阈值 hash，并非全部 metadata 字段穷举。
`make arch-check` 和本次代码 `git diff --check` 通过；Go 文件 gofmt。未运行全量 SQLite/完整七项检查或真实人脸激活。本轮仅临时 Docker 合成库，无真实媒体、server222 或当前运行库写入。
