package enum_test

import (
	"encoding/json"
	"fmt"

	"github.com/donnol/enum"
)

func ExampleStruct_MarshalJSON() {
	p := enum.InitFor[Priority, Priorities]()
	b, _ := json.Marshal(p)
	fmt.Println(string(b))
	// Output:
	// {"Low":0,"Medium":1,"High":2}
}

func ExampleStruct_MarshalJSON_struct() {
	p := enum.InitFor[Priority, Priorities]()
	sb, _ := json.Marshal(p.Struct)
	fmt.Println(string(sb))
	// Output:
	// {}
}

// ── Enum MarshalJSON ──────────────────────────────────────────────────

func ExampleEnum_MarshalJSON_int() {
	statuses := enum.InitFor[OrderStatus, OrderStatuses]()
	b, _ := json.Marshal(statuses.Enum())
	fmt.Println(string(b))
	// Output:
	// [{"key":"Pending","name":"待处理","value":0},{"key":"Processing","name":"处理中","value":1},{"key":"Shipped","name":"已发货","value":2},{"key":"Cancelled","name":"已取消","value":3,"disabled":true}]
}

func ExampleEnum_MarshalJSON_string() {
	e := enum.InitFor[Severity, SeverityEnum]()
	b, _ := json.Marshal(e.Enum())
	fmt.Println(string(b))
	// Output:
	// [{"key":"Info","name":"信息","value":"INFO"},{"key":"Warn","name":"警告","value":"WARN"},{"key":"Error","name":"错误","value":"ERROR"}]
}

// ── Enum UnmarshalJSON ────────────────────────────────────────────────

func ExampleEnum_UnmarshalJSON() {
	data := []byte(`[{"key":"Low","name":"低","value":0},{"key":"High","name":"高","value":2}]`)
	var e enum.Enum[Priority]
	if err := json.Unmarshal(data, &e); err != nil {
		panic(err)
	}
	item, _ := e.ByKey("High")
	fmt.Println("found:", item.Name(), "→", item.Value())
	// Output:
	// found: 高 → 2
}

// ── Struct (embedded) ─────────────────────────────────────────────────
// When Struct is embedded in a struct with exported fields,
// json.Marshal on the outer struct would marshal the raw fields.
// Use .Enum() to get the enum items instead.

func ExampleStruct_embedded_marshal() {
	e := enum.InitFor[Severity, SeverityEnum]()
	b, _ := json.Marshal(e.Enum())
	var items []map[string]any
	json.Unmarshal(b, &items)
	fmt.Println(items[0]["key"], items[0]["name"])
	// Output:
	// Info 信息
}

func ExampleStruct_embedded_unmarshal() {
	// Unmarshal replaces the underlying enum data.
	data := []byte(`[{"key":"A","name":"甲","value":1}]`)
	var e enum.Enum[Priority]
	json.Unmarshal(data, &e)
	fmt.Println(e.Len(), e.Contains(Priority(1)))
	// Output:
	// 1 true
}

// ── full round-trip scenario ─────────────────────────────────────────

func ExampleEnum_roundTrip() {
	// Create, marshal, unmarshal back — end-to-end.
	e := enum.InitFor[OrderStatus, OrderStatuses]()
	e.AddExt("Processing", "color", "blue")

	// Marshal.
	b, _ := json.Marshal(e.Enum())

	// Unmarshal into a fresh Enum.
	var restored enum.Enum[OrderStatus]
	json.Unmarshal(b, &restored)

	// Verify.
	item, _ := restored.ByKey("Processing")
	fmt.Println(item.Name(), item.Ext()["color"])
	// Output:
	// 处理中 blue
}

// ToMap as JSON.
func ExampleEnum_ToMap() {
	e := enum.InitFor[Severity, SeverityEnum]()
	b, _ := json.Marshal(e.ToMap())
	fmt.Println(string(b))
	// Output:
	// {"Error":{"name":"错误","value":"ERROR"},"Info":{"name":"信息","value":"INFO"},"Warn":{"name":"警告","value":"WARN"}}
}

// BuildTree produces nested JSON for multi-level dropdowns.
func ExampleBuildTree() {
	items := []enum.Item[string]{
		enum.ItemFrom("Fruit", "水果", "FRUIT"),
		enum.ItemFrom("Vegetable", "蔬菜", "VEGETABLE"),
		enum.ItemFrom("Apple", "苹果", "APPLE",
			enum.WithExt[string](map[string]string{enum.ParentExtKey: "FRUIT"})),
		enum.ItemFrom("Banana", "香蕉", "BANANA",
			enum.WithExt[string](map[string]string{enum.ParentExtKey: "FRUIT"}),
			enum.WithDisabled[string]()),
		enum.ItemFrom("Carrot", "胡萝卜", "CARROT",
			enum.WithExt[string](map[string]string{enum.ParentExtKey: "VEGETABLE"})),
	}

	tree := enum.BuildTree(items)
	b, _ := json.MarshalIndent(tree, "", "  ")
	fmt.Println(string(b))

	// Output:
	// [
	//   {
	//     "key": "Fruit",
	//     "name": "水果",
	//     "value": "FRUIT",
	//     "children": [
	//       {
	//         "key": "Apple",
	//         "name": "苹果",
	//         "value": "APPLE"
	//       },
	//       {
	//         "key": "Banana",
	//         "name": "香蕉",
	//         "value": "BANANA",
	//         "disabled": true
	//       }
	//     ]
	//   },
	//   {
	//     "key": "Vegetable",
	//     "name": "蔬菜",
	//     "value": "VEGETABLE",
	//     "children": [
	//       {
	//         "key": "Carrot",
	//         "name": "胡萝卜",
	//         "value": "CARROT"
	//       }
	//     ]
	//   }
	// ]
}
