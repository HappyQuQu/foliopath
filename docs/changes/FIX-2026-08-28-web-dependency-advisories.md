# Web 依赖高危 advisory 修复

- 日期：2026-08-28
- 状态：**Implemented / verified**
- 类型：既有 Web 构建与运行依赖的例行安全维护
- Requirement：`NFR-COMP-001`、供应链高危 finding 必须处置
- Owner：Web dependency/lockfile 与 release security
- 关联 Gate：[CUR-S4 Integrated Slice Current](../gates/POST-MVP-4/cur-s4-integrated-slice-current.md)、
  [INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)

## 修复

完整 `web-check` 在 dependency audit 阶段识别到 `react-router`、`js-yaml`、`brace-expansion`、
`nanoid` 和 `undici` 的当前 high advisory，以及 `postcss` 的 moderate advisory。修复只采用兼容补丁：

- `react-router-dom` / `react-router`：`7.18.1` → `7.18.2`；
- override `brace-expansion`：`5.0.8` → `5.0.9`；
- override `js-yaml`：`4.3.0` → `4.3.1`；
- override `nanoid`：`3.3.18`；
- override `postcss`：`8.5.26`；
- override `undici`：`7.29.0`。

未改变 API、路由结构、AI scope、媒体边界或产品交互。lockfile 由 npm 重新解析生成，没有手工修改。

## 验证

- `npm install --package-lock-only --ignore-scripts`：`found 0 vulnerabilities`；
- `npm ci --ignore-scripts`：`found 0 vulnerabilities`；
- `make web-check`：无适用 high advisory，生成/架构/视觉引用/类型检查通过，47 个测试文件、
  167 个测试通过，Storybook 构建成功；
- `make test-web-e2e`：依赖升级所需的锁定 Chromium 安装后，真实后端纵向链 7 passed、4 skipped；
  harness 同步修复见[真实浏览器 E2E harness 与当前界面合同同步](FIX-2026-08-28-browser-e2e-harness.md)；
- `make openapi-lint`：通过，保留两个既有 health 4xx warning；
- `make contract-check`：通过。

本修复解除 CUR-S4 的 dependency-advisory 子阻塞；目标浏览器/真实触摸人工签署仍未完成。
它只维护 S2A 的合同消费者验证，不改变 INT-S2A No-Go。
