# FIX-2026-07-31：双语 README

- 类型：已批准发布文档范围内的例行文档完善
- Gate：沿用 Stage 5 发布文档与 Docker 候选合同，不改变产品、镜像或部署边界
- Owner：根 `README.md` 是默认英文项目入口，`README.zh-CN.md` 是结构同步的简体中文入口
- 行为：两份 README 顶部互相链接，并同步产品定位、界面预览、能力、安全边界、支持格式、快速开始、适用范围、文档入口和许可证
- 部署：Compose 与直接 `docker run` 示例均保留只读根、非 root、capability 丢弃、唯一 `/library` 和 `/app/data` 边界
- 回归：发布文档架构测试同时校验两份 README 的语言入口、候选镜像、启动方式、格式扩展名、视频不转码和不支持格式说明
