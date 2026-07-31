# WCH-001：Linux watcher 可行性 spike

## 状态

- 状态：Passed（WCH-S0 架构可行性范围）
- 日期：2026-07-29
- Feature：[FTR-SCN-001](../features/automatic-library-discovery.md)
- Gate：[WCH-S0 Architecture Ready](../gates/POST-MVP-2/wch-s0-architecture-ready.md)
- Probe：[spikes/wch-linux-watcher](../../spikes/wch-linux-watcher/README.md)

## 目标

在不接入生产 import graph、SQLite、HTTP 或 Web 的情况下验证：

- 目录级 inotify 的 create、close-write、move、delete、new-directory 事件；
- rename cookie 是否可用于合并提示但不承担删除证明；
- 事件路径通过 openat2 安全重开时拒绝 symlink；
- watched root 被替换时，配置路径身份变化可被识别；
- 1 万目录 watch 的建立时间、FD 和 RSS；
- 不读取事件时能否观察 `IN_Q_OVERFLOW`；
- 宿主 inotify 上限与无法注入的 ENOSPC/unmount 缺口。

## 探针设计

独立 Go module 在临时树上：

1. 监听根目录，创建新目录后再为其注册目录 watch；
2. 写入、`fsync`、关闭、重命名并删除媒体样本；
3. 创建指向树外文件的 symlink，以与生产相同的 openat2 resolve flags 尝试重开；
4. 移走 watched root 并在原路径创建新目录，比较锚定 FD 与配置路径 identity；
5. 创建并注册 10,000 个目录 watch，读取 `/proc/self/status`、`/proc/self/fd` 与建立耗时；
6. 暂停读取后创建 50,000 个文件事件，再检查 `IN_Q_OVERFLOW`。

探针输出结构化 JSON，不修改 sysctl，不声明网络文件系统或 mount namespace 证据。

## 已执行

第一次尝试执行：

```sh
docker version --format '{{.Server.Version}} {{.Server.Os}}/{{.Server.Arch}}'
```

结果：本机 Docker daemon 未运行：

```text
Cannot connect to the Docker daemon at unix://<local-docker-socket>.
```

启动 Docker Desktop 后，Docker Engine `29.6.2` 提供
`linux/arm64`、内核 `6.12.76-linuxkit`。使用只读仓库 bind mount 和隔离 tmpfs 执行：

```sh
docker run --rm --platform linux/arm64 \
  --mount type=bind,src="$REPOSITORY",dst=/src,readonly \
  --tmpfs /work:rw,exec,size=2g \
  -w /src/spikes/wch-linux-watcher \
  -e GOMODCACHE=/work/modcache \
  -e GOCACHE=/work/buildcache \
  -e GOTMPDIR=/work/tmp \
  golang:1.26.4-bookworm \
  sh -c 'mkdir -p /work/tmp &&
    go build -o /work/wch-probe . &&
    /work/wch-probe -watch-directories 10000 -overflow-events 50000'
```

结果：

- 实际观察 `create`、`close_write`、`moved_from`、`moved_to`、`delete`、`move_self`
  和 `queue_overflow`；
- rename 的非零 cookie 成功配对；
- symlink 安全重开返回 `ELOOP`，没有打开树外目标；
- watched root 替换通过 FD/path device+inode 比较识别；
- 创建 `50,000` 个文件并暂停读取后真实观察 `IN_Q_OVERFLOW`；
- 宿主 `max_user_watches=1,048,576`、`max_queued_events=16,384`；
- 共 `10,002` 个目录 watches，建立耗时 `272.699ms`；
- RSS 从 `8,368 KiB` 增至 `10,732 KiB`，增量 `2,364 KiB`；
- FD 从 `9` 保持为 `9`，证明 inotify watches 不按目录增加进程 FD。

另以 `linux/amd64` 模拟执行相同 10k/50k 档：

- 事件、rename、root replacement 和 overflow 均观察到；
- 10,002 watches 建立耗时 `304.406ms`，RSS 增量 `900 KiB`，FD `15 → 15`；
- 模拟层的 `openat2` 返回 `ENOSYS`；探针和 FolioPath 均按失败关闭处理，不能把该结果记录为
  原生 amd64 路径边界通过。

当前开发环境是 `go1.26.5 darwin/arm64`。还执行：

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -o /tmp/foliopath-wch-probe-arm64 .
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -o /tmp/foliopath-wch-probe-amd64 .
file /tmp/foliopath-wch-probe-arm64 /tmp/foliopath-wch-probe-amd64
```

结果：两个构建均成功，分别得到静态链接的 Linux AArch64 与 x86-64 ELF。

## 结论、限制与后续 Gate

- **WCH-S0 结论：Passed。** 目录级 inotify、事件 hint、锚定安全重开、root identity、
  overflow 和 10k watch 资源模型在 native-architecture linux/arm64 可行。
- 10k watch 当前实测增量很小，但只是一台 Docker Linux 环境，不能直接冻结生产预算。
- `ENOSPC` 依赖宿主 sysctl，当前未通过降低上限真实注入。
- `IN_UNMOUNT` 和 nested mount 仍需隔离、特权 mount namespace。
- linux/amd64 当前只有模拟事件证据；`openat2` 按预期失败关闭，不能替代原生证据。

这些剩余项转入 WCH-S1 资源/错误合同和 WCH-S2 Backend Evidence Ready，必须在生产
实现获准前由自动测试与原生 linux/amd64/arm64 证据关闭。它们不再阻断架构方向进入
Contract Ready。
