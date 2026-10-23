# 视频只读状态与界面接通

CR-2026-030 / POST-MVP-5 / FR-INT-004/006/007/016。Owner semantic/SQLite/API/前端 ai。
Gate：现有视频任务状态消费者增量，不改变人物或发行 Gate。

## 实现

- 新增已认证 GET `/api/v1/libraries/{libraryId}/ai/video-semantic`，OpenAPI 与生成客户端同步。
- semantic 拥有窄读取服务，SQLite 单条 SELECT 读取库、设置、活动代次及其 progress；app 组合，HTTP 无 SQL。
- 没有记录返回 coverage=null，不把未知视为 0/0 完成。离线、关闭分析或模型不可用不删除既有记录。
- 不读取媒体、不请求推理、不新建任务；不复制搜索、回填和任务状态机。缺库 404、非法 ID 400、内部错误脱敏。
- 设置总览只查询当前所选库（10 秒轮询，页面卸载停止），图片和视频分别展示。
  视频显示完成/候选、降级、失败、过期与剩余待处理；请求失败隐藏旧数据。Storybook 只读夹具同步。

## 验证

- 本地 Docker regression 构建成功（编译实际主程序、libvips 竞态测试二进制）。
- Docker SQLite `TestVideoStatusPreservesUnknownAndOfflineCoverage` PASS；HTTP `TestVideoStatusHTTP`、`TestVideoStatusRequiresAuthentication` PASS。
- Docker OpenAPI 合同检查 PASS。
- Docker web 构建成功，包含 TypeScript 与 Vite。页面 22 项及 API adapter 定向测试通过。
- `make arch-check`、`npm --prefix web run generate:check`、`lint:openapi` 通过；OpenAPI 仍有两个原健康端点无 4XX 的警告，Vite 仍有大 chunk 警告。
- 首轮修复了 app 仓储接口遗漏、测试种子模型/代次成对约束、鉴权测试依赖缺项和 OpenAPI 3.0 空值写法；以上记录以最终重跑为准。

## 边界

本轮未重建最终运行镜像并启动浏览器验收，尚无完整 E2E 或全七项检查。
未新增 UI 视频回填/重建按钮；人物生产接入、逐资产失败归属、容量/并发完整矩阵仍未完成。
没有使用 server222 或真实媒体，也没有提交推送。
