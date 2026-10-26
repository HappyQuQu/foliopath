# CR-2026-047 OCR 安全图片输入

- 授权：CR-2026-046后用户要求继续下一步开发。POST-MVP-5后续OCR / FR-OCR-001～003、NFR-INT-007/009/010。当前只接受安全预处理子片，非冻结revision2新增发行承诺。
- Owner：ocr输入合同/错误与结果类型，imagevips唯一native变换/共享大图准入；未来files安全打开，remoteml既有传输不读文件。
- 决策：ADR-0033。当前1024语义预处理不能代替OCR；独立输出长边2048、RGB8 PNG、白底alpha、EXIF归正和去元数据，固定合同hash。source 64MiB/40MP，encoded output 8MiB/4Mi像素；native前检查格式/尺寸，取消与读取失败脱敏。
- 边界：只处理调用者已经打开的seekable stream，不接受路径或自行打开原件；复用imagevips全局native admission/清理。语义/tag/face预处理与身份不变，不注册OCR服务/任务/模型或写SQLite，不改6010/server222。
- 验收：原生libvips合成格式/EXIF/alpha/大图/错格式/超限/读取/取消/共享准入回归，原有native处理回归，固定中英文合成图经本适配后接现有OCR实验模型实际识别；不是生产服务联验、泛化准确率或联合容量。

状态：**安全图片输入实现与本地Docker验证完成；生产OCR服务/索引/搜索仍未接入**。原6010未更新，模型/原件/人工整理/原任务未改动。

## 实现

- `internal/ocr/image.go`拥有canonical `ocr-rgb-png-v1`、输入hash、PreparedImage及ImagePreprocessor接口；hash `753de4384b85c73f55020829b8dc96d41700881f01e9064afbbb8f963df46a4e`，规范化JSON/独立golden验证。返回PNG/归正后宽高/合同hash，不能据此推断源指纹或索引代次。
- `imagevips.Processor.PrepareOCRImage`只消费已打开stream，JPEG/PNG/WebP/GIF第一帧，64MiB/40,000,000源像素/最小边32，header最多1MiB；native前检查声明格式/源几何，native后复核实际格式/几何。固定本次encoded buffer供header/native使用，不重新按路径读取。
- EXIF归正、sRGB/uchar RGB、alpha白底、Lanczos3长边2048且不放大、PNG去元数据；最终最小边32、≤4Mi像素/8MiB编码。过窄或高熵输出超限返回input_limit，不偷偷改变缩放/有损编码。FailOnError=true，无旧JPEG修复回退。
- 复用`largeImageSourceNeedsReclaim`/`beginLargeImageDecode`及`readRemoteSemanticSource`，没有第二native门或重复的无进展读取策略；未知caller stream不额外预读，ImageRef先关闭再执行共享线程/native回收和准入释放。取消在合作式边界检查，不宣称立即打断任意C/IO调用。
- 参数、确定头部无法解析、输入/输出预算、IO/native处理和native缺失分别映射固定错误；短读/源长度变化/IO不自动标坏源，失败不返回半张PNG或合同hash。非libvips stub失败关闭。语义1024/tag/face处理与身份无修改，API/SQLite/app不组合。
- arch-check只为imagevips增加受审查消费者；仍阻止app/API/存储提前直接接OCR。新增Docker测试targets在正式runtime之外，不是新部署单元或发行镜像。

## 实际检查

最终本地Linux/arm64测试镜像 `sha256:632bccdf846784da17ec6130b967e46408a36d17e4f4c214d2cc0a8529db4a26`，使用正式固定libvips8.16.1/GLib/Expat与Go1.26.5，race编译；network none/root只读、1GiB/2CPU、临时/tmp及只读合成progressive样本，exit0/OOMKilled=false。

- 最终native包45项顶层（总74项含subtest）全部通过，0 skip；7项新OCR覆盖四格式/首GIF帧、旧语义1024保持、白底alpha/EXIF/元数据、40MP精确边界与超限、header/字节/窄图、约50MiB合法PNG文件、8MiB高熵输出拒绝、共享准入/读取取消/100次无进展、16bit与progressive。
- 首轮native全包有两项既有progressive测试缺fixture而skip；随后独立生成合成9000x9000 progressive JPEG，并用最终固定镜像全包重跑，两项实际通过。2GiB仅用于一次性fixture生成，不冒称资源测试；最终测试仍1GiB，不使用用户原图。
- [真实模型桥接](../../spikes/ocr-image-input/README.md)：三个固定合成输入先经正式Go适配，PNG和EXIF JPEG实际识别四行`FolioPath OCR 2026`、`中文文字识别测试`、`订单编号123456`、`LOCAL TEST ONLY`；宽图输出2048x410后识别`WIDE IMAGE OCR 2048`。源图与两权重hash前后一致，模型仅从独立只读OCR实验缓存加载，离线/无端口/无原库或DB挂载。
- 桥接使用既有S0镜像 `sha256:a8ac357f181861c713a08fa4fa4acad175e689b84973b9979e3c4f53c756a1a7` 的实验worker，独立进程显式4GiB预算；**不是生产OCR协议、producer、共享调度或后台索引联验**。新input hash与旧S0实验identity分别记录，未混用或激活。
- `make fmt-check arch-check generate-check lint`、Go全量/integration、定向Go race与应用runtime E2E通过；Go测试使用独立extended临时目录。无Web改动，本轮UI/三浏览器矩阵未执行。没有native amd64、真实普通照片质量、全应用+模型联合容量或发行证明。
- 首轮新Docker测试target链接少了现有LD_LIBRARY_PATH，补齐测试target环境后重建通过；正式runtime未改此设置。桥接runner首轮未给docker stdin加`-i`，worker明确返回空输入invalid_media，修正输送后真实识别通过，未改预期或伪造文字。全部初始日志保留。

[机器证据](../evidence/ocr/2026-10-25-ocr-image-input-local.json)记录固定镜像、源码/输出/log hash及未执行项；测试容器均已删除，未提交推送。

## 后续

1. 下一步审查producer和共享scheduler能力注册：现有scheduler源码参与旧runtime hash，不能无声改变CLIP/face/tag身份，也不能新建第二准入/生命周期owner。当前没有OCR服务endpoint或部署设置。
2. 再定义源指纹/独立generation、missing/失败归属、租约/fenced提交/checkpoint、停机/取消、清理与权限，才实现后台OCR索引和融合搜索/UI。
3. S0的无EXIF 90°图0/4仍是已知缺口；本片只修正已有EXIF声明方向，不负责猜测任意旋转、多栏/表格或通用准确率。10源问题继续按用户要求暂留，原6010与人工整理不动。
