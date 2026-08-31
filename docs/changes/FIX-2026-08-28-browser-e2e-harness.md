# 真实浏览器 E2E harness 与当前界面合同同步

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：既有浏览器纵向验收基础设施与断言维护
- Requirement：`NFR-TEST-001`、原媒体只读不变量、已接受的管理中心与媒体库状态交互
- Owner：Web E2E / release verification
- 关联 Gate：[CUR-S4 Integrated Slice Current](../gates/POST-MVP-4/cur-s4-integrated-slice-current.md)、
  [INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 修复

`make test-web-e2e` 原先额外拉取 `alpine/socat` 镜像，只为把应用容器内的回环监听端口转发到
宿主机；该额外拉取点因 Docker registry 凭据失败而使测试在浏览器启动前退出。测试专用应用镜像
现直接安装 Debian `socat`，代理容器复用同一临时镜像并覆盖 entrypoint。生产镜像、应用监听策略和
受信代理边界均未改变。

纵向用例同时与当前已接受界面合同同步：

- 媒体库卡片使用原位 `View status` 展开，并从实际 Browse URL 获取 opaque library ID；状态深链仍覆盖；
- 账户菜单使用 `Management Center`，配置页使用 `Configuration / 配置`；
- 扫描策略和缓存按当前 `Scan policy`、`Cache` tab 分别保存，双击仍各只提交一次；
- Theme 下拉框以明确 combobox role 定位，避免与 Header 快捷主题按钮歧义；
- 第二次设置保存等待真实 PATCH 响应后再移除 route，避免处理中的拦截器竞态；
- 综合纵向用例上限由 240 秒调整为 480 秒，只改变测试预算，不改变产品超时。

## 验证

- `make test-web-e2e`：Chromium / Pixel 5 项目共 **7 passed、4 skipped**；首次管理员、建库、扫描、
  浏览/搜索/预览、配置、账户、会话、媒体矩阵与真实故事板链通过；
- harness 末尾的媒体 hash 与路径 `cmp` 通过，确认只读 fixture 未被修改；
- `make web-check`：通过；
- `make arch-check`：通过；
- `git diff --check`：通过。

该修复提供真实浏览器回归证据，但 Pixel 5 是浏览器设备模拟，不替代目标浏览器与真实触摸设备人工签署；
也不提供最终 AI 模型、合法质量集、原生 amd64、100k 全进程或供应链证据。
