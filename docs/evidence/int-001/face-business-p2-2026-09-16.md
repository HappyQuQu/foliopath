# P2 非空人物业务闭环验收

> 后续曾发现移出后无法归入另一人物并重新打开 A05，现已修复并经最终 Docker 业务/三浏览器重新验收；P2 6/6。以 [最新修复记录](face-reassignment-2026-09-16.md) 为准，以下为历史证据。

2026-09-16 后续复查：以最新 P3 应用镜像完整重跑本页业务探针退出码 0，P2 六项保留；当前总进度以 [P2/P3 复查](p2-p3-review-2026-09-16.md) 为准，下文 9/34 是当时历史进度。

日期：2026-09-16。范围：CR-2026-030 / FR-INT-009 / NFR-INT-002、004；POST-MVP-5 S2，P2 A04～A08、D02。
Owner：face / SQLite / app / API / web。状态：**最终固定镜像完整复验退出码 0，P2 六项关闭**。当前剩余清单 9/34 关闭；不是质量或发布验收完成。

## 修复与合同

- 正式 app 组合非空 observation、匿名组和人物人工审核，不用 mock 替代后端。
- OpenAPI 补齐已有六种 `action` 的 discriminator mapping，生成客户端不再错误发送 schema 类型名；补齐已在 jobs 合同声明的 `library_offline` 枚举。没有新增端点或扩大作用域，revision digest 经复核更新。
- 分页失效返回既有 `face_cursor_stale`；UI 重试回首页，不反复提交失效 cursor。
- 自动整组归类不仅 UI 禁用，app 服务端也拒绝；质量门禁由 face owner 唯一持有。
- 派生清除后的人工 anchor/exclusion 精确唯一重绑；换代分批 reconciliation 保留 excluded 状态及 cannot-link，不跨库或依靠向量猜测身份。
- 保留按库已发布代次的可见结果，目标代次只用于新任务。已有活动任务/清理时禁止重定向。
- 人工清理先删除有界 undo snapshot，再删除无引用的审核叶节点，避免 RESTRICT 外键失败；无进展时失败，不无限循环。
- UI 新建首个人物、名称搜索目标人物、有界 URL 分页、逐张审核的新鲜 revision、未就绪/离线/冲突/空状态、清空二次确认与焦点恢复。修正人工清除警告：人物名称保留，不声称会被删除。截图复查发现匿名组的整张合照无法辨别选的是哪张脸，补上合同 region 对应的可见位置框，再复验。

## 本地测试环境与边界

全程本机 Docker Linux/arm64，不访问 server222。主应用只读挂载临时合成双库，AI 无媒体/数据库挂载且无公开端口。模型缓存只读，使用既有 buffalo_l 与 CLIP；应用数据为独立临时 volume。

非空 fixture 由 imagegen 技能调用内置生成工具制作，包含两位虚构成年人；完整提示词及来源见 `tests/fixtures/ai-faces/README.md`。每库复制两张，每张实际检出两张脸，形成两个二成员 core 组。重复图只证明非空链路，**不证明检测召回、聚类 precision/recall 或 ≥99.5% 整组归类质量**。

镜像 ID（不是对外发布 digest）：

- app：`sha256:992fd5a6d817230d9bae440b07ec3f3bb4376ecb6cb5039da47e62df1bec6682`
- AI：`sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`
- race regression：`sha256:f65cf38452e08f9dc01e786cc3996c56a25a35619f4166a8632c20a15d9def36`
- Chromium runner：`sha256:63eebf47ef9bf5115bf6bc7fc934fd222480ed82b1c5d7e2bd8e38ee403839ac`

## 可复跑入口

构建和夹具说明见 `spikes/int001-remote-ml/README.md`。

```sh
docker run --rm --network none foliopath-remote-ml-regression:local \
  '/out/app.test -test.run Face -test.timeout=3m && /out/face.test -test.timeout=3m && /out/api.test -test.run Face -test.timeout=3m && /out/sqlite.test -test.run "Face|People|Person|CannotLink|GuardedUndo" -test.timeout=8m && /out/contract.test -test.timeout=3m'
node spikes/int001-remote-ml/face-business-probe.mjs
```

