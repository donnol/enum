package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/donnol/enum"
)

// ── AsStringItems ────────────────────────────────────────────────────

func TestAsStringItems_Int(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	items := p.All()
	strs := enum.AsStringItems(items)

	if len(strs) != 3 {
		t.Fatalf("len = %d, want 3", len(strs))
	}
	if strs[0].Key() != "Low" || strs[0].Name() != "低" || strs[0].Value() != "0" {
		t.Errorf("strs[0] = %+v, want {Low 低 0}", strs[0])
	}
	if strs[1].Value() != "1" || strs[2].Value() != "2" {
		t.Errorf("values = %q/%q/%q, want 0/1/2", strs[0].Value(), strs[1].Value(), strs[2].Value())
	}
}

func TestAsStringItems_String(t *testing.T) {
	s := enum.InitFor[Severity, SeverityEnum]()
	items := s.All()
	strs := enum.AsStringItems(items)

	if strs[0].Value() != "INFO" || strs[1].Value() != "WARN" {
		t.Errorf("values = %q/%q, want INFO/WARN", strs[0].Value(), strs[1].Value())
	}
}

func TestAsStringItems_PreservesDisabled(t *testing.T) {
	oss := enum.InitFor[OrderStatus, OrderStatuses]()
	items := oss.All()
	strs := enum.AsStringItems(items)

	cancelled := strs[3]
	if !cancelled.IsDisabled() {
		t.Error("disabled flag not preserved")
	}
}

func TestAsStringItems_PreservesExt(t *testing.T) {
	p := enum.InitFor[Priority, Priorities]()
	p.AddExt("Low", "css", "muted")
	strs := enum.AsStringItems(p.All())

	if strs[0].Ext()["css"] != "muted" {
		t.Errorf("ext = %v, want {css:muted}", strs[0].Ext())
	}
}

func TestAsStringItems_Empty(t *testing.T) {
	strs := enum.AsStringItems([]enum.Item[int]{})
	if len(strs) != 0 {
		t.Errorf("len = %d, want 0", len(strs))
	}
}

func TestAsStringItems_WithDotSlashValues(t *testing.T) {
	e := enum.InitFor[Endpoint, Endpoints]()
	strs := enum.AsStringItems(e.All())

	if strs[2].Value() != "/user/login" {
		t.Errorf("strs[2] = %q, want /user/login", strs[2].Value())
	}
	if strs[3].Value() != "zapp.order.created" {
		t.Errorf("strs[3] = %q, want zapp.order.created", strs[3].Value())
	}
}

// ── BuildTree via AsStringItems ──────────────────────────────────────

func TestBuildTree_FromAsStringItems(t *testing.T) {
	cats := enum.InitFor[Category, Categories]()
	prods := enum.InitFor[Product, Products]()
	prods.AddExt("Apple", enum.ParentExtKey, "FRUIT")
	prods.AddExt("Banana", enum.ParentExtKey, "FRUIT")
	prods.AddExt("Carrot", enum.ParentExtKey, "VEGETABLE")

	all := append(enum.AsStringItems(cats.All()), enum.AsStringItems(prods.All())...)
	tree := enum.BuildTree(all)

	if len(tree) != 2 {
		t.Fatalf("roots = %d, want 2", len(tree))
	}

	fruit := tree[0]
	if fruit.Key != "Fruit" || fruit.Name != "水果" {
		t.Errorf("Fruit = %+v", fruit)
	}
	if len(fruit.Children) != 2 {
		t.Fatalf("Fruit children = %d, want 2", len(fruit.Children))
	}
	if fruit.Children[0].Key != "Apple" || fruit.Children[1].Key != "Banana" {
		t.Errorf("Fruit children = %v", fruit.Children)
	}

	veg := tree[1]
	if len(veg.Children) != 1 || veg.Children[0].Key != "Carrot" {
		t.Errorf("Vegetable children = %v", veg.Children)
	}
}

func TestBuildTree_FromAsStringItems_DualRootOnly(t *testing.T) {
	// Only roots → flat tree.
	cats := enum.InitFor[Category, Categories]()
	tree := enum.BuildTree(enum.AsStringItems(cats.All()))

	if len(tree) != 2 {
		t.Fatalf("roots = %d, want 2", len(tree))
	}
	for _, n := range tree {
		if n.Children != nil {
			t.Errorf("%q should be a leaf", n.Name)
		}
	}
}

func TestBuildTree_FromAsStringItems_Disabled(t *testing.T) {
	s := enum.InitFor[OrderStatus, OrderStatuses]()
	tree := enum.BuildTree(enum.AsStringItems(s.All()))

	if len(tree) != 4 {
		t.Fatalf("len = %d, want 4", len(tree))
	}
	if !tree[3].Disabled || tree[3].Key != "Cancelled" {
		t.Errorf("Cancelled disabled = %v, %+v", tree[3].Disabled, tree[3])
	}
	if tree[0].Disabled {
		t.Error("Pending should not be disabled")
	}
}

func TestBuildTree_FromAsStringItems_JSON(t *testing.T) {
	items := []enum.Item[string]{
		enum.ItemFrom("Root", "根", "R"),
		enum.ItemFrom("Child", "子", "C",
			enum.WithExt[string](map[string]string{enum.ParentExtKey: "R"})),
	}
	tree := enum.BuildTree(items)

	b, _ := json.Marshal(tree)
	var parsed []map[string]any
	json.Unmarshal(b, &parsed)

	if parsed[0]["value"] != "R" {
		t.Errorf("root value = %v, want R", parsed[0]["value"])
	}
	if children := parsed[0]["children"]; children == nil {
		t.Error("root should have children")
	}
}

// ── AsStringItems round-trip: convert and build tree ────────────────

func TestAsStringItems_RoundTrip(t *testing.T) {
	// Create typed enum → AsStringItems → BuildTree → verify structure.
	type Level int
	type Levels struct {
		enum.Struct[Level]
		One   Level `enum:"1,一"`
		Two   Level `enum:"2,二"`
		Three Level `enum:"3,三,disabled"`
	}
	lvls := enum.InitFor[Level, Levels]()
	lvls.AddExt("Two", enum.ParentExtKey, "1")
	lvls.AddExt("Three", enum.ParentExtKey, "2")

	strs := enum.AsStringItems(lvls.All())
	tree := enum.BuildTree(strs)

	if len(tree) != 1 || tree[0].Key != "One" {
		t.Fatalf("root = %+v, want One", tree[0])
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Key != "Two" {
		t.Fatalf("One children = %+v", tree[0].Children)
	}
	three := tree[0].Children[0].Children
	if len(three) != 1 || !three[0].Disabled || three[0].Key != "Three" {
		t.Errorf("Three = %+v", three)
	}
}
