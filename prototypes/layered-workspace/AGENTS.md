# Prototype Instructions

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.

## 本原型已知用户方向（2026-10-23）

- 保留现有信息架构，增强分区背景、文字层级、状态与精致感。
- 用户要求所有功能可查看：以功能导览串联媒体、整理、智能分析、管理及系统设置。
- 所有素材和动作显式为合成演示；该原型不能进入生产导入图。

## 功能侧栏方向（2026-10-24）

- 用户要求搜索、人物、收藏、标签时，原目录区域切换为各功能工具。
- 主导航位置保持稳定，下半区提供查找、范围、分组、类型与标签；主区域聚焦结果。
- 全局已建立人物、本库未命名组与本库收藏不能混淆。手机入口随功能命名，侧栏可关闭且独立滚动。

## 侧栏接受与归档（2026-10-24）

用户已授权将四类侧栏实施到正式界面。此原型中的侧栏仅保留历史接受证据，不再继续修订或作为第二套实现；后续修改应改生产组件、Storybook、规范与测试。其他导览不因这次侧栏接受自动转为生产需求。

2026-10-25：用户进一步接受整体分层视觉用于正式界面。此原型整体归档；后续视觉修改同步中央tokens、正式组件、Storybook、UI规范和回归，不继续维护重复屏幕。
