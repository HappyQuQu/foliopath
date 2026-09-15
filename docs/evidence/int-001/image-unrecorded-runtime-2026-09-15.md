# 图片未记录到真实回填：实际镜像验收

CR-2026-030 / POST-MVP-5 / FR-INT-005/015/016。Owner app/semantic/前端 ai。
完成上一条 image-unrecorded 记录中“实际镜像新代次验收待补”的子项，不扩大人物或发行 Gate。

## 隔离环境

- 最终运行镜像 `foliopath-app:candidate` manifest-list：
  `sha256:de10ee18783e164f242b88295707433a73b5a216673df06259ba010b2fbd7f0b`。
- 新 Compose project `foliopath-unrecorded`，创建前确认无同名数据卷。
  数据卷 `foliopath-unrecorded_candidate-app-data`，未修改旧 candidate 数据卷。
- 新合成媒体目录 `/tmp/foliopath-app-fixtures.ShH2vX`，共一张图片、两个视频；原目录只读挂载。
- 主应用仅 localhost:6091；独立 AI 无公开端口，沿用只读模型缓存及既定独立身份校验。

## 实际结果

1. fresh setup、建库和扫描成功，目录含 3 个合成资产；模型已激活且 available。
2. 未请求回填：coverageRecorded=false、complete=false，活动代次存在。
   浏览器总览、管理页均显示“当前模型代次尚无图片进度记录”，没有 0/0 或图片进度条。
3. 通过真实 PATCH 开启并请求 missing 回填，真实模型推理完成，任务 succeeded。
4. 返回 coverageRecorded=true，eligible=1、completed=1、failed=0、stale=0、complete=true。
   管理页不 reload 自动显示 1/1；返回总览也显示 1/1。
5. 视频尚未回填，仍显示“尚无视频进度记录”；不被图片完成状态覆盖。

复测脚本 `spikes/int001-remote-ml/image-unrecorded-probe.mjs`：默认 prepare 只可用于全新隔离数据卷，
随后 `FOLIOPATH_TEST_PHASE=backfill` 完成第二阶段。口令从环境变量读取，不写入仓库。
该脚本不是对任意实例可重复执行的通用初始化器，已有实例 setup 会失败关闭。

Docker 最终镜像构建（含前端类型检查与 Vite）、实际两阶段探针、浏览器交互检查通过。
make arch-check、generate:check 通过；本轮未重跑全量单元/E2E/七项发行验证。
人物生产路由仍未接入，视频操作按钮、容量/并发矩阵与正式发行验证仍待完成。

结束后停止本 project 两个容器，保留镜像、独立数据卷和合成媒体便于复查；没有 server222、真实媒体、提交推送。
