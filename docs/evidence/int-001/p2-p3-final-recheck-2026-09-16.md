# P2/P3 全项复查与修复验收

范围：CR-2026-030 / POST-MVP-5 S2/S3，既有 P2 六项及 P3 九项；Owner face/SQLite/app/API/web。
用户要求复查、修复后统一汇报。最终固定镜像的全部下列探针退出成功：P2 6/6、P3 9/9；D04 本轮关闭，剩余清单 18/34 关闭、16 项保留。此结论仅覆盖既定 P2/P3，不代表全部项目或发布完成。下文保留执行顺序与失败记录。

## 验收映射

| 工作包 | 本轮固定镜像验证入口 | 状态 |
| --- | --- | --- |
| A04/A05/A06/A07/A08、D02 | face-business-probe + SQLite/face/app/API race + 人物真实浏览器 | 本轮通过 |
| B05/C01/C02 | p3-business-probe + semantic/resourcecontrol/SQLite race | 本轮通过 |
| C03/C05 | p3-recovery-probe + recovery_smoke + upgrade_rollback_smoke | 本轮通过 |
| C04 | p3-fault-probe + recovery_smoke | 本轮通过 |
| D01/D03 | p3-browser-probe 在线、离线及 52 条真实建议冲突/分页 | 本轮通过 |
| D04 | d04-real 三浏览器 5 类页面 × 两语言 × 两主题 × 四断点、键盘、移动触控、固定视觉基线独立复跑 | 本轮通过 |

## 修复与保留边界

- 人物创建进入详情、返回列表及审核分页在旧聚焦元素卸载后，恢复到智能整理标题；不抢占仍存在的用户焦点，不对搜索输入的每次 URL 更新抢焦点。
- 新增 D04 真后端验收，不用 mock 的触屏媒体查看器替代 AI 人物/标签流程。移动设备模拟不等同物理设备。
- 视觉参考生成、人工检查和独立对比复跑分别记录，首次写入参考不算视觉回归通过。
- 整组归类质量门禁仍保留；P4 质量/容量/双架构和 P5/P6 发布不纳入完成声明。无 VoiceOver、无远程部署、无提交推送。

## 执行记录

