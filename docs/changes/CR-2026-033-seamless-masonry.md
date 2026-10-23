# CR-2026-033：无缝瀑布流

- 状态：Confirmed；用户于 2026-10-23 明确要求从 main 独立实现、合并并发布。
- 目标：Post-MVP 显示优化，下一次 minor（1.4.0）；不修改冻结 MVP 范围。
- 需求：`FR-BRW-014`；继承 `FR-BRW-006/013`、`NFR-ACC-001`、`NFR-PERF-002`。
- Owner：FolioPath maintainers；共享 `MediaCollection` 拥有布局，`preferences.ts`
  拥有浏览器偏好，Browse/GeneralSettings 拥有入口。
- Gate：复用 S3-103/S3-107 及既有 Backend Ready catalog 合同；纯显示增量不增加后端用例。

## 接受范围与合同

1. 增加 `seamless` 布局，中文“无缝瀑布流”、英文“Seamless masonry”。
2. 浏览工具栏可即时切换并记忆；通用设置可保存为默认，新打开的浏览、搜索、收藏和标签页
   复用同一偏好。既有 grid/masonry 值及 grid 默认保持兼容。
3. 沿用瀑布流的受限宽高比、查询 DOM 顺序、cursor 分页和虚拟化；图片间距为零，卡片
   无圆角、不浮起，不渲染文件名和修改时间栏。测量保留小数高度，避免四舍五入产生接缝。
4. 可访问名称保留文件名。预览、双击、键盘焦点、收藏、失败占位及分页重试沿用现有行为。
   聚合视图的来源目录链接改为图片内浮层，仍可操作；预览/查看器保留完整文件信息。
5. 不改变排序、API、数据库、原图、缩略图变换或部署拓扑；不引入 AI 分支内容。

发布前置修复：兼容升级 React Router、Vite、Storybook、js-yaml、brace-expansion 与锁文件
中的受影响构建依赖，消除既有 high/critical 审计失败。测试镜像使用独立 Docker ignore，
允许嵌入本地生产构建产物；正式镜像仍在自身构建阶段生成前端。无审计豁免新增。

## 控件与证据映射

| 可见控件或状态 | Requirement / 合同 owner | 验证 |
| --- | --- | --- |
| 无缝瀑布流按钮、通用设置默认值 | FR-BRW-014；本地 preferences v1 | Browse、GeneralSettings、preferences 单元测试 |
| 无文字栏、零间距卡片 | FR-BRW-014；生产 MediaCollection | Seamless Story、浏览器四档宽度几何回归 |
| 预览、收藏、来源、失败和载入更多 | 既有 catalog/curation 合同 | 共享组件回归、现有 Web 测试 |
| DOM 有界、键盘可操作 | NFR-PERF-002 / NFR-ACC-001 | 共享组件有界窗口、浏览器焦点检查 |

详细执行结果见 [验证记录](../evidence/seamless-masonry-2026-10-23.md)。
