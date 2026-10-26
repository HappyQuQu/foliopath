# CR-2026-045 OCR 离线推理可行性

- 授权：CR-2026-044后用户要求继续开发。POST-MVP-5后续OCR扩展，当前仅S0可行性验证；FR-OCR-001～003候选，不变更已冻结revision2的生产范围，不把OCR搜索宣称完成。
- Owner：spikes/ocr本地验证；复用既有固定Immich3.1.0运行时、RapidOCR3.8.1和ADR-0021调度器的测试命令注入点，生产服务/模型身份/API/数据表均不修改。
- 目标：固定PP-OCRv5_server检测/识别权重，独立缓存、哈希校验和私有快照；中英文合成图离线实际识别、空白/损坏/超限/模型失配分类、输出有界验证；单请求进程结束释放后才接下一请求，测峰值与取消/超时回收。
- 边界：离线测试无网络、无媒体库/数据库挂载，仅合成图片；模型准备可按用户已授权代理从固定上游URL下载，预先固定SHA256，校验后原子发布到独立实验缓存。不可自动回退直连，不接任意模型URL，不碰原模型缓存或6010/server222。
- 不交付：此阶段没有生产OCR接口、后台索引、持久化或融合搜索；OCR筛选保持未接入。S0通过后才可单独确定ADR/版本化内部合同、源指纹/存储/任务一致性与安全清理，再推进后端和UI。不得将调度器实验输送kind标识视为语义模型身份或生产协议。
- 模型来源：固定运行时内RapidOCR的get_model_url返回ModelScope v3.8.0两个ONNX与预置SHA256；代码参考固定Immich v3.1.0 OCR检测/识别实现。上游许可证参考不是发行批准，模型再分发/完整SBOM/native双架构与质量评估仍独立。

状态：**S0隔离实验收口：横排中英文推理可行；非生产Backend Ready**。OCR索引/搜索/API/UI没有实现；90°方向/版面与质量问题保留，未部署原6010。

## 实现与复现

- 实验源码：[spikes/ocr](../../spikes/ocr/README.md)，合成素材：[tests/fixtures/ocr](../../tests/fixtures/ocr/README.md)，机器结果：[JSON](../evidence/ocr/2026-10-25-ocr-feasibility-local.json)。范围/生成文字及失败素材明确标注；无真人/真实库图片、数据库或字体文件进入测试素材。
- 基础运行时固定 `sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626`；最终实验镜像 `sha256:a8ac357f181861c713a08fa4fa4acad175e689b84973b9979e3c4f53c756a1a7`，本地Linux/arm64。它只COPY实验文件，未改生产AI镜像或原6010容器。
- 两权重总计172,695,790 B（约164.7 MiB），准备至独立实验缓存 `/Volumes/extended/foliopath/ocr-feasibility-cache`。预先已知SHA256/尺寸，固定ModelScope HTTPS及精确CDN重定向白名单，显式HTTP代理，不继承环境代理/不自动直连。运行中禁下载、只读源模型，验证私有快照在独立临时scratch，结束删除。
- 输入限制8 MiB / 4 Mi像素，最小边32，EXIF/RGB/长边2048；检测分辨率参数736遵循上游短边行为，并非长边上限。128原始轮廓/文本框、16 Mi总裁剪像素、单行512/总8192字符，检测.5/识别.8，单crop批处理。原始轮廓超限在上游截断前拒绝；四边形/数值/置信度/文本全量验证，非空异常不伪装成空结果；NFC规范化后再次核对单行字符上限，避免组合字符扩张越界。
- 横排阅读顺序由实验合同明确拥有；不依赖上游数组天然顺序，排序同时重排对应分数。不是竖排/多栏/表格解析。实验身份 `e343e2b7a27eaa03a7967310b02ec31f0ee118fbc3df2d996edb5c50b9738681` 绑定固定运行时、权重、输入/输出策略和worker/contract源码；不复用CLIP/face/tag身份。

## 实际模型结果

