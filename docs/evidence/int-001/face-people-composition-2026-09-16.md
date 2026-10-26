# P2 人物服务生产组合增量

日期：2026-09-16。范围：CR-2026-030 / FR-INT-009，POST-MVP-5 S2 / P2 A05。
Owner：app / face / SQLite / API。合同：既有 api/openapi.yaml 人物、匿名组和人工审核端点，无新 API 或迁移。

## 实现

- app 在 face 部署启用时组合人物列表/详情、人物图片、资产人脸、匿名组列表/详情、人物修改和人工 review 服务。
- databaseService 复用生命周期读锁和未就绪保护，代理既有 SQLite 仓储；HTTP 不直接访问数据库。
- 人物详情查询经 face 服务校验 ID。分类、排除、合并和撤销继续使用原 ReviewService，不复制策略。
- 新增真实临时 SQLite 的组合测试：启停边界、创建、幂等重放、列表/详情、改名、旧版本拒绝、删除。
- 本地双容器探针增加正式 HTTP 人物创建、重放、空投影、改名、强杀重启后持久化读取和删除。

## 验证

本地 Docker 重新构建 app 和 remote-ml-regression 镜像成功。以下执行成功（exit 0）：

```sh
docker run --rm --network none foliopath-remote-ml-regression:local \
  '/out/app.test -test.run "FaceResults|Deployment|Face|SemanticContentSource" -test.timeout=3m &&
   /out/face.test -test.timeout=3m &&
   /out/api.test -test.run Face -test.timeout=3m &&
   /out/sqlite.test -test.run "Face|People|Person|CannotLink|GuardedUndo" -test.timeout=8m'
make fmt-check
make arch-check
git diff --check
```

编译测试二进制带 race；SQLite 使用临时合成数据，不读取实际媒体库。未运行完整七项发布检查，未进行容量或质量验收。

最终镜像双容器 HTTP 探针 `node spikes/int001-remote-ml/face-startup-probe.mjs` exit 0：
离线启动、CLIP/face 激活、停服恢复、正式分析/覆盖率、人物创建/重放/列表/改名/空投影、
强杀后同任务真实租约恢复、人物持久化读取/删除全部通过。自建临时容器、网络和数据卷已由探针清理。

`docker image inspect --format '{{.Id}}'` 返回：

- app `foliopath-app:face-worker`：`sha256:75bc3a6ea769624e9ff22a43cd028ae008c10dd8cd68f23d64adc4593ed91655`
- regression `foliopath-remote-ml-regression:local`：`sha256:0403455deb6afd056246350f530b3d01505daa9cb2113000ceaf59c0a7637fe6`

## 限制

本次夹具仍是合成零人脸图片。未证明真实非空检测、分组归类/排除/合并/撤销的最终镜像链路、浏览器流程、质量和发布。
P2 尚未完成；总清单保持 3/34 关闭、31 项未关闭。
