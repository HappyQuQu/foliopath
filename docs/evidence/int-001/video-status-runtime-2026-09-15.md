# 视频状态实际镜像及自动恢复验收

CR-2026-030 / POST-MVP-5 / FR-INT-004/015/016。Owner app/semantic/前端 ai。
本轮只测试本地 Docker、既有临时合成夹具；未接触 server222 或真实媒体。

## 实际运行

- 重建最终运行镜像 `foliopath-app:candidate`，最终 manifest-list
  `sha256:51fe0ded4e8424f7a3bf2f9ba38be0992005f29a976ad86fbfb84f1dac370005`。
- 主应用仅发布 `127.0.0.1:6091`；AI 私网、无公开端口、缓存只读。沿用隔离测试数据卷。
- `app-chain.mjs` 增加专属状态与真实搜索覆盖率比对：视频 eligible=2、completed=1、degraded=1、failed=0、stale=0、complete=false，逐字段一致。图片仍为 1/1。
- 浏览器实际设置页确认图片/视频分别展示；不是 Storybook 数据。
- 停止本地 AI 容器后，`video-status-probe.mjs` 等待 active model unavailable，专属状态仍返回原覆盖率；不发起推理或任务。
- 恢复 AI 容器后，同一探针确认 available，覆盖率不变。

## 验收发现与修复

首次启动期间模型列表和 ready 图片状态无轮询，服务恢复后 UI 会滞留旧数据。
在 canonical queries 增加 10 秒刷新；building/clearing 仍为 1.5 秒，不引入第二计时器。
实际浏览器同页不 reload，验证“已启用模型 → 尚无可用模型 → 已启用模型”自动更新。

Docker web 33 项定向测试通过（页面 22、API adapter 8、query 配置 3）；构建包含双 TS 检查与 Vite，
修正首轮新测试的数组可空类型报错后重跑成功。`make arch-check` 通过。保留既有 chunk 大小警告。

## 明确未完成

- 图片状态接口的读取依赖 model.state=available（semantic_settings.go），离线时返回 0/0，恢复为原 1/1。
  本次验证没有数据丢失，但不能把离线 0/0 视为完成；此展示/读取语义仍待修复。
- 人物生产路由仍未接入；视频操作按钮、容量/并发完整矩阵、完整 E2E 和发行七项验证仍未完成。
- 本轮不改变发布 Gate，不提交推送。测试结束停止本地两个容器，保留镜像、合成夹具和隔离数据卷便于复测。
