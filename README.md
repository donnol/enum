# enum

[![Go Reference](https://pkg.go.dev/badge/github.com/donnol/enum.svg)](https://pkg.go.dev/github.com/donnol/enum)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.26-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

零依赖的类型安全枚举库，支持命名值集合、`Struct` 泛型反射构造，以及 `Tree` 递归多级下拉渲染。

## 快速开始

```go
// 方式一：纯 Enum — 手写 Item 列表
var WeekdayEnum = enum.New(
    enum.ItemFrom("Monday", "周一", time.Monday),
    enum.ItemFrom("Tuesday", "周二", time.Tuesday),
    // ...
)

// 方式二：Struct — struct tag 反射构造
type Priorities struct {
    enum.Struct[Priority]
    Low    Priority `enum:"0,低"`
    Medium Priority `enum:"1,中"`
    High   Priority `enum:"2,高,disabled"`
}
p := enum.InitFor[Priority, Priorities]()
```

## 类型

### `Item[T]` — 枚举成员

| 字段 | 类型 | 用途 |
|------|------|------|
| `key` | `string` | 程序化标识，如 `"UserCreated"` |
| `name` | `string` | 展示名，如 `"用户创建"` |
| `value` | `T` | 枚举值 `int`/`uint`/`string` |
| `disabled` | `bool` | 是否禁用（仅展示，不可选） |
| `ext` | `map[string]string` | 扩展元数据 |

**访问器：** `Key()` `Name()` `Value()` `IsDisabled()` `Ext()`

**构造：** `ItemFrom(key, name string, value T, opts ...ItemOption[T]) Item[T]`

**选项：** `WithDisabled[T]()` `WithExt[T](map[string]string)`

### `Enum[T]` — 枚举集合

有序、逻辑不可变（只有 ext 可写）。并发读安全（`AddExt` 与读调用不并发时）。

| 方法 | 说明 |
|------|------|
| `ByKey(key) (Item[T], bool)` | 按 key 查找 |
| `MustByKey(key) Item[T]` | 按 key 查找，找不到 panic |
| `ByValue(value) (Item[T], bool)` | 按 value 查找 |
| `MustByValue(value) Item[T]` | 按 value 查找，找不到 panic |
| `Index(i) (Item[T], bool)` | 按位置索引 |
| `MustIndex(i) Item[T]` | 索引，越界 panic |
| `Contains(value) bool` | 值是否存在 |
| `All() []Item[T]` | 所有成员（定义顺序） |
| `Keys() []string` | 所有 key |
| `Values() []T` | 所有 value |
| `Len() int` | 成员数量 |
| `Range() iter.Seq2[string, Item[T]]` | Go 1.23 迭代器 |
| `AddExt(itemKey, extKey, extValue string)` | 添加扩展元数据 |
| `GetExt(itemKey) map[string]string` | 获取扩展元数据 |
| `ToMap() map[string]map[string]any` | 转为 map（前端 O(1) 查找） |

**JSON 序列化：**

```json
[
  {"key":"Low","name":"低","value":0},
  {"key":"Medium","name":"中","value":1},
  {"key":"High","name":"高","disabled":true,"value":2}
]
```

### `Struct[T]` — struct tag 驱动构造

通过反射读 `enum:"value,name[,disabled]"` tag 自动构建：

```go
type EventTopics struct {
    enum.Struct[string]
    UserCreated string `enum:"user.created,用户创建"`
    OrderPaid   string `enum:"order.paid,订单已付"`
}
events := enum.InitFor[string, EventTopics]()

// 字段即常量
switch topic { case events.UserCreated: ... }

// 通过 .Enum() 拿到 *Enum[string]，所有 Enum 方法都可用
events.ByKey("UserCreated") // Item[string], true
events.All()                // []Item[string]
events.Enum().MarshalJSON() // JSON
```

### `TreeNode` / `BuildTree` / `AsStringItems` — 多级下拉

用 `ext["parent"]` 将平面列表转为递归树：

```go
type Area int
type Areas struct {
    enum.Struct[Area]
    China     Area `enum:"101,中国"`
    Guangdong Area `enum:"102,广东"`
    Shenzhen  Area `enum:"103,深圳"`
}
areas := enum.InitFor[Area, Areas]()
areas.AddExt("Guangdong", enum.ParentExtKey, "101")
areas.AddExt("Shenzhen",  enum.ParentExtKey, "102")

tree := enum.BuildTree(areas.All())
// JSON:
// [{key:"China",name:"中国",value:"101",children:[
//   {key:"Guangdong",name:"广东",value:"102",children:[
//     {key:"Shenzhen",name:"深圳",value:"103"}
//   ]}
// ]}]

json.NewEncoder(w).Encode(tree)
```

异构枚举合并：

```go
all := append(enum.AsStringItems(cats.All()), enum.AsStringItems(prods.All())...)
tree := enum.BuildTree(all)
```

## 常量

| 常量 | 值 | 用途 |
|------|-----|------|
| `ParentExtKey` | `"parent"` | BuildTree 的 ext key |
| `NameKey` | `"name"` | ToMap 的展示名字段 |
| `ValueKey` | `"value"` | ToMap 的值字段 |
| `DisabledKey` | `"disabled"` | 禁用字段 |
| `ExtKey` | `"ext"` | 扩展元数据字段 |

## 枚举值类型约束

`EnumBase` 支持：

```go
~int | ~int8 | ~int16 | ~int32 | ~int64 |
~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
~string
```

自定义类型只要底层是 `int`/`uint`/`string` 即可（`type Status int` 等）。

## 多级下拉示例

```
水果 ──┬── 苹果
       └── 香蕉
蔬菜 ──── 胡萝卜
```

```go
prods.AddExt("Apple",  enum.ParentExtKey, "FRUIT")
prods.AddExt("Banana", enum.ParentExtKey, "FRUIT")
prods.AddExt("Carrot", enum.ParentExtKey, "VEGETABLE")
```

## 添加新领域枚举

```go
// 1. 定义类型
type MyEnum int
type MyEnums struct {
    enum.Struct[MyEnum]
    Alpha MyEnum `enum:"0,阿尔法"`
    Beta  MyEnum `enum:"1,贝塔"`
}

// 2. 初始化
var E = enum.InitFor[MyEnum, MyEnums]()

// 3. 使用 E
```

## 设计原则

- 零外部依赖 — 仅使用 Go 标准库
- enum tag 格式：`"value,name[,disabled]"` — 第三段 `"disabled"` 标记为仅展示不可选
- 没有 `init()` — 枚举显式构造，无隐式副作用
