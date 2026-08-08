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

有序、逻辑不可变（只有 ext 可写）。并发读安全；`AddExt` 修改 ext 元数据，**不得与读操作并发**（否则可能 `concurrent map writes`）。`Struct[T].AddExt` 同样受此约束。

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

> **⚠️ JSON 序列化注意**：外层 struct 只要有自己的字段（不止嵌入 `Struct[T]`），Go 的
> `encoding/json` 就会**忽略内嵌的 `MarshalJSON`/`UnmarshalJSON`**，序列化结果会是
> `{"enum":null,"UserCreated":""}` 这种。此时必须显式用 `.Enum()`：
> `json.NewEncoder(w).Encode(events.Enum())`。
>
> **⚠️ `UnmarshalJSON` 只重建内部枚举**：它更新 `Enum` 的数据，但**不会**更新外层
> struct 的常量字段（`events.UserCreated` 等仍保持旧值）。反序列化后如需字段同步，请重新
> `InitFor`。

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
e := enum.InitFor[MyEnum, MyEnums]()

// 3. 使用
switch v {
case e.Alpha:
case e.Beta:
}
```

## 配合 lint 工具（只读保护）

`enum.InitFor` 返回的结构体字段是**导出的**，运行期可以被改写（如 `Events.UserCreated = "x"`），
破坏枚举的只读语义。配套 lint 工具（`lint`，包 `lint/`，CLI `cmd/enumlint/`）
用静态分析扫描整个模块，找出所有对枚举变量或其字段的**写入**，把这类 bug 挡在提交前。

### 运行

```bash
# 在模块根目录（有 go.mod 的地方）
go run ./cmd/enumlint/ ./...
# 或安装为全局命令
go install github.com/donnol/enum/cmd/enumlint@latest && enumlint ./...
```

**指定 enum 包路径**：工具会自动发现声明 `InitFor` 的包；若 enum 是外部依赖且自动发现不到，
用 `-enum-pkg` 显式指定：

```bash
go run ./cmd/enumlint/ -enum-pkg github.com/donnol/enum ./...
```

无违规：exit 0，输出 `✅ enum check is good 🌟`。

发现违规：exit 1，输出违规表格（`路径:行号` 可直接点击跳转）：

```
🚨 enum check is bad 💥
Location                          Kind   Target
--------                          ----   ------
server/biz/order/foo.go:6         field  event.Events.UserCreated
server/biz/order/foo.go:20        variable  Events
⚠️  请修正后重试！
```

### 检测规则

- 只追踪**包级** `var X = enum.InitFor[T, struct{...}]()` 声明的枚举（须限定为 enum 包的 `InitFor`，同名其他函数不误报）
- 检测以下写入：
  - `X = ...` — 整体重写枚举变量
  - `X.Field = ...` / `X.Field += ...` — 字段赋值、复合赋值
  - `X.Field++` / `X.Field--` — 自增自减
  - 跨包写入 `pkg.X.Field = ...` 同样检测
- `X := ...`（短声明）视为局部声明，不误报；**函数内局部 shadow 只在自身函数内生效**，不影响其他函数对包级枚举的检测
- 支持自定义 import 别名（`import myenum "…/enum"`）

### 已知限制

- 只检测包级枚举；函数内局部 `enum.InitFor` 不追踪 -- 函数内作用范围小，可自行检查
- 不校验枚举**定义**的正确性（重复 value、非法 tag、缺嵌入 `Struct[T]` 等仍要到运行时 panic）
- 通过指针间接修改（`f(&Events.Field)`）无法静态检测
- `_test.go` 同样受约束（测试代码也不应改写枚举）

## 设计原则

- 零外部依赖 — 仅使用 Go 标准库
- enum tag 格式：`"value,name[,disabled]"` — 第三段 `"disabled"` 标记为仅展示不可选
- 没有 `init()` — 枚举显式构造，无隐式副作用

## ts

```go
type Priority int

type Priorities struct {
    enum.Struct[Priority]
    Low    Priority `enum:"0,低"`
    Medium Priority `enum:"1,中"`
    High   Priority `enum:"2,高,disabled"`
}
```

用 enum.InitFor 后，Enum.MarshalJSON 的数组元素是：

```json
{"key":"Low","name":"低","value":0}
{"key":"Medium","name":"中","value":1}
{"key":"High","name":"高","disabled":true,"value":2}
```

1. 对应的 TS as const 枚举数据

```js
export const PRIORITIES = {
  Low: {
    key: 'Low' as const,
    name: '低' as const,
    value: 0 as const,
  },
  Medium: {
    key: 'Medium' as const,
    name: '中' as const,
    value: 1 as const,
  },
  High: {
    key: 'High' as const,
    name: '高' as const,
    value: 2 as const,
    disabled: true as const,
  },
} as const;
```

这就是「ts {...} as const」部分：每一项有 key / name / value / disabled?，和 Go 侧 JSON 结构一一对应。

2. 从 as const 中取出 key / value 相关类型

```js
// 所有枚举项的联合类型
export type PriorityItem = (typeof PRIORITIES)[keyof typeof PRIORITIES];

// key 的字面量联合："Low" | "Medium" | "High"
export type PriorityKey = PriorityItem['key'];

// value 的字面量联合：0 | 1 | 2
export type PriorityValue = PriorityItem['value'];
```

如果你只想要一个「key 到 value」的简单映射结构，可以再包一层：

```js
// key -> value 的映射对象类型
export type PriorityKeyValueMap = {
  [K in PriorityKey]: Extract<PriorityItem, { key: K }>['value'];
};

// 实例：由 PRIORITIES 推导出的 key/value map
export const PRIORITY_KEY_VALUE: PriorityKeyValueMap = {
  Low: PRIORITIES.Low.value,
  Medium: PRIORITIES.Medium.value,
  High: PRIORITIES.High.value,
};
```

这样你就同时有：

- PRIORITIES：完整的枚举元数据（key/name/value/disabled）——对应 Go 侧 Enum.MarshalJSON 的数组元素结构；
- PriorityKey / PriorityValue：类型级别的 key/value 联合；
- PRIORITY_KEY_VALUE：在 TS 里方便用的「key → value」映射。
