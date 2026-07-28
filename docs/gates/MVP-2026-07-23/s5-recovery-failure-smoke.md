# Stage 5 恢复与失败关闭候选演练

## 结论

**Go — `S5-004A/004B` 的恢复、失败关闭、真实前一候选升级与配对数据回滚已在
原生 linux/arm64 和 linux/amd64 通过。**

`tests/release/recovery_smoke.sh` 已并入 `make test-release-image`；
`tests/release/upgrade_rollback_smoke.sh` 另要求两个不同的不可变 image ID，以旧候选
创建真实管理员状态，当前候选直接打开旧数据，再用“旧镜像 + 升级前离线备份”完成回滚。
它没有用同版本重启冒充升级，也没有让旧镜像直接打开升级后的唯一数据副本。

## 范围与所有权

- 目标版本：`MVP-2026-07-23`
- Stage / task：Stage 5 / `S5-004A`、`S5-004B`
- 需求/质量：`FR-DEP-001～004`、`NFR-REL-001～002`、`NFR-SAFE-001`、
  `NFR-OPS-001`
- owner：`tests/release/recovery_smoke.sh` 拥有候选容器恢复 fixture；
  `internal/app` 与 SQLite adapter 继续拥有启动、migration、WAL 和失败关闭语义
- 所有读写只发生在测试创建的临时数据/媒体目录；真实媒体不会被读取或修改

## 已验证

候选镜像使用真实应用二进制和 schema 执行：

1. 在只读根与只读 `/library` 下首次启动并创建真实管理员状态；
2. 优雅停止后离线归档完整 SQLite family，同时省略可重建 `cache/` 与 `tmp/`；
3. 恢复到新空目录，验证管理员初始化状态保留、cache/tmp 自动重建；
4. 对恢复数据执行同版本二次启动，验证当前 migration 幂等；
5. 运行中发送 `SIGKILL`，确认退出码 137；随后重启并通过 readiness 与管理员状态验证，
   证明 SQLite/WAL 恢复；
6. 在已耗尽的 64 KiB `/app/data` 上启动，应用非零退出且不进入 readiness；
7. 使用损坏 SQLite 文件启动，应用非零退出且不进入 readiness；
8. 每个步骤前后媒体 sentinel SHA-256 保持不变。

升级/回滚入口还验证：

1. 前一候选与当前候选解析成两个不同的不可变 image ID；
2. 前一候选初始化的数据在停止并完成离线备份后由当前候选成功启动；
3. 回滚目录只从升级前备份恢复，再与前一候选配对启动；
4. 两个方向都保留管理员初始化状态，媒体 sentinel 保持不变。

2026-07-28 本机原生 linux/arm64：

```text
tests/release/recovery_smoke.sh <candidate-image>
candidate recovery smoke passed
```

同日原生双架构升级/回滚结果：

```text
linux/arm64 previous=sha256:ca5c9c7d... current=sha256:062d2160...
linux/amd64 previous=sha256:ff9ea318... current=sha256:a321a10e...
candidate upgrade and paired rollback smoke passed
```

由于仓库 Actions Billing 在 runner 分配前失败，amd64 使用发布负责人指定的原生服务器
重复同一脚本，arm64 使用本机原生 Docker；这关闭 `S5-004` 的产品/运行证据，但不冒充
CI artifact。当前证据仍不授权在线复制活动 SQLite，也不承诺 schema 向后兼容；未来
新增 migration 时必须继续用同一入口对当时真实前一发布 digest 复跑。
