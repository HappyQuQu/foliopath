# INT-S2C：人脸隐私与模型准入

- 日期：2026-08-29
- 目标版本：[POST-MVP-5 revision 2](../../releases/POST-MVP-5-scope-r2.md)
- 范围：E 匿名人脸聚类与管理员人物库
- 当前判断：**No-Go / External Admission Pending**
- Contract input：[INT-S1R2 Contract Ready](int-s1r2-contract-ready.md)（Go）

2026-08-31 产品用户决定当前执行顺序先跳过人脸测试。该决定把 S2C 的实现与测试暂缓，不删除 E、
不降低本 Gate，也不允许用“未执行测试”宣称完成；工程继续推进非人脸 S2A/S2B 和发布证据，直到
下列外部准入输入到位后再恢复 S2C。

## 开始 production 实现前必须同时具备

1. 隐私 owner 接受默认关闭、告知、数据分类、访问、删除、备份/恢复与 incident fallback；
2. 合法真实 face ground truth 的来源、授权、purpose、访问、保留、删除 owner 与删除证明；
3. 可商业分发的 detector/embedding/runtime 候选及精确 hash/license/native 双架构来源；
4. 冻结评测协议能证明 anonymous core precision ≥99.5%，否则合同明确降为逐脸/小组；
5. 数据不进入 git、普通 CI artifact、日志、诊断或公共对象存储的可执行控制。

在这些输入签署前，不创建 production `internal/face`、migration、handler、worker、route 或 UI。允许继续
非真人、无权重的隔离 contract/transaction test 设计，但不得解释为隐私、质量或模型准入证据。

失败回退：保持 E 全局不可用并不创建/保留新的 biometric 派生数据；核心浏览、人工标签、故事板以及
已独立通过 Gate 的其他切片不受影响。