探针驱动正式认证 HTTP API；`web/tests/e2e/face-real.spec.ts` 仅在探针提供隔离环境时执行，浏览器不拦截 API。退出清理自己的容器/网络/临时媒体/数据卷，截图位于忽略目录 `test-results/face-real`。

## 实际验证记录

- `make fmt`、`make fmt-check`、`make arch-check`、`git diff --check`：通过。
- `make contract-check`：通过。首次 Docker contract 检查发现 schema digest 未更新，复核上述兼容修正后更新并复验通过；不是跳过锁定测试。
- Docker Go race：app Face、face 全包、api Face、SQLite `Face|People|Person|CannotLink|GuardedUndo`、contract 全包均通过，最后组合命令退出码 0。
- Docker 前端：生成一致性、架构、TypeScript、AI/adapter 5 文件 **55 测试通过**。新增清除确认测试的 mock 未返回 operation 导致一次未捕获错误，已修正 test fixture 并确认退出码 0。
- 额外运行 Docker `npm run test` 全前端单元测试：**52 文件 / 226 测试通过，退出码 0**。认证路由测试仍打印一条 `auth/session` mock 返回 undefined 的 TanStack 提示，未将其隐藏，也不将本次单元测试等同完整 E2E/发布回归。
- OpenAPI lint：通过，保留健康端点没有 4xx 的两条既有 warning；Vite build 通过，仍有大 chunk warning。
- 真实 API 已通过逐脸归类、人物投影、排除/拆分/合并及撤销、cannot-link 拒绝、服务端整组门禁、两类重建、禁用/启用、派生清理重建后人工保留、代次切换/取消/旧结果可读、AI 离线恢复。
- 实际 Chromium 已通过非空组、创建/改名、单脸归类/移出、双脸连续归类、人物分页、刷新、移动宽度检查。新增双库清空弹窗测试首次因 locator 同时命中两库而失败，已按目标库定位，完整探针退出码 0。
- 该完整探针还验证了两类清理后的原图 SHA-256、正式人工标签保持不变；媒体库离线保留索引，恢复后仍有结果；修改临时图的 mtime 后重新扫描/补齐不自动继承人工归类，移走临时图后扫描/重建移除派生状态，另一库不变。仅在临时合成树执行这些变更。
- 人脸位置标记已在固定镜像 `4e34357d…` 完整复验通过（含 API/浏览器/源变化所有步骤）。此前仍运行旧镜像的一轮对新增位置框断言失败（0 个而非 2 个），不算通过；未将中途重建 tag 的测试当作最终镜像证据。最后补齐从真实库状态区分离线与未就绪，上述 `992fd5a6…` 镜像完整探针再次退出码 0，正常及离线两轮 Chromium 都通过。
- 最终探针覆盖强杀主应用后读取已提交人物关系；换代只修改测试 profile，按库重建，不声称跨模型质量等价。活动任务强杀/租约恢复沿用本轮之前 P1 最终镜像证据，不用这里的静态持久化冒充活动任务恢复。
- 桌面匿名组、移动人物详情截图已人工查看；匿名成员按真实 `region` 标记具体人脸。最终运行产物在 `test-results/face-real/workflow` 和 `test-results/face-real/offline`，不提交运行日志、trace 或图片缓存。

最终探针成功输出：

```text
PASS first start offline: core ready, no unverified models
PASS CLIP and face active
PASS real nonempty inference, observations, two clusters per library
PASS individual assignment, projections, exclusion/split/merge undo, cannot-link, server group gate
PASS missing/all rebuild, disable/enable, derived clear preserves manual relationships and cannot-link, library isolation
PASS generation retarget, cancel, old per-library visible results, exact manual rebind, offline reads and recovery
PASS browser nonempty face workflow
PASS manual clear limited to requested library; original synthetic files remain
PASS offline media root preserves index and other library, restoration retains results
PASS changed source requires review, removed synthetic asset reconciles, other library/manual tags preserved
```

## 不能据此声称完成

模型质量校准/困难样本真值、100k 容量、原生 amd64、混合故障全矩阵、三浏览器/双语言全矩阵、正式 Compose 交付、全仓库七项发布检查与签署均不在本轮通过结论内。P2 功能完成不等于 P4 质量或 P6 发布完成。
