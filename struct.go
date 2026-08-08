package enum

import (
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
)

// Struct is a reflection-driven enum backed by Enum[T]. Use InitFor:
//
//	type Statuses struct {
//	    Struct[Status]  // embed anywhere (found by scanning anonymous fields)
//	    Pending Status  `enum:"0,待处理"`
//	    Active  Status  `enum:"1,活跃"`
//	}
//
//	s := enum.InitFor[Status, Statuses]()
//	s.ByKey("Active")     // → Item, true
//	switch v { case s.Pending: ... }
type Struct[T EnumBase] struct {
	enum      *Enum[T]
	buildTree func() []TreeNode
}

// InitFor is a one-liner: allocates, inits, and returns a value of R.
// It delegates to Init so struct validation stays in one place.
//
//	p := enum.InitFor[Priority, Priorities]()
func InitFor[T EnumBase, R any]() R {
	var zero R
	initStruct[T](&zero)
	return zero
}

// initStruct walks the exported fields of impl (a pointer to a struct
// containing Struct[T]) and populates both the struct field values and
// the underlying enum. Struct-typed fields (not assignable to T) that
// contain a nested Struct[T] are recursively processed — their items
// are merged into the parent enum with ext["parent"] references,
// enabling BuildTree to produce a hierarchy.
func initStruct[T EnumBase](impl any) {

	rv := reflect.ValueOf(impl).Elem()
	rt := rv.Type()

	// Find the embedded Struct[T] field.
	var s *Struct[T]
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.Anonymous {
			continue
		}
		fieldVal := rv.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}
		var v any
		if f.Type.Kind() == reflect.Pointer {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(f.Type.Elem()))
			}
			v = fieldVal.Interface()
		} else {
			v = fieldVal.Addr().Interface()
		}
		var ok bool
		s, ok = v.(*Struct[T])
		if ok {
			break
		}
	}
	if s == nil {
		panic(fmt.Sprintf("enum.Init: %T has no embedded enum.Struct[T] field", impl))
	}

	// Collect all items (including nested) and build the enum.
	items := processStruct[T](rv)
	s.enum = New(items...)

	// Tree builder — uses ext["parent"] references to reconstruct hierarchy.
	s.buildTree = func() []TreeNode {
		return BuildTree(s.enum.All())
	}
}

// processStruct extracts enum-tagged fields and recursively merges
// items from nested struct fields, maintaining field declaration order.
//
// Naming convention: a struct whose first exported field is "Self" is a
// hierarchical group. Sibling items (immediate or nested within a flat
// sub-struct) get ext["parent"] = Self.Value() if they have no parent
// from a deeper level.
func processStruct[T EnumBase](rv reflect.Value) []Item[T] {
	const (
		SelfFieldName = "Self"
	)

	rt := rv.Type()
	var items []Item[T]
	hasSelf := firstExportedFieldName(rt) == SelfFieldName

	outerSelfValue := func() string {
		if len(items) > 0 {
			return fmt.Sprint(items[0].Value())
		}
		return ""
	}

	for i := range rt.NumField() {
		f := rt.Field(i)
		if f.Anonymous || f.PkgPath != "" {
			continue
		}
		fieldVal := rv.Field(i)

		if f.Type.Kind() == reflect.Struct {
			nested := findStructIn[T](fieldVal)
			if nested == nil {
				continue
			}
			initStruct[T](fieldVal.Addr().Interface())
			if nested.enum == nil {
				continue
			}
			nestedItems := nested.enum.All()
			if len(nestedItems) == 0 {
				continue
			}
			innerHasSelf := firstExportedFieldName(f.Type) == SelfFieldName

			for _, it := range nestedItems {
				key := f.Name + "." + it.Key()
				var opts []ItemOption[T]
				_, hasParent := it.Ext()[ParentExtKey]

				if !hasParent && innerHasSelf {
					// Flat siblings within a hierarchical group —
					// but the first item (Self) already has no parent
					// at this level, and j=0 items are the group root.
					// Non-first items get parent = first_item.value at
					// the inner level.
					if it.Key() != nestedItems[0].Key() {
						opts = append(opts, WithExt[T](map[string]string{
							ParentExtKey: fmt.Sprint(nestedItems[0].Value()),
						}))
					}
				} else if !hasParent && !innerHasSelf && outerSelfValue() != "" {
					// Flat container: items get parent from the
					// outer struct's first item.
					opts = append(opts, WithExt[T](map[string]string{
						ParentExtKey: outerSelfValue(),
					}))
				} else if hasParent {
					// Preserve ext from deeper level.
					opts = append(opts, WithExt[T](it.Ext()))
				}
				if it.IsDisabled() {
					opts = append(opts, WithDisabled[T]())
				}
				items = append(items, ItemFrom(key, it.Name(), it.Value(), opts...))
			}
			continue
		}

		// Regular enum-tagged field.
		tagValue, name, disabled := parseTag(f, rt, f.Name)
		val := tagToValue[T](tagValue)
		var opts []ItemOption[T]
		if hasSelf && len(items) > 0 {
			opts = append(opts, WithExt[T](map[string]string{
				ParentExtKey: outerSelfValue(),
			}))
		}
		if disabled {
			opts = append(opts, WithDisabled[T]())
		}
		items = append(items, ItemFrom(f.Name, name, val, opts...))

		// Set the struct field value so callers can use it as a constant.
		fieldVal.Set(reflect.ValueOf(val).Convert(f.Type))
	}
	return items
}

