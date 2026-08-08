# perm

这是一个在初始化enum变量后，还对其进行修改的示例。

只要在项目里执行`enumlint`，就能快速检测出违规的写操作。定位到它们的位置后，就可以快速修正了。

Build: `go install ../../cmd/enumlint/`.

Run `enumlint --enum-pkg=github.com/donnol/enum ./...`.

Got:

```sh
🚨 enum check is bad 💥
Location    Kind      Target
--------    ----      ------
perm.go:20  variable  Perms
perm.go:28  field     Perms.UserCreate
perm.go:30  field     Perms.UserList
⚠️  请修正后重试！
```
