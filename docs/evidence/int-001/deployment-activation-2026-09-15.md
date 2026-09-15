# 服务代次激活事务验证

2026-09-15；POST-MVP-5 / CR-2026-030 / ADR-0020；本地 Docker Linux/arm64。

## 实现边界

semantic.DeploymentActivationRepository 定义带 SpaceIdentity 的提交合同；SQLite 原
CommitAIModelActivation 和新 CommitDeploymentActivation 进入同一私有事务 owner。
空间绑定复用从原 BindSemanticGenerationSpace 提取的事务 helper，不复制校验策略。

核对 running model_activate、操作 revision、模型可用性 revision、service 类型、用途、
content hash 与维度。事务内新建 building→绑定→active，旧 active 退役，指针/操作更新；
任何失败回滚。旧文件入口拒绝 service，服务入口拒绝文件模型。无新迁移或 API 变动。

## 已执行

```sh
docker build --target remote-ml-regression -t foliopath-remote-ml-regression:local .
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/sqlite.test -test.run "TestDeploymentActivation|TestAIModelActivation|TestGenerationSpace" -test.v -test.timeout=3m && /out/semantic.test -test.timeout=3m && /out/aimodel.test -test.timeout=3m'
```

构建与联合命令 exit 0；所有二进制启用 race。
新事务测试 7 个场景 PASS（14.86 秒）：成功、错误空间、错误维度、过期模型 revision、
误用旧入口、插入绑定时注入失败、误用文件模型。
成功案例确认身份绑定及活动指针、重放旧 commit 拒绝；所有失败案例旧代次仍 active，
没有残留新代次，操作未误标 succeeded。所有场景旧 embedding 均保留。
旧激活/空间绑定回归、semantic 与 aimodel 全包 PASS。

按专用候选镜像 Compose 流程启动服务，从镜像文件独立读取预期身份，复跑 persisted-probe。
探针已使用新的服务模型记录和真实原子激活提交，不再直接 UPDATE 代次为 active。
结果 PASS（5.05 秒）：3 张合成图片回填，数据库重开后文本检索；四帧视频计划入库检索，
不完整计划不返回。模型描述/空间与队列准入仍为测试夹具，不是生产部署加载器。

宿主静态 `make arch-check` PASS，`git diff --check` PASS。
本轮未跑全仓 lint/生成/E2E/SQLite 全包；未使用 server222 或原媒体，未提交/推送。
主应用配置、激活调度及生产运行时组合未接入，不能宣称正式界面可用或整体验收完成。
