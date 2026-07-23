# ADR-0008：统一应用组合根并分离纯路径策略

## 状态

已接受（2026-07-23）；修正 ADR-0001 中关于 `cmd/foliopath` 责任的表述

## 决策角色

- 产品：N/A，不改变用户行为或版本范围
- 架构：系统架构基线评审
- 安全：路径策略仍保持失败关闭，真实文件访问仍只属于 files adapter

## 背景与驱动因素

ADR-0001 曾把配置、依赖组装、启动和优雅退出都写入 `cmd/foliopath`，而后续模块边界把
`internal/app` 定义为唯一组合根。两种规则无法同时实施，也会诱使 `main` 累积生命周期和业务接线。

路径 spike 还需要 library、scanner 与真实文件 adapter 复用同一套相对路径词法规则。若规则放在
`internal/files` 下，能力层看起来依赖外层 adapter；若各能力自行实现，又会产生安全策略漂移。

## 备选方案

1. **由 `cmd/foliopath` 组装全部依赖。** 文件少，但入口会知道数据库、HTTP、worker 和停机细节，
   难以复用应用生命周期并违反单一组合根。
2. **由 `internal/app` 统一组装，`cmd` 只交出进程控制。** 依赖方向清楚，可集中管理资源和停机。
3. **把纯路径规则留在 files adapter 或各 capability。** 前者造成层次歧义，后者复制安全逻辑。
4. **建立无 I/O 的内层 `internal/pathpolicy`。** 共享词法不变量，同时保留真实文件打开的 adapter 边界。

选择方案 2 与 4。

## 决策

- `cmd/foliopath` 只处理 Go 进程入口所需的最小 argv/env 交接、调用 `internal/app.Run` 并把退出结果
  返回运行时；不拥有业务依赖图、迁移、路由、worker 或优雅停机状态机。
- `internal/app` 是唯一组合根，拥有配置加载/校验、具体依赖接线、全局资源预算、根 context、启动、
  readiness 与有界停机顺序。
- `internal/pathpolicy` 是内层纯策略包，只验证并保持相对路径文本；不得访问 OS、数据库、HTTP 或
  业务状态。library、scanner 和 files 可以依赖它。
- `internal/files` 仍独占 `/library` 根身份、真实打开、symlink/特殊节点与设备边界；pathpolicy 成功
  不等于文件访问已获授权。
- 当前及未来的 Go import 方向由 `make arch-check` 检查；扩大 pathpolicy 职责或改变组合根需新 ADR。

## 后果

- 入口保持可审计，生命周期与全局并发不会分散在 main、handler 或 capability。
- 相对路径词法只有一个实现，同时避免业务包依赖具体文件 adapter。
- 增加了一个很小的内层包；它不能演变为通用 `utils`，也不能承载真实路径或领域状态。

## 验证与复审

- Fitness function：AF-001、AF-002、AF-006。
- 当前证据：`make arch-check`、pathpolicy/files/library/scanner 单元与集成测试。
- 若 Go 进程需要多个独立应用组合或 pathpolicy 开始拥有 I/O，应复审本 ADR。
