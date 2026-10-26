# OCR 内部传输合同 v1

CR-2026-046 / ADR-0032 / FR-OCR-001/002、NFR-INT-007/009/010；2026-10-25。
更新2026-10-26：CR-2026-046接受Go消费者，CR-2026-048/ADR-0034交付候选producer；CR-2026-049/050与ADR-0035/0036已交付源绑定存储和显式opt-in后台/管理。此内部传输v1未变化；主应用新增管理OpenAPI，全文索引/融合搜索和UI仍未接入。Owner：ocr合同/生成、remoteml传输、inference producer与冻结调度核心，app独占组合。

## 身份与请求

部署预先固定JSON字段，大小写精确、不缺失、不null/未知/重复：
`version=1`、`protocol=foliopath-ocr-v1`、`model=PP-OCRv5_server`、
`detector_sha256`、`recognizer_sha256`、`input_contract_sha256`、`output_contract_sha256`、`runtime_contract_sha256`、`cache_manifest_sha256`。
六个SHA256必须是64位小写hex；指纹为包含以上全部字段的UTF-8 JSON（键字典序、无多余空白）SHA256。
不从模型名称推断权重，不将实验身份/现有语义face/tag身份用作OCR身份。预配置验证不等于健康或模型激活。

通过原部署origin的`GET /api/v1/ocr/identity`（JSON ≤4096 B）核验完整身份后，才发送
`POST /api/v1/ocr/predict` multipart，只有固定字段`image`/文件名`input`、1 byte–8 MiB编码图片。
只接受ADR-0033准备的RGB8非交错PNG（32～2048/≤4Mi像素），拒绝EXIF/文本/ICC/动画等元数据或未知chunk；允许pHYs，验证CRC与完整帧。没有用户URL/文件路径/词表/语言切换/任意模型。Go调用者负责安全打开/预处理，HTTP只验有界编码结构，不在主进程解码native像素。
两个HTTP步骤共用一个现有slot和30秒deadline，调用者更短deadline优先；满2slot立即Busy，无新排队/重试。
无环境代理、无重定向；失败时不发图，不把身份GET成功缓存成永久授权。

## 成功响应

HTTP200，JSON Content-Type，最多128 KiB：
`{"identity":"<expected fingerprint>","lines":[{"text":"...","box":[8个数],"detection_score":0.9,"recognition_score":0.95}]}`。
严格UTF-8、字段名、必填/null/重复字段/尾随JSON全部拒绝。最多128行，单行512 Unicode码点，NFC后再次检查，合计8192码点。
拒绝空白文本、控制/代理码点；保留原文作为纯文本，不执行HTML/推断tag/拼接文件名。box为读入图片完成EXIF/profile后的归一化凸四边形，4点顺序连续，非零面积，有限值[0,1]。
检测分数≥.5、识别分数≥.8且≤1，不表示概率或准确率。数字null/缺失/NaN等拒绝。任一行不合规则整响应失败，不截取前N项。
空`lines:[]`成功，不等于输入不存在/损坏，也不证明图片无字；方向/复杂版面质量另验收。阅读顺序由未来output_contract固定，不能推断通用版面。

## 错误响应

JSON最多1024 B，只有`identity`与`error`，身份仍须匹配；不回传正文/URL/模型异常：

| HTTP/code | 消费者结果 |
| --- | --- |
| 400/invalid_input | 传输invalid input |
| 413/input_limit | OCR输入超限，不是损坏 |
| 422/invalid_media | OCR明确输入不可解码，不能泛化为源文件不存在 |
| 422/output_invalid | invalid response，不能标坏源 |
| 503/busy | Busy |
| 503/unavailable或model_unavailable | Unavailable，无隐式重试 |
| 500/inference_failed | Unavailable，不返回异常正文 |
| 合法响应中的身份指纹不符 | identity mismatch，不持久化部分结果 |
| 缺字段/null/其余组合或畸形响应 | invalid response，不降级为空或源错误 |

未就绪时identity GET返回503；错误体可带静态兼容指纹，但它不表示模型可用，只有200完整identity匹配才允许发送图片。身份GET非200视为Unavailable；POST因断连或未知服务失败亦不猜测输入错误。context取消/截止优先且释放slot，后续请求可恢复；HTTP取消本身不证明服务native child已回收，生产服务须继承ADR-0021。

## 持久运行与后续边界

CR-2026-049/050已覆盖源指纹/代次/结果表、missing/显式retry_failed、lease/checkpoint、原源变化/离线、计划停机、清理/库删除及元数据管理。见[持久合同](ocr-persistence-contract.md)与权威OpenAPI。默认不配置/不启用，不扩展原6010。传输层错误类别不变；调用者须区分阶段：preprocessor明确input_limit/invalid_media可记源失败，producer拒绝已经准备的PNG或区域预算属于任务/程序错误，不能倒推原件损坏。

尚未交付OCR全文索引/搜索来源、公开正文或产品UI；90°/复杂版面/真实质量、历史保留、联合容量/双架构/发行继续独立。原6010、旧向量/标签/人脸/人工记录未操作。
