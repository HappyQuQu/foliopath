# 独立 ML HTTP 传输合同 v1

2026-09-14；CR-2026-030 / ADR-0019；POST-MVP-5；FR-INT-017/019/020、
NFR-INT-007/009/010。Owner：inference 传输适配；semantic 保留向量规范化与代次所有权。
本合同接受 **Go 传输层及其隔离测试**，不接受模型激活、生产组合或新存储代次。

## 边界

`internal/inference/remoteml` 只消费调用方提供的编码图片/文本，不访问文件、不读 SQLite、
不启动任务、不下载模型。app 尚不得将它接入现有 SigLIP ImageEncoder 张量接口。
部署提供 HTTP/HTTPS origin；无用户信息、路径（除 `/`）、query、fragment。
默认目标必须是同机 Compose 私有网络服务，不开放端口；构造器的 URL 语法验证不是
网络隔离或远端身份认证。不得把浏览器输入传给构造器。忽略所有环境代理，禁止重定向。

## 请求与资源上限

- 当前固定协议：Immich ML v3.1.0；固定候选模型 `ViT-B-32__openai`、512 维。
- POST `/predict` multipart：`entries` 为 clip/visual 或 clip/textual 的 modelName；
  图片字段 `image` 使用固定文件名，文本字段 `text` 保留原文本，由服务负责 tokenizer。
- 单张编码图片 1 byte–8 MiB，文本 1–512 Unicode 码点且 UTF-8 合法、非纯空白。
  图片解码/像素预算必须由后续媒体适配层负责，本层仅限制编码载荷。
- 单客户端最多 2 个在途请求，满额立即 busy，不在传输层排队；app 未来只组合一个共享实例。
- 每请求最长 30 秒，调用方更短截止时间优先；无自动重试。响应最多 64 KiB。
  取消 HTTP 不保证上游 native 推理立即停止，服务端并发/容器资源限制仍必需。

## 响应与故障

只接受 HTTP 200、合法 JSON 对象中的 `clip` 字符串，该字符串必须编码一个恰好 512 项的
有限、非零 float32 向量（null 项也拒绝）。允许 imageWidth/imageHeight 等上游附加字段。
复用 semantic.NormalizeEmbedding，不创建第二份向量规范化策略。

错误分为 invalid input、busy、unavailable、invalid response、调用方取消/截止。
不返回服务端响应正文、URL 或底层网络错误。请求超时映射 context.DeadlineExceeded；
连接失败、非 200（包括重定向）映射 unavailable。错误均不产生部分向量、索引写入或回退。

**成功仅证明 HTTP 向量有效，不证明模型身份。** 上游响应不带权重/预处理 hash；不能把
请求的 modelName 当作服务端身份声明。生产准入仍需受信镜像和完整只读模型缓存清单、
部署指纹以及新 generation 合同。512 维结果禁止写入现有 768 维代次。
旧 API、旧迁移、UI 和正式 Compose 在本片不变。

## 验证

Go httptest 合成合同覆盖 multipart、向量错误、响应超限、失败脱敏、取消/超时、
并发拒绝与释放、禁止重定向及显式无代理。真实服务测试只用生成图片，不读 coser。
主应用搜索/回填/重启集成、代次切换、模型身份准入、人脸适配仍是后续 Gate。

上游合同来源：[固定版本 main.py](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/main.py)、
[schemas.py](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/schemas.py)。
