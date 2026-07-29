# VSP-301 视频故事板真实产品纵向链

## 结论

**Done — 真实生产镜像已贯通扫描、派生、浏览与搜索 hover。**

本记录完成 `VSP-301`，但不替代 `VSP-302` 的原生 linux/amd64 与 linux/arm64
最终候选复验，也不把 `FTR-VID-001` 标记为 Integrated Slice Done。

## 环境

- 生产 `Dockerfile` 的 distroless final image；
- Linux arm64 容器，4 CPU / 4 GiB；
- read-only root、只读 `/library`、非 root、capabilities 全部移除；
- 真实 10 秒 MP4 fixture，原文件 chmod 0444；
- 真实 SQLite、`internal/files`、FFmpeg、durable jobs、cache、认证 API 和嵌入 React SPA；
- 主机 Playwright Chromium 通过发布端口访问容器，不使用 Vite proxy 或 API mock。

## 贯通流程

`make test-storyboard-vertical` 成功执行：

1. API setup/login 后创建根媒体库并触发真实扫描；
2. 等待 grid poster 与 10-frame/5×2 storyboard 后台 ready；
3. 验证未认证 401、认证 200、强 ETag/304、immutable/nosniff 与 1600×360 WebP；
4. 浏览器用独立会话真实登录；
5. 浏览页先显示 poster，200 ms 内无 storyboard 请求，300 ms intent 后才进入 playing；
6. 移出恢复 poster，单击打开非模态视频预览，关闭后恢复卡片焦点；
7. 搜索 `clip` 得到同一真实资产，并复用同一 hover controller；
8. 定位并删除临时测试 cache 树中唯一 1600×360 storyboard，HTTP 先返回 202，worker
   重建后恢复 200；
9. 最终重新校验原 MP4 SHA-256 与 mtime 完全不变。

cache deletion 通过扫描临时测试 cache 并用 `ffprobe` 验证唯一尺寸后删除；测试不再用外部
SQLite CLI 接触运行中的 WAL 数据库。

## 故障与降级覆盖

- cache missing/rebuild：本纵向链真实覆盖；
- decode failure、pending/failed、页面隐藏、卡片回收、reduced-motion、touch/粗指针：
  由 [VSP-S3 Consumer/UI Ready](vsp-s3-consumer-ui-ready.md) 的组件和多浏览器矩阵覆盖；
- offline、源变化、worker restart、ENOSPC 与取消：由
  [VSP-S2 Backend Evidence Ready](vsp-s2-backend-evidence-ready.md) 的真实后端矩阵覆盖。

## 剩余阻断

- `VSP-302`：同一最终源码状态的原生 linux/amd64 与 linux/arm64 完整候选复验；
- `VSP-303`：在平台证据后收敛版本 manifest、release notes 与全部“计划”措辞；
- `VSP-304`：聚合 `VSP-AC-001～008` 并作最终 Go/No-Go。
