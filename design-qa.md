# 智能分析配置作用域视觉复核 — 2026-09-22

final result: passed

## 范围与证据

- 来源视觉真相：`/var/folders/21/hjbt152d0vs3n8td7233xd8r0000gn/T/codex-clipboard-c4ad92dd-c88f-4a4c-a2be-8ca9f2be5cd3.png`（4592×2242）。来源用于确认原页面中“当前媒体库”作用域不清晰的问题，不是像素级复刻目标。
- 桌面实现：`docs/evidence/prototypes/ai-settings-config/implementation-desktop.png`（1130×1378 px），CSS 视口 1145×1230，DPR 2；全页捕获会按浏览器可用内容宽裁到 1130px。
- 移动实现：`docs/evidence/prototypes/ai-settings-config/implementation-mobile.png`（375×1884 px），CSS 视口 390×844，DPR 1，可用内容宽 375px。
- 并列比较：`docs/evidence/prototypes/ai-settings-config/comparison-desktop.png`（2400×1464），来源与实现各归一化到 1200px 宽后并列；两者主题与视口不同，比较信息层级、作用域和密度，不做像素差异评分。
- 原型 URL：`http://127.0.0.1:6007/iframe.html?id=features-intelligence-settings--configuration-prototype&viewMode=story`。
- 状态：浅色主题；智能搜索展开，人物、任务与全局服务默认收起；另行展开人物与全局服务完成交互检查。

## Findings

- 没有剩余 P0/P1/P2。原来的单一“媒体库设置”已拆为“当前媒体库的功能”和“全局 AI 服务”，切库控件只属于前一分区。
- 实现有意不展示来源截图中的硬编码模型名或后端未公开阈值。原型直接渲染生产组件，并以页面底部说明标记合成数据；正式运行时只消费 API 返回的模型、安全元数据和逐库状态。
- 移动视口 `scrollWidth=375`、`clientWidth=375`，没有横向溢出；折叠项、选择器、开关和主要操作均在内容宽内。

## 五项视觉检查

- 字体与排版：复用 FolioPath 字体、字号、字重和标题层级；两级分区标题和资源行标题可快速区分作用域，无异常截断。
- 间距与布局节奏：桌面使用侧栏后的全部可用宽度，资源行单列排列；移动端分区、选择器和卡片转为纵向，间距连续。
- 颜色与 token：只使用中央背景、边框、强调、成功与警告 token；未复制参考产品外观。
- 图像与图标：页面无需位图；所有功能图标和 disclosure caret 均来自既有 Phosphor 图标库，没有文本符号或自制 SVG 替代。
- 文案与内容：逐库开关/进度/维护和全局服务/模型文案保持一致；未公开配置明确写为部署管理，不伪造可保存行为。

## 交互与运行检查

- 人物分析与服务模型折叠项可展开，展开后“查看人物”、语义模型和人物模型均可见。
- 当前媒体库选择、智能搜索开关、维护入口和任务入口具有可识别名称。
- 浏览器 console error 为 0；窄屏无横向溢出。
- TypeScript、AI 设置 29 项单测、全前端 234 项单测和前端架构检查通过。

## 比较历史

- 较早原型把逐库功能、维护与全局服务放在一个连续列表，且维护了独立原型组件，存在作用域误解和实现漂移风险。
- 修复：将任务维护归入当前媒体库；新增独立全局服务分区；删除独立原型组件，让接受态 Storybook 直接渲染生产 `AISettingsPage`。
- 修复后证据：上述桌面、移动与并列比较图；交互展开检查、无溢出检查和 console 检查全部通过。

## Open Questions

- 当前公共 API 不提供服务 URL、模型包名称或人脸质量阈值，因此生产页面不展示这些值。将来只有在合同正式增加安全字段后，需求、原型、代码和测试才能同步加入。

## Implementation Checklist

- [x] 逐库功能与全局服务分区。
- [x] 原型复用生产组件并标记合成数据。
- [x] 桌面、移动、折叠交互与 console 检查。
- [x] 需求、用户流程、UI 规格、AGENTS 和测试同步。

## Follow-up Polish

- 无阻断项。若公共模型合同未来增加质量配置，可在全局服务折叠区补充只读行，不应放入媒体库区。
