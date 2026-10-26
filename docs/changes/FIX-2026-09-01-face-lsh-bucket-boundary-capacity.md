# FIX-2026-09-01：人脸 LSH 桶边界与 512 维容量收敛

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-243`、`INT-250`
- 影响不变量：大集合候选生成必须有界；合成容量不得冒充真实身份质量或偏差证据

## 问题

原 100k 容量只使用 32 维合成向量。升级到 AuraFace 候选的 512 维后，运行在 10 分钟超时：LSH
有序桶的窗口从固定起点向前扫描；当该起点仍属于前一个 signature 桶时，循环会立即停止，跳过当前桶
开头本应比较的相邻成员。被漏掉的成员随后进入逐成员 × 逐 core cluster 的 edge 扫描，使容量路径退化。
独立审计还发现，即使没有桶边界缺陷，大集合 edge 仍会遍历所有 core cluster，并可能附着到先前创建的
edge-only 单例；前者在真实单例占比较高时仍是平方级，后者违反“edge 只附着 core”的合同。

## 修复

- LSH 窗口先在最多八个前驱内越过不同 signature，再按原有顺序扫描当前桶成员；桶边界后的第二个成员
  会正确与第一个成员比较，仍保持每表最多八个候选的上界。
- 大集合 edge 只评估四张 LSH 表中已经登记的有界邻居；cannot-link 先映射到 core cluster 后优先拒绝。
  小集合 exact 路径也冻结 core cluster 数，不再让后续 edge 挂到先前创建的 edge-only 单例。
- 新增超过 exact 路径阈值的双桶回归，要求 4,098 个成员全部保持 core，禁止桶首成员静默降为 edge。
- 新增大集合 edge 回归，证明候选只附着 core，且对任一 core 成员的 cannot-link 会拒绝整个 cluster；
  小集合另固定没有 core 时相似 edge 仍保持独立单例。
- 100k 容量 fixture 升级为 512 维，并连续运行两个极端场景：50,000 个确定性双成员 core，以及
  100,000 个无 core 单例；输出两侧成员/cluster 数、总耗时、Go memory sys 与不含向量的结果指纹。
- 原生候选镜像同时编译该容量测试；harness 在断网、只读、4 CPU、4 GiB、256 PID 容器中执行并生成
  `face-capacity.json`。严格配对 verifier 绑定 native identity、同镜像、100k × 512、资源档、非质量 flags
  与跨架构结果指纹。

## 回归证据

- 修复前同一 Darwin/arm64 运行在 10 分钟超时，栈停在 edge 的逐簇扫描。
- 修复后 Darwin/arm64 的 paired-core + all-singleton 两个 100k × 512 场景合计 7.194 秒完成，Go
  `memory.Sys=421,134,696` bytes。
- 同一双场景测试在 Linux/arm64 Docker、4 CPU/4 GiB 限额下于 7.157 秒完成，
  `memory.Sys=425,300,328` bytes；结果指纹为
  `ed978ca7f471ba742a38f680cebb5f83481b8f622a70b005b906e654f2b706d4`。
- 本地 Linux/arm64 结果不是远端 native amd64/arm64 配对结果，也没有身份 ground truth、最终模型联合负载
  或 owner 签署，因此 `INT-250` 与 S2C Gate 继续未完成。
