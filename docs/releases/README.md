# FolioPath 版本范围清单

每个冻结版本使用一份只追加的 scope manifest，固定需求、非目标、验收 ID 和修订号。
产品需求文档可以补充说明，但不能通过直接改写它来悄悄改变已冻结版本。

- [MVP-2026-07-23 scope revision 1](MVP-2026-07-23-scope.md)

已合入的 manifest 不原地改写。范围变化先创建 Change Record；获批后新增 revision 文件或下一版本
manifest，并在新旧文件中链接替代关系。安全不变量不能通过 scope-budget exception 移除。
