# 服务模型记录与迁移验证

2026-09-15，POST-MVP-5 / CR-2026-030 / ADR-0020；本地 Docker Linux/arm64。

## 范围

ai_models 新增 service storage mode；aimodel 内部 RegisterDeployment 接受部署核验后的
语义模型描述，source identity = space fingerprint，不能是目录或 URL。
普通 RegisterInstalled 不接受 service，旧文件 AvailabilityService 跳过它，旧激活
StartActivation 明确拒绝。API 模型响应 enum 包括 service；安装请求仍仅 managed/direct。

migration 39 沿用既有 v35 的表重建策略；升级前旧代次及一次性空间绑定完整保留，
外键仍有效。服务行不允许 installed 状态或路径 source identity。降级到 v38 因服务行
约束失败而拒绝；测试显式 rollback 后验证模型保留。不是生产降级操作。

## 已执行

- Docker 构建 regression 成功，包含 aimodel/contract 测试二进制。
- Docker `/out/aimodel.test -test.timeout=3m` 全包 PASS（race）。
- Docker `/out/sqlite.test -test.run "TestServiceModelMigration|TestAIModelActivation|TestGenerationSpace" -test.v -test.timeout=3m`：9 项通过（race）；新迁移测试 2.63 秒。
- 宿主生成 `cd web && npm run generate:api` 成功；`make generate-check` 和 `make arch-check` 成功。
- Docker 构建中的前端类型检查和 Vite 构建成功。
- Docker `/out/contract.test -test.timeout=3m` API 合同全包 PASS（race）；`git diff --check` PASS。

本轮没有执行全仓 lint/E2E 或 SQLite 全包；只更改测试临时库，没有打开本地应用库、
没有操作 server222、没有提交/推送/部署。注册接口未被主应用组合调用，新模型没有实际激活。
下一步仍需服务代次激活事务、主应用配置/运行时组合与真实 API/UI 联调。