| 场景 | 真实结果 |
| --- | --- |
| 横排中英文 | 四行依次识别FolioPath OCR 2026、中文文字识别测试、订单编号123456、LOCAL TEST ONLY；断言使用大小写/空格规范化，不称逐字排版还原 |
| EXIF与宽图 | EXIF方向修正后四行匹配；3000宽图缩至2048后识别WIDE IMAGE OCR 2048；归一化坐标/分数有效 |
| 空白 | ready且0行，区别于读取失败 |
| 额外90°图 | ready但0/4行匹配，是**已知漏识别**；此场景仅观察，不算准确率通过或宣称图片无文字 |
| 坏源与输入预算 | 损坏/空输入invalid_media；8 MiB以上及6MP图input_limit |
| 模型失配 | 缺失、同尺寸错误字节、符号链接均model_unavailable，不下载补齐或放行 |
| 准入与取消 | 共用既有单child调度器，pending最多2，第三请求Busy；观察child RSS845,028 KiB后取消，reap/删快照后anon由31,625,216回31,621,120 B，后续真实识别恢复 |
| 超时 | .25秒启动/cache-preprocessing deadline正确终止并reap；不是完整原生推理超时性能基准，调度器超时分类另由继承单测覆盖 |

最终固定镜像15个场景断言通过（包含分类/生命周期与旋转负面观察），容器2CPU/4 GiB实验预算、根只读、network none/无端口、仅OCR缓存和合成素材只读挂载。模型/素材hash前后一致，scratch为空，child均已reap，测试容器已删除。峰值1,925,079,040 B（约1.79 GiB），单child最高RSS1,573,688 KiB（约1.50 GiB），memory.events max/oom/oom_kill均0。不能横比为小模型预算、与其他模型联合容量或长期稳定性证明。

## 验证与首轮问题

- 最终镜像contract/download/worker共15项单测通过，继承scheduler/runtime-manifest共11项通过；宿主纯合同7项/下载4项通过。下载单测只用tiny合成字节，覆盖哈希/尺寸/上限/已有篡改不覆盖、无环境代理、重定向/代理凭据拒绝。
- 实际执行fmt-check、arch-check、generate-check、lint通过。Go全量与integration使用独立extended临时目录通过；首轮默认临时盘触发insufficient_space，保留失败，不放宽测试或空间保护。
- 原生Docker应用运行时E2E重试通过；首轮Docker Hub获取dockerfile:1.7元数据EOF，保留日志后原命令重跑，无镜像源/测试代码变更。UI/浏览器矩阵本轮未运行，不能沿用上一切片通过数。
- 首轮旧face-candidate没有scheduler测试模块，改用已核验固定photo-tags-v2基础；上游OCRcache_dir会自行追加子类型，修正双重路径；脚本添加固定上游模块路径；排序错位/阅读顺序修正后重跑实际模型。Docker Desktop的未配置隧道模板不能当作已联网接口，改为核验真实NetworkMode/端口/挂载、无路由和egress socket失败。初次准备的未列入CDN重定向失败关闭后精确增加白名单，最终权重hash仍不变。上述均为实验接线问题，不冒称修复了原库故障。

## 后续门槛

1. 先明确旋转/版面支持范围与真实中英文质量门槛；当前90°漏识别不能隐藏。小规模合成文本不是普通照片、复杂文档、低清/恶意样本质量证据。
2. 生产集成前接受ADR及独立OCR协议/身份和预处理合同；主应用仍由internal/files/imagevips安全读取/变换，不能将本实验PIL代码移入HTTP处理器。源指纹/代次、空结果与失败、bounded队列/恢复、事务推进、库删除与敏感OCR文本清理须唯一owner。
3. 后端索引/搜索合同与持久化回归通过后，才将OCR来源接入融合检索和正式原型/UI；现有OCR筛选仍未可用。本轮没有OpenAPI/数据迁移/Story/生产组件变更，未重复跑浏览器矩阵。
4. 原6010部署/真实库OCR、10源文件恢复、模型再分发/许可审查/SBOM、native amd64、联合容量/长期/发行仍独立保留。没有提交推送。

来源：[固定Immich检测实现](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/models/ocr/detection.py)、[识别实现](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/models/ocr/recognition.py)、[RapidOCR模型元数据](https://github.com/RapidAI/RapidOCR/blob/v3.8.1/python/rapidocr/default_models.yaml)。