- 初始前端生成/类型/架构检查、52 文件 / 237 单测通过。
- 候选应用 `foliopath-app:p23-audit`：`sha256:55cb955b7e1bead0c1027f0da51e410751faff2188171ff12cf3e2ea8b35f8ca`。
- 候选回归镜像 `foliopath-remote-ml-regression:p23-audit`：`sha256:a982fadc26a5988dbcf170e173b081006a66c112dc39c6e58833fab21774e70c`。
- 最终状态必须以之后实际完成的结果更新，不能只凭本表有入口就关闭。
- 最终应用 `foliopath-app:p23-final`：`sha256:a9a444a960d58b70325ec6a9b3263d8211de09fcba26d3e123843c6d2ae62717`。补齐移出最后一张媒体后的返回按钮焦点，以及人物卡片文件名/数量分行和中英文数量文案。
- 最终源码 Docker 前端生成、类型、架构及 52 文件 / 237 单测通过；Go SQLite 选定业务、face、semantic、resourcecontrol、app 选定组合、API 和 contract 七组 race 回归通过。Go 回归镜像与最终应用后端源码一致，后续差异仅为前端；不等同 G01 全仓七项验证。
- 最终应用 `recovery_smoke.sh`、`upgrade_rollback_smoke.sh` 通过。旧镜像 `sha256:992fd5a6d817230d9bae440b07ec3f3bb4376ecb6cb5039da47e62df1bec6682`；配对升级前备份 SHA-256 `138bde45d11d5c8ea01ebceb2d2a31057230c553843e208b16687be0b32b3fee`。
- 初次扩展 D04：Chromium/移动 Chromium 通过；Firefox 暴露移出后焦点丢失，WebKit 长页面截图超过 32767 像素上限。分别补齐生产焦点恢复及调整截图范围；初次失败不计为验收通过。初版 locator 截图受固定页头裁切影响，已废弃，改为页面顶部截图。
- 最终镜像视觉参考生成轮：在线 23 通过、8 条件跳过，D04 四项目全部通过，离线三浏览器 3 通过；32 张参考生成完成。检查了覆盖两语言、两主题及四断点的六张代表截图；首屏与完整人物页无原先固定页头裁切及元信息粘连。参考独立比较仍在进行。
- 独立比较首次启动早于上一轮清理退出，6095 端口占用导致启动失败；已移除这一失败实例自己的 Created 容器、空数据卷、网络和临时镜像标签，未操作其他环境。等上一轮完整退出后新建环境重跑，不计启动失败为通过。
- 最终镜像 `face-business-probe.mjs` 完整通过（含非空、审核/撤销、重建/清理、代次、模型/库离线、来源变化及独立 Chromium）；`p3-business-probe.mjs` 完整通过（B05、C01、C02）。故障与恢复探针继续运行。
- 最终镜像 `p3-fault-probe.mjs` 完整通过：缺失/同大小损坏模型、快照盘满/只读、128 MiB AI 实际 OOM 均拒绝假可用、保留旧索引和人物，并在恢复后重新成功分析。
- 独立视觉比较轮：在线 23 通过、8 条件跳过，D04 四项目通过；32 张参考在新数据卷、新人物 ID 的环境中以只读方式比较通过（像素差异上限 0.1%，人物随机 ID 局部遮罩）。三浏览器离线及最终 C03/C05 仍待本轮结束。
- 独立比较轮随后离线 3 通过、服务恢复成功、探针退出 0。8 条件跳过为在线轮的 3 条离线用例、3 个桌面项目中的移动专用用例及移动项目中的 2 个桌面专用用例；离线和移动各自在对应阶段执行，不是 8 个功能未测。设置 48 + 审核 240，共 288 页面状态经过 axe 与水平溢出检查；并非 288 条独立测试。媒体 mock viewer 与真实 AI 流程分开，runner H.264/AAC 不可用时只证明降级，不证明实际播放。
- C03 最终长测已通过正常停机、SIGKILL 后真实租约恢复及进行中的推理连接中断/服务重启恢复；人物、人工标签和原图哈希保留。继续等真实 HTTP deadline 和最终备份恢复，不改数据库时间。
- 最终 C03/C05 全部通过：真实 HTTP 超时记失败而非用户取消，恢复允许重试；错误部署身份失败关闭、无跨代次写入；停机卷备份恢复人物/人工标签，清除派生数据再重建仍保留精确人工关系，原图哈希不变。完整业务→故障→恢复命令链退出 0。
- 最后 `make arch-check`、`make fmt-check`、`git diff --check` 通过。没有执行并宣称整仓 `make fmt/generate-check/lint/test/test-integration/test-e2e` 七项全量通过；G01 仍未关闭。

### 复跑入口

```sh
FOLIOPATH_FACE_APP_IMAGE=foliopath-app:p23-final node spikes/int001-remote-ml/face-business-probe.mjs
FOLIOPATH_P3_APP_IMAGE=foliopath-app:p23-final node spikes/int001-remote-ml/p3-business-probe.mjs
FOLIOPATH_P3_APP_IMAGE=foliopath-app:p23-final node spikes/int001-remote-ml/p3-fault-probe.mjs
FOLIOPATH_P3_APP_IMAGE=foliopath-app:p23-final node spikes/int001-remote-ml/p3-recovery-probe.mjs
FOLIOPATH_P3_APP_IMAGE=foliopath-app:p23-final FOLIOPATH_P3_D04=1 FOLIOPATH_D04_VISUAL=1 node spikes/int001-remote-ml/p3-browser-probe.mjs
sh tests/release/recovery_smoke.sh foliopath-app:p23-final
sh tests/release/upgrade_rollback_smoke.sh foliopath-app:face-worker foliopath-app:p23-final
```

环境为本机 Docker 14 CPU / 7.748 GiB，不是 E01 的 4 CPU / 4 GiB 容量验收。媒体为仓库合成 fixture、临时目录和独立卷；模型缓存只读。AI 无公开端口，应用绑定回环地址；没有在 server222 测试或部署。
