# CR-2026-048 OCR producer 与共享调度接入

- 授权：CR-2026-047后用户要求继续下一步开发。POST-MVP-5后续OCR，FR-OCR-001～003、NFR-INT-007/009/010。
- Owner：ocr合同/生成，inference OCR producer/固定模型，原InferenceScheduler唯一准入和回收；app/存储/UI不组合。
- 决策：ADR-0034，显式OCR HTTP/worker kind，兼容路由委托冻结核心，不修改旧模型空间或源文件；独立缓存/身份/错误。新镜像隔离测试，不部署原6010，不连server222。
- 目标：Go预处理→Go客户端→真实OCR HTTP/child、模型及输出边界、跨族共享/取消/故障、冻结identity和旧功能输出回归；缺OCR缓存不关闭旧能力。
- 不交付：源代次、持久任务/索引/清理、公开搜索/原型/UI或实际全库OCR；质量/联合容量/双架构/发行仍独立。

状态：**producer与共享调度候选开发、本地真实模型Docker验收完成（2026-10-26）**。未组合主应用OCR配置/源代次/任务/索引/搜索/UI，未更新6010或连接server222。

## 实现与身份

- [ADR-0034](../adr/0034-ocr-producer-shared-admission.md)与[内部合同](../architecture/ocr-transport-contract.md)明确边界。`Dockerfile.ocr`固定base52f4a40c，只COPY OCR新文件；未修改旧scheduler/server/worker或manifest。
- `ocr_scheduler.OCRScheduler`仅继承/委托v1核心，ContextVar保存真实kind；OCR在冻结白名单中映射private准入类别，持原lock进入_run后传实际ocr kind与固定worker/cache。原infer仍拥有pending2/单child/26秒总截止，原_run/_reap/close负责清理；恢复route在child回收后执行。没有复制admission/timeout/retry或增加队列，ownership AST fitness进入arch-check。
- 基础lifespan完成且旧owner为空/未服务HTTP时关闭它，再建立唯一共享路由owner；原lifespan负责最终关闭。OCR cache只读挂载/独立校验，两权重固定hash且私有snapshot每child再验。缺OCR cache只关闭OCR readiness，不让旧能力失效；不下载或回退其他模型。
- Go `internal/ocr`是input/output策略唯一owner，contractgen确定性生成producer JSON，make generate-check校验；共享12组conformance覆盖中英文/NFC/纯文本/空结果/框/分数/总量等。输入hash保持753de438…，输出hash93afd78d…；新runtime/cache身份独立，不重标旧结果。
- `/api/v1/ocr/identity`与`/api/v1/ocr/predict`仅AI内网：先验部署pin，PNG RGB8/CRC/尺寸/元数据预算，HTTP不做native像素解码。child只加载已准备PNG，不再裁剪/EXIF/缩放或颜色变换。检测/框/crop/文本有界，阅读顺序与分数同步；0行是完整成功，90°方向仍不推断。
- 错误只含固定code与OCR静态指纹（兼容不是就绪/认证）；identity GET只有200完整匹配才可发送图片。资源/坏源/模型/输出/暂时故障分开，未知model异常不冒称invalid_media。复用原Body限额，OCR边界只规范化自己的限额/timeout错误，旧路径直接委托原application。

## 最终固定Docker验收

- AI候选 `sha256:e995f5748c36ae8de9f3dd86633e4a2f60f3f6d6b7eaaba6616bbdf6ee73da21`；Go native/race probe `sha256:e612537c587d82c8a825b12d28fdfd5718fea0e6097481b8c9d475f1e229e5f3`。本地linux/arm64，原生Go1.26.5/libvips固定依赖。
- `ocr_tests + test_scheduler + test_runtime_manifest`共25项通过（含12组跨语言夹具的subtest），额外2项真实权重native测试通过。缺失/同尺寸篡改/符号链接模型均model_unavailable，非源错误；真实child RSS915,388 KiB时取消后PID消失、scratch为空，anon由26,976,256回26,988,544 B（+12KiB），后续legacy与真实OCR恢复。
- 冻结base→OCR→无OCRcache三阶段实际Go联验通过：28份旧.py/.json字节hash及旧semantic/face/tag身份保持；完整文本/图像向量、JoyTag与face结果序列化逐字节相等，摘要 `8d782cbc3df4b189082f3fa25516e2a1c34597f016accf893a06e2ab4e3f94e7`。只对本轮固定合成输入作兼容结论，不当作普通照片质量。
- Go准备PNG/EXIF JPEG实际OCR均四行中英文/数字命中，宽图命中英文、空白0行，坏PNG明确invalid_media；没有伪造返回或使用S0视觉transport worker。缺OCRcache时旧输出继续相等，OCR校验与调用Unavailable，不自动重试或发送图到其他能力。
- 独立internal网络/no ports/root只读，模型源只读+专用磁盘scratch；无媒体/DB/应用数据。服务4GiB RAM且无额外swap/2CPU，客户端1GiB/2CPU；三阶段服务peak约3.613/3.656/3.685GiB，memory.events max/oom/oom_kill均0。不是主应用+AI联合4GiB预算或长期容量通过。
- fmt-check/arch-check/generate-check/lint、Go全量/integration/定向race与应用runtime E2E通过。无Web变更，UI/三浏览器矩阵本轮未运行。没有native amd64、真实OCR质量/版面、联合压力/发行验收。源码与日志hash见[机器证据](../evidence/ocr/2026-10-26-ocr-producer-local.json)。

## 初始失败与更正

1. 首轮基线测试误选不含JoyTag的缓存，启动因hash/缺失失败关闭；改为现有完整预备缓存并在启动前逐文件hash核对，没有下载/改原模型。
2. 首个512x512 native取消样本在结束前未观测到768MiB阈值，故测试失败；独立复核同图可返回正常空结果，换为1000x700合成白图后实际观测约894MiB RSS并取消。未降低阈值或将未观测宣称通过。
3. 实现审查将模型ValueError与PIL decode异常分开，补回归验证未知inference错误不归因源；未就绪错误也绑定静态指纹，GET仍503，避免就绪变化的错误分类歧义。

失败日志保留；测试容器/network/快照已清理，原件/人工标签/抑制/人物与旧任务未改，没有提交推送。

## 下一步

先接受并实现源指纹/独立generation、missing/失败归属、租约/fenced提交/checkpoint、计划停机/取消、清理与权限，再接后台OCR索引和正式搜索来源。当前没有可显示的全库OCR进度，也没有用户OCR启用按钮。无EXIF 90°、多栏/表格/普通照片准确率、模型再分发/供应链、双架构/联合容量/发行及原6010部署继续独立保留。

复现：[README](../../spikes/ocr-service/README.md)。
