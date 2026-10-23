# FIX-2026-09-01：人脸安全读取投影合同补全

- 目标版本：`POST-MVP-5` revision 2 / S2C
- 关联 Gate：`INT-S2C Privacy Ready`（保持 **Release No-Go**）
- 影响任务：`INT-248`
- 影响不变量：多人图片只按 face ID 操作；API 不暴露 embedding、crop、精确 bbox、路径或身份推测

## 问题

已批准的 S1R2 资源语义要求人物资产、匿名组详情和单资产多人脸读取，但冻结 OpenAPI 只包含人物与匿名组
列表，缺少这三个消费者读取合同。若由后续 UI 自行拼接，会复制游标、覆盖率和隐私投影规则。

## 修复

- 补充人物资产稳定 keyset 页、匿名组成员稳定 keyset 页和单资产多人脸投影合同。
- 人物资产 cursor v2 同时绑定 person revision 与其所有 bound source library 的状态 revision；任一来源
  offline/not-ready 时返回 `face_not_ready`，不再静默过滤后把不完整结果呈现成空人物。查询后再次复核
  source snapshot，覆盖列表期间的状态竞争。
- catalog hydration 逐项绑定 library ID + asset ID + 原顺序，并要求来源仍 available 且媒体类型为
  image/animated；资产删除、短页、乱序或跨库替换返回 cursor stale，hydration 期间转 offline 返回
  `face_not_ready`，不会降级成 500 或串入其他资产。
- 人物资产复用既有 `Asset` DTO；额外只返回该人物在资产中经人工确认的 opaque face ID。
- 人脸和组成员只返回 opaque face ID、asset ID、角色/状态、revision 及整数百分比粗略区域。
- 明确禁止 embedding、crop、模型分数、精确 bbox、路径和身份推测字段。
- 架构 fitness test 解析全部 face HTTP JSON tag/literal，拒绝 embedding/vector/crop/path/source fingerprint、
  detector/quality/score/similarity 和 raw runtime output；同时禁止 face capability/SQLite adapter 直接导入
  日志包，并禁止现有媒体诊断与系统日志依赖 face 状态。
- 合同与 adapter 可以先完成测试，但生产组合继续由 S2C Gate 失败关闭。

## 回归证据

- SQLite 覆盖多人资产、active cluster build、游标篡改、revision/build 过期和区域边界。
- SQLite/HTTP 额外覆盖人物来源 offline 首屏/续页、恢复后旧 cursor stale 与 `409 face_not_ready`。
- HTTP hydration 覆盖资产消失、短页、跨库替换、offline 和错误媒体类型的失败关闭映射。
- HTTP wire 测试覆盖三个投影无敏感字段并拒绝未知/重复 query key。
- `go test ./tests/architecture -count=1` 覆盖上述防泄露边界。
- OpenAPI lint、生成一致性和架构 Gate 纳入完整验证。
