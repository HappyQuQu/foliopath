# CR-2026-046 OCR 独立合同与传输适配

- 授权：用户在CR-2026-045验证过程中再次要求“继续下一步开发”。POST-MVP-5后续OCR扩展，当前只实现合同/Go传输子片；不将新OCR能力加入冻结revision2发行承诺。
- 需求：候选FR-OCR-001/002及NFR-INT-007/009/010；Owner internal/ocr（模型身份/结果边界），internal/inference/remoteml（既有HTTP/共享准入），未来app才可组合。
- 前置：CR-2026-045横排离线实验已收口，90°识别0/4仍保留，不能直接将实验PIL worker或visual输送kind转为生产OCR。
- 决策：ADR-0032仅接受独立内部OCR身份/请求/响应合同与传输隔离实现；没有新部署/存储/任务/公开API。具体合同先于实现，见architecture/ocr-transport-contract.md。
- 目标：预配独立身份、发送图片前校验；每响应绑定身份；严格UTF-8/字段/重复键/框/置信度/文本及大小预算；空白与文件失败分开、未知故障不标坏源；继承共享准入/截止/无环境代理/禁重定向，不增加自动重试。
- 不交付：生产AI OCR endpoint/调度注册、主应用启用配置、文件读取/预处理、源指纹/代次与任务事务、SQLite索引/清理、融合搜索/原型/UI、原6010部署。后续这些合同仍需独立接受和验收，90°及质量/容量/发行仍未完成。

状态：**合同/Go传输消费者及隔离本地Docker验证完成，尚未生产组合**。无OCR服务endpoint/模型启用、源代次/任务/索引或搜索/UI，原6010没有更新。

## 交付与边界

- [ADR-0032](../adr/0032-independent-ocr-transport.md)与[内部合同](../architecture/ocr-transport-contract.md)先定义身份/请求/输出/失败；不更改主应用公开OpenAPI，内部固定端点尚未由生产AI服务实现，旧服务缺少端点时安全失败。
- `internal/ocr`唯一拥有独立Identity/Fingerprint和Normalize，绑定两份权重/输入/输出/runtime/cache的六个hash；指纹使用键排序JSON，含独立Python计算的golden一致性测试。校验UTF-8、NFC前后长度、纯文本、凸框与分数；成功空数组保留，nil拒绝。
- `remoteml.OCRClient`只消费调用者编码字节，BindOCR不发请求/激活模型，GET完整身份正确后才发送固定image/input multipart。与原Client/标签/语义等共用2个HTTP slot，单次GET+POST+响应读取共用30秒总截止，父截止优先，满额Busy、取消释放，无新队列/重试/代理/重定向。OCR自己的runtime/cache pin不与其他能力强行相等，也不修改旧身份。
- 响应≤128 KiB、最多128行/单行512/总8192 Unicode码点；严格字段大小写、重复/转义重复/未知/null、尾随JSON、数字与框。特别拒绝Go JSON默认会替换为U+FFFD的不成对surrogate转义，保留合法emoji及纯文本HTML字样，不返回上游异常/路径。
- 固定HTTP+code区分input_limit、invalid_media、busy、model_unavailable和输出/服务错误；仅合法身份和明确源代码可归为OCR坏图，500/断连/超时/未知code不归因源文件。任何失败返回零Result，不保留前半段文字。
- 架构检查将ocr加入capability依赖边界，并在当前阶段只允许remoteml消费它；app/API/SQLite/其他capability不能提前import。没有原型/页面控件变更，OCR仍未可用。

## 实际验证

- 定向`go test -race -count=1 ./internal/ocr ./internal/inference/remoteml`通过。新OCR顶层测试15项，含身份变化/非TOFU、NFC扩张/总量/凸框、wire异常整包拒绝、真实multipart、空白、错误脱敏、共享slot/取消/恢复、同一deadline、截断响应、禁重定向/代理。
- 独立Linux/arm64 Docker测试镜像`sha256:16b2f5e433cbb93ee0083ac97c2b16da92236f86027cbc295253227f5a7718d6`（**不是应用发布镜像**），固定Go1.26.5编译race测试；当前两包40项顶层（含15项新OCR）和总138项含subtest通过。运行时network none、root只读、1 GiB/2CPU、/tmp tmpfs，无模型/媒体/数据库挂载/端口；exit0、OOMKilled=false，容器已移除。不把合成HTTP结果称为真实OCR服务联验或native child回收证明。
- 实际fmt-check、arch-check、generate-check、lint、Go全量、integration与应用runtime E2E通过；本轮无Web改动，UI/浏览器矩阵未重复执行。Go使用独立extended临时目录，保留S0默认盘insufficient_space记录。
- 首轮编译修正response limit的int64类型；NFC扩张wire测试首轮因Go字符串与raw JSON转义不同没有真正替换素材，修正为raw字面值后重新通过，未放松预期。runtime E2E第一次Docker Hub frontend metadata EOF，原命令重试通过；初次日志保留。
- [机器证据](../evidence/ocr/2026-10-25-ocr-transport-local.json)记录固定镜像、源码/日志hash、检查与未执行项。单机arm64和测试预算不能代替native双架构/联合容量/发行。

## 下一步仍需实施

1. 接受安全imagevips输入合同和独立服务producer/调度注册；沿用共享native准入及单child，不直接搬实验PIL脚本到产品，不将visual输送kind当OCR协议。明确90°/版面支持范围，S0的90°0/4仍是质量缺口。
2. 定义源指纹/独立generation、missing选择、失败归属、fenced提交/checkpoint、停机/取消、结果清理与库删除/权限，再实现存储和后台索引。**本片没有这些结果或任务，不能显示全库OCR进度。**
3. 后端索引与搜索合同/真实模型隔离Docker验收后，才接正式融合检索OCR来源与原型/UI。原6010备份/部署另验，人工标签/删除抑制/人物/向量和10源文件继续保留。没有提交推送。
