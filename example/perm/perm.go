package perm

import (
	"github.com/donnol/enum"
)

type Perm string

func (p Perm) String() string { return string(p) }

var Perms = enum.InitFor[Perm, struct {
	enum.Struct[Perm]
	UserCreate Perm `enum:"/user/create,新建用户"`
	UserModify Perm `enum:"/user/modify,修改用户"`
	UserDelete Perm `enum:"/user/delete,删除用户"`
	UserList   Perm `enum:"/user/list,查询用户"`
}]()

func init() {
	// Run enumlint --enum-pkg=github.com/donnol/enum ./...
	//
	// got:
	// 🚨 enum check is bad 💥
	// Location    Kind      Target
	// --------    ----      ------
	// perm.go:20  variable  Perms
	// perm.go:28  field     Perms.UserCreate
	// perm.go:30  field     Perms.UserList
	// ⚠️  请修正后重试！
	Perms = struct {
		enum.Struct[Perm]
		UserCreate Perm "enum:\"/user/create,新建用户\""
		UserModify Perm "enum:\"/user/modify,修改用户\""
		UserDelete Perm "enum:\"/user/delete,删除用户\""
		UserList   Perm "enum:\"/user/list,查询用户\""
	}{}

	Perms.UserCreate = ""

	Perms.UserList = ""
}
