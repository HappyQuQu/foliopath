# ADR-0032：独立 OCR 身份与有界传输子片

- 日期：2026-10-25；状态：Accepted for transport-only implementation；生产组合/索引/任务未授权。
- 范围：CR-2026-046 / POST-MVP-5后续OCR，FR-OCR-001/002、NFR-INT-007/009/010。
- 决策角色：产品范围沿用用户继续AI开发的授权；代理负责技术子片设计与隔离证据。此记录不是用户签署最终OCR质量/发行。

## 背景与选择

CR-2026-045在既有AI运行时中验证了横排中英文真实OCR，并暴露90°漏识别。OCR输出是文本/框，不是CLIP向量或JoyTag标签；沿用visual实验kind会混淆身份和结果。旧服务也没有生产OCR接口，不能先做可用UI。

比较保持实验（不能建立可验证消费者）、复用CLIP身份（错误兼容）、新第三服务（违反部署边界）后，选择在现有两单元边界内定义独立内部OCR协议。此阶段仅实现Go消费者，不改变现有AI服务或生产组合。

## 决定

1. internal/ocr拥有身份字段/指纹及文本、框、分数和输出预算；remoteml只实现传输、严格wire解码与现有共享client准入，不能读文件/DB、下载或创建任务。
2. 独立`foliopath-ocr-v1`身份绑定PP-OCRv5_server两权重、输入合同、输出合同、OCR运行时与cache manifest的精确hash。预配期望值，不信任首次GET；任何OCR变更不得重标CLIP/face/tag身份或旧结果。
3. 请求先GET `/api/v1/ocr/identity`，匹配后才POST `/api/v1/ocr/predict`；内部端点，不加入主应用公开OpenAPI。原6010/旧服务无端点时不可用，不回退CLIP、标签或上游自由模型调用。
4. Client.BindOCR共用同一个HTTP client和2请求slot；验证与预测共用一个30秒截止，满额立即Busy；没有额外队列/自动重试。URL/代理/重定向继续Client唯一owner。独立OCR身份不要求与另一个能力的runtime/cache相等，分别校验部署pin；这不是远端认证。
5. 成功必须带匹配身份指纹和非null数组；严格UTF-8/字段名/重复键/规范化后的文字与有界凸四边形/分数。整响应失败关闭，0行成功允许，但绝不能据此证明图片无文字。
6. 错误只按固定HTTP+code合同脱敏映射；源失败需匹配响应身份与明确代码，未知故障不标坏源、不输出上游正文。取消优先；无tag式恢复窗口扩展。
7. 后续生产集成仍须定义internal/files/imagevips预处理、源变化/代次、租约/事务/checkpoint、清理/权限/搜索合同和质量；该决定不允许绕过这些门槛，也不添加第三部署单元。

## 后果与复审

可先验证消费者失败语义并保持现有功能不受影响；成本是生产服务、存储/任务和UI仍未实现。实验输入/identity不自动成为生产profile，90°/版面仍有限制。公开API、持久化或任务一致性变化必须另有接受记录。

fitness：独立identity/结果单测、httptest错身份不发图片/大小与畸形输出/无重定向代理/并发取消截止、定向race和隔离Linux Docker；全量回归见CR-2026-046。没有新模型或原库测试。