// firstExportedFieldName returns the name of the first non-anonymous,
// exported field in the given reflect.Type, or "" if none exists.
func firstExportedFieldName(rt reflect.Type) string {
	for f := range rt.Fields() {
		if f.Anonymous || f.PkgPath != "" {
			continue
		}
		return f.Name
	}
	return ""
}

// findStructIn walks the exported fields of a struct value looking for
// an embedded Struct[T]. Returns nil if not found.
func findStructIn[T EnumBase](rv reflect.Value) *Struct[T] {
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.Anonymous {
			continue
		}
		fieldVal := rv.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}
		if f.Type.Kind() == reflect.Pointer {
			if fieldVal.IsNil() {
				continue
			}
			if s, ok := fieldVal.Interface().(*Struct[T]); ok {
				return s
			}
		} else {
			if s, ok := fieldVal.Addr().Interface().(*Struct[T]); ok {
				return s
			}
		}
	}
	return nil
}

// ── delegation to Enum ───────────────────────────────────────────────

// Enum returns the underlying Enum[T]. When Struct is embedded in
// another struct that has its own exported fields, Go's JSON encoder
// ignores the embedded MarshalJSON/UnmarshalJSON — use .Enum() to
// serialize/deserialize directly:
//
//	json.NewEncoder(w).Encode(event.Events.Enum())       // marshal
//	json.NewDecoder(r).Decode(event.Events.Enum())       // unmarshal
//
// Returns nil when the Struct has not been initialized via InitFor.
func (s *Struct[T]) Enum() *Enum[T] { return s.enum }

// uninitialized reports whether InitFor has not been called on this Struct.
func (s *Struct[T]) uninitialized() bool { return s.enum == nil }

// MarshalJSON / UnmarshalJSON delegate to the underlying enum. They
// return an error (rather than panic) when the Struct is uninitialized.
func (s *Struct[T]) MarshalJSON() ([]byte, error) {
	if s.uninitialized() {
		return nil, fmt.Errorf("enum.Struct: not initialized; call InitFor before use")
	}
	return s.enum.MarshalJSON()
}
func (s *Struct[T]) UnmarshalJSON(data []byte) error {
	if s.uninitialized() {
		return fmt.Errorf("enum.Struct: not initialized; call InitFor before use")
	}
	return s.enum.UnmarshalJSON(data)
}

