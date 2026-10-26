# 浏览器多库选择与差异覆盖率验收

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-004/016。
Owner 前端 ai；沿用既有选择器、URL、任务和状态合同。本轮无生产代码变更。

## 环境与操作

复用 `foliopath-multilib` 本地 Docker 双容器、合成 alpha/beta 视频及隔离数据卷。
Compose 启动健康，主应用 localhost:6091；镜像仍为
`sha256:39d8d0d74b6ab6f90c49e38b07382b99e7d13cf6665fd75ab207a4bba8f89193`。
通过实际浏览器登录 multilib-test，未使用 Storybook 或模拟 API。

1. alpha 初始视频 1/1；选择 beta 后 URL 为 `?library=lib_2`，
   开关标签和视频进度标题均指向 beta。
2. 暂停本地 AI，在 beta 确认重建；beta 变为完成 0/1、待处理 1。
   从任务页取消，观察 running→cancelling→cancelled，处理 0/1；随后恢复 AI。
3. 返回总览，beta 仍为 0/1。选择 alpha，URL 变为 `?library=lib_1`，
   视频仍为完成 1/1、待处理 0。刷新页面，选择和结果均保留。
4. 再选择 beta，仍为 0/1，不误用 alpha 的缓存覆盖率。
   点击 beta 的补齐按钮，视频恢复 1/1，待处理 0。

结论：选择、URL、显示和操作目标在本次差异状态流程中正确隔离。
测试结束停止容器，保留合成媒体和数据卷。不代表全部任务页或发行已完成。

## 新发现的未完成项

该浏览器之前访问过同一 origin 上的另一个隔离实例；任务页残留 5 张旧任务
“暂时无法载入/重新尝试”卡片。新任务可正常取消，但旧任务提示没有区分永久不存在。
代码读取 `web/src/lib/storage/preferences.ts` 中本地持久任务 ID，
`TasksPanel` 对所有 query error 一律显示暂时加载失败。本次未清除本地存储掩盖问题。
后续需核实并区分 `ai_operation_not_found` 与可重试网络错误，补回归和处理旧记录入口；
不能自动删除服务端任务或把所有读取失败当作不存在。

本轮是实际浏览器功能验收，无截图视觉矩阵或完整 E2E 套件；
`make arch-check` 及本轮已跟踪文档 `git diff --check` 通过。
人物生产路由、混合任务压力和发行验收仍待完成。
