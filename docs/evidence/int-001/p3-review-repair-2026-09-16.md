# P2/P3 复查后的修复增量

> 本文保留当时状态、失败及撤回记录；最新结论见 [P2/P3 全项复查](p2-p3-final-recheck-2026-09-16.md)：修复后最终固定镜像重新验收，P2 6/6、P3 9/9、18/34；不代表质量、容量或发布完成。

> 本页新发现的 A05 已在后续修复并复验；最新 A05/D03 关闭、D04 保留，17/34，见 [人物移动修复](face-reassignment-2026-09-16.md)。下文保留当时失败记录。

日期：2026-09-16；CR-2026-030 / FR-INT-007、011，POST-MVP-5 S3；Owner web AI review / SQLite face。
接续 [复查](p2-p3-review-2026-09-16.md)，不是全部完成声明。仅本地 Docker 与临时合成媒体，未部署或推送。

## 修复

- 逐项读取 HTTP 200 `items[].outcome`，不再把 conflict 当成功；提示已保存/未确认数量，仅移除成功选择，保留冲突项；刷新建议及人工整理 revision 后重试。版本未就绪、提交及刷新期间禁用重复提交。复用 curation query-key owner。
- 标签查询、查询键及 URL 携带 cursor，复用已有审核分页；刷新保留页，状态/库切换回首页，选择不跨页。suggestion_cursor_stale 重试回首页。
- 全冲突、混合结果、分页/状态切换回归通过；未改变 API 合同、数据库或模型阈值。

## 新发现，不掩盖

三浏览器非空人物测试：Chromium 创建/归类/移出后，Firefox/WebKit 把同一脸归入新人物均得到 face_constraint_conflict。
不是浏览器差异：SplitFace 将 anchor 设为 needs_review/current_face_id=NULL，保留 person_id；assignLineage 按同一 source/region 找到旧 anchor 后拒绝不同 person_id。“移出→归入另一人物”仍不可用，首次匿名归类通过不能替代移动验收。

**A05 重新打开，P2 5/6；D03/D04 仍打开，P3 7/9；总计 15/34，19 项未关闭。**
后续在 face/SQLite 唯一事务 owner 内补旧/新人物 revision、转移、cannot-link、撤销和重放合同，再修复；不能简单删除安全检查或用两个非原子 UI 请求伪装移动。
失败断言保留在完整浏览器探针；标签专项显式 grep 排除人物流程，专项通过不冒充完整探针通过。

## 实际测试

- 最终 Docker 全前端 **52 文件/233 单测通过**；生成一致性、类型、前端架构检查通过。
- 中途测试误用 Testing Library 的 exact 参数、Toast 不支持 warning 导致类型/首轮构建失败，已修正，不计作通过。
- 首轮扩展浏览器 **18 passed / 5 failed / 8 条条件 skipped**：三条是测试误读 aria-label（复选框由 label 文本命名，已改）；两条是上述实际人物缺陷，未修复、不删除。
- 最终应用 foliopath-app:p3-review-fix-final 构建成功，ID `sha256:02df320d21e1b2042a9617335684a58c1bf3dde90af4f0b52bb6b67f53ab9948`。
- 最终专项完整退出码 0：三浏览器标签/四断点设置 **6 passed**（其中设置包含 48 个主题/语言/断点组合），停服回退文件名检索 **3 passed**。每浏览器独立 52 条真实模型建议，经 UI 翻页/刷新、真实 favorite mutation 触发 curation revision 冲突、保留选择/刷新重试接受均通过。没有拦截 API 伪造结果。
- git diff --check、两个变更探针 node --check 通过，探针临时容器已清理。仍有既有 Vite 大 chunk 与单测 auth/session mock 警告，不把警告藏起来。
- 设置矩阵扩到 390/768/1265/1440；仍不等于人物/标签全部主题/语言矩阵、完整触控或视觉差异回归。D04 不关闭，VoiceOver 不恢复。

```sh
FOLIOPATH_P3_APP_IMAGE=foliopath-app:p3-review-fix-final \
FOLIOPATH_P3_TEST_GREP='real tag pagination|real settings responsive' \
node spikes/int001-remote-ml/p3-browser-probe.mjs
```

不设置 grep 仍执行完整探针（含尚未修复的人物重归类失败）。全仓库七项门禁、全部后端故障恢复本轮未重跑。
