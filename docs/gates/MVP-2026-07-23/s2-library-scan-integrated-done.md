# Stage 2 媒体库与扫描前端 Integrated Done

## 结论

**Go — Stage 2 媒体库与扫描纵向切片 Integrated Done。**

媒体库目录选择、创建、列表、改名、异步安全移除、扫描状态/取消/重试、定时扫描和缓存配额
已经通过生成客户端连接真实后端，并完成成功链、故障状态、可访问性、响应式和设计一致性证据。
该结论允许前端进入 Stage 3 浏览与缩略图，不表示完整 FolioPath 或发布版本已经完成。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- 需求：`FR-LIB-001～008`、`FR-SCN-001～009`、`FR-UI-001～007`、
  `NFR-SAFE-001`、`NFR-SEC-001～002`、`NFR-PRIV-001`、`NFR-ACC-001`
- 前序 Gate：[媒体库 Backend Ready](s2-library-backend-ready.md)、
  [扫描 Backend Ready](s2-scan-backend-ready.md)
- 权威契约：`api/openapi.yaml`
- 前端 owner：`web/src/features/libraries`、`web/src/features/settings`
- API adapter owner：`web/src/lib/api`
- 共享交互 owner：`web/src/components`、`web/src/lib/useSubmissionGuard.ts`

## 验收判断

| 判断项 | 证据 | 结论 |
| --- | --- | --- |
| 真实纵向成功链 | 一次性 SQLite、只读合成 `/library` 和真实 Go 进程执行 setup → 两层长路径建库 → 改名 → 手动扫描 → 设置 → logout/login → 异步移除 | 通过 |
| API 与状态所有权 | 生产页面只经生成 client 和领域 adapter；ETag、CSRF、幂等键、Query key、错误映射和轮询各有唯一 owner | 通过 |
| 安全与隐私 | 路径选择只显示 `/library` 相对路径；移除明确不触碰原媒体；失败/离线保留可靠索引；状态页不显示宿主路径或原始诊断 | 通过 |
| 状态矩阵 | 真实成功状态结合评审契约 fixture 覆盖 loading、error/retry、running、cancelling/cancelled、failed/部分不可读、offline 和异步 removal | 通过 |
| 重复提交 | 创建、改名和设置保存快速双击各只产生一次请求；共享同步 guard 覆盖移除、扫描和取消，组件回归测试固定锁释放语义 | 通过 |
| 长内容与响应式 | 128 字符库名、两层长目录、390/768/1024/1440px 无页面级横向溢出；移动长名称扫描状态实拍为 375/375 | 通过 |
| 键盘与可访问性 | Enter 打开改名、Escape 关闭并恢复触发按钮焦点；语义 dialog/status/progress；Chromium axe serious/critical 为零 | 通过 |
| 主题与语言 | 浅/深主题、简体中文/英文、`html[lang]` 和四档布局沿用唯一 provider/token owner | 通过 |
| 设计一致性 | `web/design-qa.md` 对照原型 10～14；桌面并排与移动长内容/对话框/状态证据完成，最终结果 passed | 通过 |
| CI 固化 | `Authentication and library browser E2E` job 使用锁定 Chromium 和一次性真实后端执行 `make test-web-e2e` | 通过 |

## 自动证据

- `web/tests/e2e/auth.spec.ts`
- `tests/e2e/web_auth.sh`
- `web/src/lib/useSubmissionGuard.test.tsx`
- `web/design-qa.md`
- `web/qa/stage2-library-comparison-1440.jpg`
- `web/qa/stage2-settings-comparison-1440.jpg`
- `web/qa/stage2-long-name-status-mobile.png`
- `.github/workflows/ci.yml`

本地实际执行并通过：

```text
npm --prefix web run check
npm --prefix web run build
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make test-web-e2e
```

## 保留限制

- Stage 3 仍负责真实目录树、资产分页/虚拟化、缩略图和非模态预览；Stage 4 负责搜索与完整查看器。
- Stage 5 仍负责 Firefox/Safari、最终 Chrome 版本矩阵、发布镜像、可信代理、网络配置和发布级
  视觉回归。
- 受控扫描故障 fixture 只证明前端按冻结契约呈现并操作这些状态；后端故障、恢复和容量事实由
  `S2-105～107` Backend Gate 证明，两者组合构成当前 Integrated Done。
- 本 Gate 不宣称 10 万媒体的前端浏览性能、完整产品可用或稳定 MVP 可发布。

## 交接

- 后端：媒体库与扫描 Backend Ready。
- 前端：媒体库与扫描 Integrated Done。
- 允许的下一步：`S3-101` 桌面侧栏、移动抽屉、目录树、面包屑和可恢复 URL。
- 评审日期：2026-07-28。
