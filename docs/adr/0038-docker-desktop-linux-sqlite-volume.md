# ADR-0038：Docker Desktop 的 SQLite 数据使用 Linux 数据卷

状态：Accepted，2026-10-26。NFR-OPS-001/INT-010，既有本地6010 OCR恢复维护，承接ADR-0023。

## 证据

宿主共享目录上的64MiB合成库，在单写者、两个独立只读完整性检查进程下，先后复现SQLITE_PROTOCOL(15)和`_walFramePgno`的SIGBUS；固定Go1.26.5再测仍出现PROTOCOL(15)。同一探针与SQLite版本在Docker管理的Linux卷完成1280次更新和160次检查。小库120秒/213次检查正常，说明短小用例不能否定大WAL边界风险。未拿真实图库做测试，未从宿主打开活跃DB；不能逐次证明历史原库退出均由此产生。

SQLite的WAL-index依赖同文件共享内存映射（[官方说明](https://sqlite.org/walformat.html)）；Docker Desktop的宿主bind经虚拟文件共享，而[命名卷](https://docs.docker.com/engine/storage/volumes/)由Linux daemon管理。采取隔离该文件共享层的部署修复，不吞掉未知数据库错误或替换驱动。

## 决策

- Docker Desktop本地真实部署将完整`/app/data`置于预先创建并核验的外部Linux命名卷；原`/library:ro`和AI模型缓存/身份边界不变。原生Linux本地bind部署仍受ADR-0023约束，不泛化为所有bind均不支持。
- 可选第五覆盖`compose.ai.data-volume.yaml`仅按目标`/app/data`替换挂载。external卷必须显式提供，不允许遗漏后默默创建空库；保留base/unlimited-memory/multilingual/ocr及同一项目。
- 主应用正常退出、完整停写备份data/私有配置/全部覆盖并逐文件校验后，在独立容器将只读源复制到新卷，核对完整清单、只读保护摘要和database-check。再启动主应用；禁止两个应用同时拥有同一DB。
- 原data目录和完整备份保留，不在运行中同步。切换后它们是历史副本，不再从旧宿主目录读取在线状态。后续在线审计/检查只在原主容器Linux锁域；停写备份从已停止的卷导出并核验。回退先停主，核对新卷是否有新写入，不能直接启动旧陈旧目录覆盖进度。

## 后果

没有新增服务、数据库、任务或公开接口。卷仍由Docker Desktop管理，删除卷/重置Docker可丢失数据，完整独立备份依然必需。此修复通过有界合成复现与恢复验证，不宣称无限期稳定、质量、容量或发行门通过。
