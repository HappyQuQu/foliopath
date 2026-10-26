# ADR-0034：OCR producer 与兼容共享准入路由

日期：2026-10-25；状态：Accepted for isolated producer integration。
范围：CR-2026-048 / POST-MVP-5后续OCR / FR-OCR-001～003、NFR-INT-007/009/010。
用户继续开发授权下的技术子片；不签署全OCR质量/存储/产品启用或发行。

## 决定与兼容边界

固定photo-tags-v2基础镜像的旧源文件和模型缓存清单保持字节不变。OCR采用同一个HTTP listener上的独立identity/predict接口、新worker、独立只读/cache子集（/ocr-cache）和OCR runtime/input/output身份；不加载媒体/数据库，不新增部署单元或第二队列。

注册adapter继承原InferenceScheduler，始终委托原infer拥有pending≤2、单lock、26秒总截止与取消，原_run/_reap/close拥有子进程/scratch回收。仅用每请求ContextVar保存真实能力kind；OCR为了进入冻结v1核心的白名单，以既有visual**准入类别**调用super.infer，但不调用CLIP逻辑。持锁进入_run后用真实`ocr`参数、固定worker/cache调用原_run，finally在原child完成reap后恢复旧route。不是OCR的HTTP协议、模型kind、缓存/输出或身份使用visual；实验visual worker不进入生产图。其余能力原kind/command/cache照旧，无新增admission/timeout/retry代码。

这是一层明确的兼容路由，不更改/重解释旧runtime digest：旧runtime hash继续覆盖其冻结源/worker合同，而非对任意overlay或最终OCI的认证。OCR runtime digest覆盖固定baseOCI、adapter/worker/HTTP/生成合同源码。新镜像整体仍需独立不可变digest验证；只有旧文件/身份和实际输出回归通过才可保留旧空间。不允许修改旧scheduler/server源文件后沿用其hash，不允许把core准入类别映射用于其他任意能力。

启动先完成原base lifespan；在尚未提供HTTP且旧owner无请求/child时，关闭空owner，替换为一个继承相同核心的新路由owner。之后所有旧/OCR请求只走该owner；shutdown仍由原base lifespan关闭它。无并发两个live owner。

OCR输入仅接受ADR-0033的RGB8非交错PNG、32～2048/≤4Mi像素、≤8MiB；HTTP只做有界格式/CRC/元数据检查，child负责像素解码与ONNX，不再EXIF/缩放/颜色变换。模型snapshot复用原cache_snapshot，startup及child独立校验固定两份权重，禁下载。缺失OCR缓存只关闭OCR能力，不伪称ready，也不让旧能力失效。

输入合同由internal/ocr唯一持有，输出策略由同一owner生成JSON供producer读取；确定性生成加入generate-check。Go/Python用共享正反fixtures验证一致。源输入/输出/模型/临时故障分开，错误带OCR指纹且不含原文/异常。共享body限额复用原BoundedBody，OCR专属边界只规范化其固定错误，不更改旧路由。

## 验证与剩余

调度unit证明真实child kind/cache、跨族pending2/active1、并发ContextVar隔离、取消/超时/断连与回收后恢复；冻结文件hash/identity与基线相等。真实模型隔离Docker用Go输入+Go客户端调用OCR HTTP，另比较旧语义/tag/face实际结果；错/缺/篡改OCRcache失败关闭。

主应用app/SQLite/API、配置启用、源代次/持久任务/清理和搜索/UI仍未授权组合；6010不部署。90°无EXIF与版面/准确率、native双架构、联合容量和发行另验。兼容路由若以后需新增能力/改变核心，必须复审并优先设计正式可注册核心，不扩展本例外。
