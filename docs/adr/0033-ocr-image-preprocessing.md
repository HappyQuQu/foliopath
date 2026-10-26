# ADR-0033：独立 OCR 图片预处理合同

日期：2026-10-25；状态：Accepted for image-preprocessing-only implementation。
范围：CR-2026-047 / POST-MVP-5后续OCR / FR-OCR-001～003、NFR-INT-007/009/010。
产品授权来自用户继续AI开发；本技术子片不代表用户签署质量/生产OCR启用或发行。

## 背景与选择

CR-2026-045已证明横排OCR离线推理可行，CR-2026-046已有严格Go传输消费者；主应用安全图片输入还没有合同。现有语义输出长边1024，直接复用会无声降低OCR小字保留。实验PIL直接处理源图不能成为主应用文件边界。

选择imagevips拥有独立OCR transform，不改变语义/tag/face profile，不新增native并发门或第三服务。原件仍经未来internal/files打开；预处理只接已打开stream，不接受路径。不在HTTP handler调用vips。

## 决定

- ocr拥有`ocr-rgb-png-v1`输入合同的canonical JSON及hash、PreparedImage和ImagePreprocessor接口。imagevips实现，复用既有全局大图准入和readRemoteSemanticSource有界读取机制；不新增通用包或独立资源预算。
- JPEG/PNG/WebP/GIF仅首帧；source≤64 MiB、源像素≤40,000,000、最小边32。编码buffer读取前按既有策略取得全局native准入；native load前根据有界header校验声明格式与源几何。使用buffer固定本次输入，避免header校验后改读另一个源快照；未来任务仍需源指纹复核。
- EXIF归正，转换sRGB/8bit、透明度白底合成，最长边2048且不放大，输出无元数据RGB PNG。最终最小边32、像素≤4*1024*1024、编码≤8MiB，过于狭长或输出超限明确input_limit，不偷偷换profile/有损压缩。
- 尺寸/预算超限与确定的头部不可解析分开；调用参数错误、读/seek/processing失败和native缺失有独立错误。取消优先、未知native/IO错误不等于坏源。编码输出只在全部成功时返回，ImageRef在共享准入释放前关闭并完成native回收。
- 原语义、人脸、自动标签和模型身份不变。此输入hash必须在将来的OCR独立身份中显式绑定，不能自动填充部署或复用实验hash。当前app/SQLite/API仍不得消费OCR；只扩展允许imagevips预处理实现。

## 验证和后续

原生Docker回归包含旧路径、共享大图取消、格式/header/像素、alpha/EXIF/元数据、输出上限、无进展/短读与读取取消。合成中英文经过本transform后送独立OCR实验模型核对；单arm64证据不等于native双架构/联合资源/质量/发行。

服务producer及共享调度注册仍需接受：现有scheduler源码参与旧runtime合同，不能为注册OCR无声重标其他能力或建立第二准入owner。源指纹/代次、任务事务、持久化/清理与融合搜索/UI仍需各自合同。90°/版面质量不由EXIF归正解决。
