# 滚动加载目录与媒体列表

2026-10-29 用户要求目录和列表的“载入更多”支持触摸下滑、鼠标滚动自动加载，截图包含目录树分页和媒体下一页失败状态。

- 范围：POST-MVP-5 既有 FR-BRW-001/004/005、FR-UI-001～003/006；复用已实现的目录/媒体游标分页合同，无API或数据库变化。Backend Ready沿用既有浏览与搜索分页Gate。
- Owner：共享 `useScrollPagination` 拥有滚动容器检测、底部触发、请求合并与失败后的用户手势重试；MediaCollection保留原有虚拟窗口预加载判断；browse消费同一hook供目录树和展开后的目录网格使用。失效游标恢复仍只由 `retryInfiniteNextPage` 拥有。
- 行为：正常接近已加载内容底部自动请求下一页，鼠标/触摸/键盘滚动均可触发；目录网格保留默认两行折叠，展开后才自动分页。下一页失败保留已有内容、错误提示及按钮；再次向下滚动可重试，同一滚轮/触摸手势不反复请求。没有新手势时不自动重试错误。保持游标分页和媒体虚拟化，加载中不并发请求同一列表。
- 状态：2026-10-29 本地实现与预览验收完成，未提交、未发布、未更新222；与本分支未发布的7z实现一同保留。普通测试/截图/输出依旧本地专用，不扩大25项发行白名单。

## 本地验证

- `cd web && npm run check`：通过。包含高风险依赖审计、生成一致性、前端架构、视觉清单、TypeScript、80个测试文件/546项测试和Storybook构建。Storybook保留已有大chunk提示，构建成功。
- `make arch-check`：通过；`git diff --check`：通过。本轮未修改Go业务逻辑，未重复运行完整Go/integration/e2e目标；浏览器实测结果如下，不冒称全套E2E通过。
- 生产hook回归覆盖请求合并、最近滚动容器、快速越过分页控件、鼠标/触摸/键盘重试、输入控件与其他面板隔离；生产MediaCollection回归保持预览时暂停自动追加、关闭后恢复；目录下一页失败回归确认旧目录仍可访问。
- 稳定生产组件Story：MediaCollection `ScrollPagination` 明确标注合成数据并演示首次追加失败/重试；BrowsePage `ManyDirectories` 使用合成分页目录，覆盖默认折叠/展开与后续自动加载。
- Linux arm64生产镜像构建成功；本地6020预览使用独立合成媒体/数据，媒体单挂载只读。当前镜像ID为 `sha256:63e727a5836ff4ed2a9ba8ad473a80c6d4b42ea7fcee7e99a8504ddae8c4cc55`，容器healthy，容器内 `database-check` 通过。
- 实际IAB浏览器：125个合成空目录、150张合成PNG。目录树50→100→125、展开目录网格快速滚动50→125、媒体50→100→150均无需点击分页按钮。分别阻断目录/媒体带cursor的下一页请求，旧内容保留；解除阻断并发出新的向下滚动后恢复加载，错误消失，最终全部载入。阻断已移除。
- 390/768/1440/1920×1000响应式截图与DOM几何验证无横向溢出；临时viewport已恢复。触摸手势由真实hook事件测试验证，未宣称真机验证。
- 本地证据：`/tmp/foliopath-scroll-final-check.log`、`/tmp/foliopath-scroll-arch-check.log`、`/tmp/foliopath-scroll-preview/image-build.log`、`/tmp/foliopath-scroll-preview/evidence/`。这些输出不入库。
