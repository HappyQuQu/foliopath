# P4 冻结候选小样本收口证据

范围：CR-2026-030 / FR-INT-002、005、009 / NFR-INT-002、003；Owner app/semantic/face/测试。依据用户约 100 样本变更，停止新增生产优化，本轮仅补测试记录。

## 固定身份

- arm64 app：`sha256:6325266035ffb7329d8e11ada7d84e1497dd8bc4ae5d407a6caa5affa52d09d3`。
- amd64 app：`sha256:b754adb456758137f334b26a0974bede13a16b63301908eec0f851a70ba16640`，同一取消修复源码原生编译。
- arm64 AI：`sha256:28f8fb6351234b5e94d09477f9d112ef36f2b41bc8537268d50e6bd1459f0d85`；amd64 AI：`sha256:1647c1214bd014f7c36b8577656cfbd2e15f625a59f0ab4e6f1d1058797b3422`。
- p4-large：app 1 CPU / 1024 MiB，AI 3 CPU / 2304 MiB，无 swap；私有 AI 网络、只读样本/模型缓存。测试控制器另计，不将控制器内存冒称应用自身内存。
- 匿名样本 manifest SHA256：`99fb58c553cf7a018a4de77d2c9b8796791951634a9fbdaf3e4317d7703715ce`，90 图/10 视频；原图不变。

## 恢复与升级

`p3-recovery-probe.mjs` 在冻结本地镜像退出 0；日志 `test-results/p4-final/recovery.log` 从 11:29:06 到 11:36:56，约 7 分 50 秒，6 类实际检查：

1. 正常停机后的实际任务租约恢复，不标为用户取消。
2. SIGKILL 后实际租约恢复，人工状态保留。
3. 推理连接中断任务失败、重启服务后可重试，不误报成功。
4. 实际 HTTP 超时及不可用状态、恢复服务后重试。
5. 错身份失败关闭，不跨代次写入；正确身份恢复。
6. 停写备份恢复人工人物/标签、清除派生脸后重建保留关系；原图 hash 不变。

这是有限故障循环，不是数小时/数天浸泡、内存泄漏排除或 100k 联合负载证明。

`p4-semantic-upgrade-probe.mjs` 在旧 app `sha256:0421fb761cbf2d00c847038f566dfdebb031f8b59ca4cf02630995ac86503b64` 创建独立合成旧库，升级到上述冻结 app：新代次完成数不继承旧向量、两库重建、人工状态及人脸代次保留、重启持久化、旧镜像＋升级前备份配对回退全部通过。共享短视频固定为 2 eligible / 1 completed / 1 degraded / 0 failed，未冒充全部视频 complete。报告 `test-results/p4-final/upgrade.json`，pass=true。

## 原生 amd64

9 项语义预处理 race 专项在 1 CPU/1 GiB/无 swap/无网络容器通过，包含取消、大图、渐进 JPEG 和边界测试；日志 `native-unit.log`。

维护 `/tmp/foliopath-p4-native.cTAmnT`：图片语义 90/90（37415 ms）、人脸 90/90（86646 ms）、视频 10/10（16136 ms）、标签 90/90、中英文搜索 8 次完整通过，functionalPass=true，hash 不变。结束内存快照 app 203.2 MiB / AI 1.895 GiB（非峰值）。报告 `test-results/p4-final/native-sample.json`。

最终 `test_exit=0`、`services_restored=1`，原四服务 healthy，前后镜像 ID 相同；不是线上更新。

## 跨架构与剩余边界

配对本地最终样本报告 `test-results/p4-final/local-sample.json`：语义 90/90（16108 ms）、人脸 90/90（44311 ms）、视频 10/10（10091 ms）、标签 90/90、搜索 8 次通过，functionalPass=true、hash 不变。两边 manifest digest 完全一致。

`p4-rank-compare.mjs` 以匿名样本名对齐，而不是不可跨库复用的数据库 ID；输出 `test-results/p4-final/rank-comparison.json`：8 组查询全部 exactRankMatch=true，结果集合相同；4 组视频代表帧相同，最大绝对分数差分别 0.00000345、0.00000129、0.00000215、0.00000068。这里只报告测量值，没有临时制造通过阈值。图片 API 不提供相似度分数，不能将图片排序观察写成全向量数值证明，更不能将两侧一致写成准确率通过。

独立质量标注未取得。已生成 100 项匿名标注草稿 `test-results/p4-final/annotation-draft.json`，所有答案为 null、标记 UNREVIEWED；既不是模型预测真值，也不是正式质量验证器输入或批准材料。

本轮执行架构检查、Node 结果门禁 3 测试、排名比较 3 测试、脚本语法和 diff 检查；未重跑全部七项仓库发行检查。原大规模门禁延期，质量/长期/发行未验证项不标为通过。

## 用户验收决定（2026-09-17）

用户明确确认“算完成”“标记通过”，本轮 P4 按约 100 样本工程范围标记**验收通过、已完成**。人工质量校准确认延期，不是准确率测试通过；本决定不改变上述实测数据、证据限制、生产质量门禁或 P5/P6 状态。见 [范围变更](../../changes/CR-2026-09-16-p4-small-sample-validation.md)及 [收口台账](../../features/intelligent-media-discovery-p4-closeout.md)。
