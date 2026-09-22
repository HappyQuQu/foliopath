# P5 F02 模型故障与恢复验收

2026-09-17；POST-MVP-5 / CR-2026-030，沿用 ADR-0019/0020。F02 本轮通过；F03/F04/G02 仍未关闭。没有生产代码、API、代次规则或阈值默认值变更。

## 环境与命令

本地 Docker arm64，实际根 `compose.ai.yaml`，项目 `fp-p5-d2a02c70`。合成渐变图及虚构人脸图两张，独立临时数据库；复制模型缓存只读，故障只修改临时缓存副本。AI 仅私网、无模型下载。原媒体与 server222 未使用。

- 主应用 `sha256:6325266035ffb7329d8e11ada7d84e1497dd8bc4ae5d407a6caa5affa52d09d3`。
- AI `sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`。
- runtime/cache 身份与 [F01 固定候选](p5-compose-2026-09-17.md) 相同。人脸阈值为显式 `p5-synthetic-only`，不是质量校准或生产推荐。

执行 `FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache node tests/release/ai_compose_smoke.mjs --faults`，完整重跑退出 0。

## 结果

| 场景 | 故障断言 | 恢复断言 |
| --- | --- | --- |
| 空模型缓存 | AI 校验失败退出/重启；两个活动模型 unavailable；图片/人脸新任务返回 409 及对应安全错误码 | 恢复原缓存并重建 AI，两个原模型再次 available，语义与人脸任务均成功 2/2 |
| 等长损坏文件 | 复制镜像清单第一项到临时目录，仅翻转一个字节；AI 不就绪，两个模型不可用，新任务拒绝 | 换回可信缓存，原模型/代次恢复，两种分析均成功 2/2 |
| 预期身份不匹配 | AI 正常运行，主应用配置两个模型的错误 runtime hash 后重建；旧模型不误报可用，新任务拒绝 | 恢复配对 JSON 并重建主应用，原模型/代次恢复，两种分析均成功 2/2 |

每次故障均检查目录资产 ID、非空人脸结果和语义覆盖率保留。semantic 离线 `activeGenerationId=null` 按既有准入合同验证，恢复后必须等于原 ID；face 记录代次在故障期间保持。每次重新分析后刷新派生结果基线，避免把合法重新生成的观察行当作故障丢失。结束核对两张合成原件 hash 不变，重启管理员仍可登录。测试容器、网络、临时数据库和坏缓存均已清理，只删除测试自产数据。

## 首轮失败及修正

首轮测试将 semantic 离线代次字段误当作必须非空，与已有 [离线覆盖率合同](image-offline-coverage-2026-09-15.md) 不符，断言失败并清理环境。修正为“离线字段 null、覆盖率保留，恢复后核对原 ID”，未更改产品实现；随后完整矩阵通过。保留此记录，不将失败首轮记为通过。

## 其他实际检查

- 本地 Docker、无网络：AISettingsPage 专项 **31/31**，覆盖服务不可用/离线时的操作门禁；这是组件测试，不是本轮真实浏览器 E2E。
- 固定 AI 镜像、本地 Docker 无网络：`unittest discover -s /opt/foliopath-inference -p 'test*.py' -v` **13/13**，含快照独立性、错误清单、符号链接、HTTP/人脸合同。
- `make arch-check`、两份测试脚本 `node --check`、`git diff --check` 通过。

只验证固定候选及选定故障；不是任意模型支持、准确率验收、原生双架构复验、G02 全部安全复核或 P6 七项完整检查。该 Compose 不执行模型下载，下载代理不适用于本轮 F02。
