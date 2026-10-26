# 图片指纹变化后的覆盖率（2026-09-15）

范围 CR-2026-030 / POST-MVP-5 / FR-INT-005/015/016，Owner SQLite。
修复既有“当前有效结果覆盖率”合同，不新增 API 或迁移。

问题：扫描更新 assets.source_fingerprint 后，搜索排除旧 embedding，进度却仍显示完成。
修复：扫描有界 upsert 的既有事务调用 semantic_invalidation；只对与变更前指纹匹配
的旧向量所对应进度扣减 completed、增加 stale、推进 revision。设置 ready→degraded。
不会删除旧向量，不需要在 HTTP 请求上遍历整库；新回填复用原有进度重置与提交。

## 已执行

- 构建本地 remote-ml-regression 和实际 foliopath-app:candidate 镜像成功。
- Docker SQLite race 选择 TestScanInvalidatesSemantic、TestSemanticSearch、
  TestSemanticEmbedding、TestFailedScanPreserves：全部 PASS。
- 新增测试经真实扫描仓储验证变更、相同指纹重复 upsert、继续变更至另一新指纹、
  扫描失败保留结果；设置和搜索 snapshot 覆盖率一致，旧向量保留，搜索排除。
- make arch-check、git diff --check 通过。
- 实际本地 Docker app-chain.mjs（FOLIOPATH_TEST_INVALIDATION=1）通过：
  只改变 `/tmp/foliopath-app-fixtures.*` 合成图片的时间戳，调用扫描 API，
  等待 completed=0 / stale=1 / complete=false / state=degraded；查询结果为空且
  coverage 也不再显示完成。请求新的 missing 回填后 completed=1 / stale=0、检索恢复，
  activeGenerationId 不变。图片及 4 帧视频基础链路同轮仍 PASS。

没有访问 server222 或真实照片。测试主进程、AI 已停止；测试数据卷和临时夹具保留。
未运行完整仓库七项检查、完整浏览器 E2E 或双架构发行验证。

## 边界

本子片只处理扫描已观察到的图片指纹变化。不是实时文件监控，也不自动修复升级前
已经漂移的历史计数（可通过新的 missing 回填重算）。新增/删除资产、指纹回退复用、
视频/标签/人脸覆盖率以及变更与全量回填并发矩阵仍需后续验证，不能标为全部完成。