func (s *Struct[T]) ByKey(name string) (Item[T], bool) {
	if s.uninitialized() {
		return Item[T]{}, false
	}
	return s.enum.ByKey(name)
}
func (s *Struct[T]) MustByKey(name string) Item[T] {
	if s.uninitialized() {
		panic("enum.Struct: not initialized; call InitFor before use")
	}
	return s.enum.MustByKey(name)
}
func (s *Struct[T]) ByValue(value T) (Item[T], bool) {
	if s.uninitialized() {
		return Item[T]{}, false
	}
	return s.enum.ByValue(value)
}
func (s *Struct[T]) MustByValue(value T) Item[T] {
	if s.uninitialized() {
		panic("enum.Struct: not initialized; call InitFor before use")
	}
	return s.enum.MustByValue(value)
}
func (s *Struct[T]) Index(i int) (Item[T], bool) {
	if s.uninitialized() {
		return Item[T]{}, false
	}
	return s.enum.Index(i)
}
func (s *Struct[T]) MustIndex(i int) Item[T] {
	if s.uninitialized() {
		panic("enum.Struct: not initialized; call InitFor before use")
	}
	return s.enum.MustIndex(i)
}
func (s *Struct[T]) Contains(value T) bool {
	if s.uninitialized() {
		return false
	}
	return s.enum.Contains(value)
}
func (s *Struct[T]) All() []Item[T] {
	if s.uninitialized() {
		return nil
	}
	return s.enum.All()
}
func (s *Struct[T]) Keys() []string {
	if s.uninitialized() {
		return nil
	}
	return s.enum.Keys()
}
func (s *Struct[T]) Values() []T {
	if s.uninitialized() {
		return nil
	}
	return s.enum.Values()
}
func (s *Struct[T]) Len() int {
	if s.uninitialized() {
		return 0
	}
	return s.enum.Len()
}
func (s *Struct[T]) Range() iter.Seq2[string, Item[T]] {
	if s.uninitialized() {
		return func(func(string, Item[T]) bool) {}
	}
	return s.enum.Range()
}
func (s *Struct[T]) AddExt(itemKey, extKey, extValue string) {
	if s.uninitialized() {
		panic("enum.Struct: not initialized; call InitFor before use")
	}
	s.enum.AddExt(itemKey, extKey, extValue)
}
func (s *Struct[T]) GetExt(itemKey string) map[string]string {
	if s.uninitialized() {
		return nil
	}
	return s.enum.GetExt(itemKey)
}
func (s *Struct[T]) ToMap() map[string]map[string]any {
	if s.uninitialized() {
		return nil
	}
	return s.enum.ToMap()
}

// Tree returns the enum items as a recursive TreeNode tree.
func (s *Struct[T]) Tree() []TreeNode {
	if s.buildTree == nil {
		return nil
	}
	return s.buildTree()
}

// TreeOptions converts the tree to Ant Design Cascader format.
func (s *Struct[T]) TreeOptions() []CascaderOption {
	tree := s.Tree()
	if tree == nil {
		return nil
	}
	return treeToCascader(tree)
}

func treeToCascader(nodes []TreeNode) []CascaderOption {
	result := make([]CascaderOption, len(nodes))
	for i, n := range nodes {
		opt := CascaderOption{
			Label:    n.Name,
			Value:    n.Value,
			Disabled: n.Disabled,
		}
		if len(n.Children) > 0 {
			opt.Children = treeToCascader(n.Children)
		}
		result[i] = opt
	}
	return result
}

// ── helpers ──────────────────────────────────────────────────────────

// typeLabel returns a readable name for a struct type. Anonymous structs
// have an empty Name(), so fall back to the full type string (e.g.
// "struct { ... }") to keep panic messages debuggable.
func typeLabel(rt reflect.Type) string {
	if rt.Name() != "" {
		return rt.Name()
	}
	return rt.String()
}

// parseTag splits "<value>,<name>[,disabled]" — e.g. "0,待处理" or "0,待处理,disabled".
func parseTag(f reflect.StructField, rt reflect.Type, fieldName string) (value, name string, disabled bool) {
	tag := f.Tag.Get("enum")
	if tag == "" {
		panic(fmt.Sprintf("enum.Init: %s.%s missing enum tag", typeLabel(rt), fieldName))
	}
	parts := strings.SplitN(tag, ",", 3)
	if len(parts) < 2 || parts[1] == "" {
		panic(fmt.Sprintf("enum.Init: %s.%s invalid enum tag %q", typeLabel(rt), fieldName, tag))
	}
	value, name = parts[0], parts[1]
	if value == "" {
		value = strings.ToUpper(fieldName)
	}
	if len(parts) == 3 && parts[2] == "disabled" {
		disabled = true
	}
	return
}

// tagToValue converts the tag's value string to a value of type T.
func tagToValue[T EnumBase](s string) T {
	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("enum: cannot parse %q as int: %v", s, err))
		}
		rv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("enum: cannot parse %q as uint: %v", s, err))
		}
		rv.SetUint(n)
	case reflect.String:
		rv.SetString(s)
	default:
		panic(fmt.Sprintf("enum: unsupported enum type %T (must be int or string based)", zero))
	}
	return rv.Interface().(T)
}
