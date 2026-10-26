# 独立人脸服务合同与验证

范围：CR-2026-030 / FR-INT-009 / NFR-INT-002、004、005；POST-MVP-5 S2 接入子片。
Owner：inference 负责私有传输/输出规范化，face 负责质量、代次、聚类及持久化；app 负责组合。
沿用 ADR-0019/0020 两容器边界，不添加部署单元，不变更公开 API。

## 先行合同

- 私网 `GET /api/v1/face/identity`：version=1、protocol=foliopath-face-v1、model=buffalo_l、dimension=512、缓存和运行时 hash。不得当作 CLIP 身份或激活凭据。
- 私网 `POST /api/v1/face/predict`：唯一 multipart 字段 image，最多 8 MiB；只允许单帧 JPEG/PNG/WebP，最长边 1600、总像素不超过 1600²；不接受 URL、路径、模型名和阈值。
- 固定 detection+recognition，检测阈值 0.7；共享服务并发上限 2。响应最多 64 个 faces，每项为归一化 box、detection 和 L2 归一化 512 维 embedding。不是姓名或独立质量分数；不得把 detection 冒充质量。
- 越界框裁剪到图片；空框、无穷/NaN、零向量、维度错误、超过 64 张脸整次失败，不静默截断。零人脸是有效成功。
- 未就绪/模型缺失/推理错误返回 503，输入无效 400、正文过大 413；不泄露上游错误。缓存必须含两个固定 ONNX 文件且通过已有逐文件 hash 快照验证。
- 镜像保留默认 CLIP 模式；构建参数 `FOLIOPATH_INCLUDE_FACE=1` 才纳入人脸缓存。加入能力会改变缓存/运行时身份，旧部署身份不可沿用。
- 本子片不是主应用接入、非空识别、质量校准、聚类/人物 UI 或发行完成。Go 人脸适配、独立代次准入、任务组合仍待实现。

## 验证

本地 Docker Linux/arm64 已执行成功：

- `python -m unittest discover -s /opt/foliopath-inference -p 'test*.py' -v`：13 项通过（新增人脸 6 项 + 原服务/缓存 7 项）。非空框/向量来自模拟输出，非真实人物识别证据。
- `--network none --read-only`，只读本地模型缓存、`/verified-cache` 1200 MiB tmpfs；真实服务 lifespan 校验缓存并加载 upstream，320×240 合成纯色 PNG 推理 HTTP 200、`faces=[]`。确认 recognition batch size=1，未修改原缓存。探针固化为 `services/inference/probe_face.py`。
- `make arch-check`：通过。

缓存身份为 `e6d671f9d9fada323fd77a02d217349f46af8e103eccb297b1ebcac10112efe8`；运行时身份为 `6e5aadf096b691ce0f1e9b6c99a5ddc13fa01a1a326fb3a39b32f4e63b4c42a4`。
固化探针首次按绝对脚本路径执行时 Python 未包含上游工作目录，出现 `ModuleNotFoundError`；改用与服务启动一致的 `python -m probe_face`。不修改上游安装或模型缓存。
所有测试容器退出并自动移除；未替换当前 app/CLIP candidate，未访问 server222 或真实媒体。
未运行完整七项验证、非空真实人脸识别、主应用入库/聚类/人物管理、容量与发行验收。
