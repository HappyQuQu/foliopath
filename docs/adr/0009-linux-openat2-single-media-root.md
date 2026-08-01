# ADR-0009：Linux `openat2` 与单一媒体根挂载

## 状态

已接受（2026-07-23）；修订 [ADR-0002](0002-library-path-model.md) 中允许把多个宿主机目录分别挂载到 `/library` 子目录的部署形式；发布身份部分由 [ADR-0012](0012-root-runtime-bind-data.md) 替代

## 决策角色

- 产品：保持“Docker 只声明一个大媒体边界，多个媒体库在界面内配置”的已确认行为
- 架构：`internal/files` 继续拥有唯一真实文件访问边界
- 安全／发布：Linux 使用内核级路径解析约束；不满足运行条件时失败关闭

## 背景与驱动因素

本决策落实 `FR-DEP-001～003`、`FR-LIB-001～003`、`NFR-SAFE-001`、
`NFR-SEC-001` 与 `NFR-COMP-001`。管理员希望首次部署只映射一个大媒体目录与一个
数据目录，之后在 Web 设置中创建多个媒体库并选择任意普通子目录；应用同时必须
保证原媒体只读、路径不能逃逸且扫描不会越过授予的媒体边界。

[FS-01](../spikes/fs-01-path-boundary.md) 证明，只比较目标的 device/inode 或在打开前
检查规范路径，无法拒绝 `/library` 下的同设备 bind mount。攻击或错误部署可以在
设备号不变的情况下把另一个目录接入已批准的树；检查后再打开也存在竞态。Linux
`openat2` 能以已锚定的目录文件描述符为起点，在一次内核路径解析中同时禁止逃逸、
符号链接和 mount crossing。FS-01 的同设备、跨设备与 self-bind 探针验证了该行为。

这意味着两个目标必须一起成立：容器内只能有一个媒体根挂载，而多个产品媒体库是
该根下的普通目录，不是多个 Docker 子 volume。

## 备选方案

1. **继续允许 `/library/<name>` 子挂载，只比较路径、device/inode。**
   Compose 可以直接拼接多个宿主机目录，但同设备 bind mount 无法可靠识别，
   检查与使用也可能竞态。FS-01 已证伪，拒绝。
2. **允许嵌套挂载，并建立 mount ID 白名单与动态重验证。**
   这会把部署拓扑、mount namespace 变化和不同内核/文件系统行为引入产品配置，
   显著扩大信任边界与测试矩阵，也违背减少 Docker 配置的目标。MVP 不采用。
3. **只使用 Go `os.Root`、realpath 或用户态逐段打开。**
   这些机制可以限制词法逃逸和符号链接，但不提供与 Linux
   `RESOLVE_NO_XDEV` 等价的原子 no-mount-crossing 保证。不能作为 Linux 发布边界。
4. **只挂载一个 `/library` 根，并在 Linux 用 `openat2` 失败关闭。**
   UI 仍可在根下创建任意多个非重叠媒体库并递归包含子目录；部署与安全模型都更小。
   选择此方案。

## 决策

- 容器只支持一个媒体挂载目标：`/library`。`/app/data` 是独立的可写数据挂载，
  不属于媒体挂载。`/library` 必须只读。
- `/library` 自身可以是 Docker/OCI 提供的 bind mount 或 volume；其下必须是同一
  呈现文件系统中的普通目录树。禁止把任何 volume、bind mount 或其他 mount point
  嵌套到 `/library` 的后代路径。
- 多媒体库是产品层配置，不是部署单元。管理员可在 UI 中选择 `/library` 本身或其
  任意普通后代目录；每个媒体库默认递归包含该目录的全部普通后代。重叠规则保持不变。
- Linux 上所有真实媒体文件和目录打开都必须从已锚定的 `/library` 目录文件描述符
  开始，并使用 `openat2` 的
  `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV`。不能以先检查路径、
  再通过另一条调用链打开文件代替该约束。
- Linux 内核必须实现 `openat2` 及上述全部 resolve flags，容器 seccomp/LSM
  也必须允许该系统调用。出现 `ENOSYS`、不支持的 flags/结构或策略阻断时，
  `internal/files` 必须返回明确的边界不可用错误并阻止媒体根进入可用状态；
  不得静默回退到 `os.Root`、realpath 或 device/inode 检查。
- 非 Linux adapter 只用于本地开发和补充测试。其成功结果不得被登记为与 Linux
  同等级的 mount-boundary 证据，也不扩大 MVP 的 Linux 发布平台承诺。
- 若媒体分布在多个独立宿主机卷，部署者只能先在宿主机侧提供一个、对容器表现为
  单一文件系统且没有后代 mount crossing 的呈现根，再把该根一次性只读挂载到
  `/library`。FolioPath 不选择、配置或承诺某种 union、聚合文件系统或 NAS 技术；
  具体方案须由部署者验证，且应用的 `NO_XDEV` 检查仍是最终边界。
- 发布镜像必须验证非 root、`/library:ro`、默认安全配置下的 `openat2` 可用性。
  需要 `CAP_SYS_ADMIN` 的 mount 探针只在隔离测试环境运行；生产容器不因此获得该
  capability。

## 后果

- 最小部署保持两个 volume：一个大的只读媒体根和一个可写数据目录。已有根中的新
  普通子目录无需修改 Compose，管理员可直接在 UI 创建媒体库并包含其子目录。
- 同设备 bind mount、跨设备 mount、符号链接与 `..` 逃逸由一次内核路径解析共同
  拒绝，避免用户态检查与打开之间的竞态。
- 不能再用多个 `/library/<name>` 子 volume 拼装媒体命名空间。已有此类部署必须
  改为一个共同宿主根或经验证的单一呈现根后才能使用 FolioPath。
- 运行环境存在最低能力要求：Linux 内核与 seccomp/LSM 必须允许所需 `openat2`
  行为。版本号本身不足以证明兼容，发布和启动检查必须验证实际系统调用。
- 该策略有意放弃遍历 `/library` 后代 mount point 的能力。未来若要支持多个媒体
  根挂载，必须新建 ADR，重新定义 mount 身份、动态变化、离线语义、部署契约和
  双架构安全验收，不能仅放宽一个 flag。

## 验证与复审

- 对应 architecture fitness function：`AF-006`。
- 当前证据：[FS-01 路径边界 spike](../spikes/fs-01-path-boundary.md)；Linux
  非 root 单元/集成测试；带 `fsboundary` tag 的同设备、跨设备与 self-bind
  mount namespace 探针。
- 发布前仍需：linux/amd64 与 linux/arm64 最终镜像、默认 seccomp、真实只读
  `/library` volume、运行期挂载/权限变化和生产 HTTP 链路验证。
- 需要复审本 ADR 的条件：产品要求多个媒体挂载目标、允许嵌套 mount、支持新的
  服务端操作系统，或运行平台无法稳定提供所需 `openat2` 语义。
- 替代／被替代关系：仅替代 ADR-0002 中“多个宿主机目录可分别挂到
  `/library` 子目录”的部署形式；ADR-0002 的单一允许根、UI 多媒体库和相对路径
  模型继续有效。
