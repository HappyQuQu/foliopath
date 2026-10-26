# 大 JPEG 语义输入兼容修复

范围：CR-2026-030 / FR-INT-002、005 / NFR-INT-002、003，POST-MVP-5 P4；Owner semantic/imagevips/app。修复既有图片分析切片，不新增 API、存储或部署边界。

失败原因：语义预处理在长边 1024 resize 前拒绝超过 16 MP 的源图；进一步复验发现其中 sample-062.jpg 为 58668512 bytes，还超过 32 MiB 源字节预算。仅修复像素限制后为 89/90，保留失败记录。

方案仅允许 JPEG 利用 decoder shrink-on-load，在第一次 native load 前选择 1/2/4/8 中最小可满足 16 MP 工作像素预算的倍率；源尺寸仍由 media.ValidateImageDimensions（JPEG 180 MP 及边长限制）校验。JPEG 源字节预算单独提高至 64 MiB（仍低于通用媒体的 256 MiB），其他格式保持 32 MiB；8 MiB 输出、1024 长边及 16 MP 工作预算不变。加载后再次检查真实格式、工作像素，失败关闭；不修改原图。额外源缓冲风险必须通过 1 GiB 容器复验，不把理论预算当实际峰值。

预处理算法合同升级 remote-image-v2，纳入 semantic.SpaceIdentity 的 SHA256，沿用现有代次变更/重新回填机制，旧向量不能冒充新代次。外部 ML 归一化/裁剪及服务身份不变。无需新迁移或公共 API 变化。

字节读取先在已打开的 seekable 流上验证长度，再一次分配准确缓冲区，拒绝读期间大小变化，避免 ReadAll 扩容副本。保留取消检查、格式验证及单一 worker 所有权。

验证计划：倍率边界、损坏/伪格式、源大小/尺寸越界、旧小图行为、超大/渐进式 JPEG 及输出尺寸；本地 Docker 1 GiB 下固定匿名 90 图/10 视频完整探针。未运行的门禁不记通过，P4 完整门禁保持开放。[执行证据](../evidence/int-001/p4-semantic-large-jpeg-2026-09-17.md)。
