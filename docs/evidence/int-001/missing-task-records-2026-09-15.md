# 旧实例任务记录提示修复

CR-2026-030 / POST-MVP-5 接入验证阶段 / FR-INT-016，Owner 前端 ai。
修复既有任务追踪流程，沿用 OpenAPI `404 ai_operation_not_found`，不新增 API 或服务端删除能力。

任务页在明确的 404 + ai_operation_not_found 时显示“当前服务中找不到此任务”，
说明浏览器记录可能来自旧实例。对浏览器持久化的该 ID 提供“移除浏览器记录”，
复用 preferences owner 写回剩余 ID；不调用取消、清除或删除 API。
不存在提示优先于可能残留的旧 query data，不继续显示旧任务成功状态。
其他错误仍显示原重试界面，不自动删记录。仅服务端恢复的下载任务不提供本地移除按钮。

## 本地 Docker 验证

- `web` 构建目标成功，包括 TypeScript 检查与 Vite 打包。
- `docker run --rm --network none foliopath-web-check:local npm run test -- src/features/ai/pages/AISettingsPage.test.tsx src/lib/api/ai.test.ts src/features/ai/queries.test.ts`：42/42 通过。
- 新增 4 个用例：准确不存在时仅移除目标本地 ID、保留另一任务且不调用服务端修改；
  网络错误、错误状态码和认证失败不提供移除并保留重试。
- `make arch-check` 通过。

回归镜像 manifest list：`sha256:b96f0d6f8488d6bce04abef8e22eadaf0f24059614b51d2ebae827168a1ec52f`。
本轮未重建运行用主应用镜像，旧实例真实浏览器卡片的修复后验收待补；
未执行完整七项验证。人物、混合任务压力和发行验收仍未完成。
