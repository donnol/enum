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
	enum *Enum[T]
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
// whose first field is Struct[T]) and populates both the struct field
// values and the underlying enum.
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
		v := rv.Field(i).Addr().Interface()
		var ok bool
		s, ok = v.(*Struct[T])
		if ok {
			break
		}
	}
	if s == nil {
		panic(fmt.Sprintf("enum.Init: %T has no embedded Struct%T field", impl, s))
	}

	items := make([]Item[T], 0, rt.NumField())
	for i := range rt.NumField() {
		f := rt.Field(i)
		// Skip embedded fields and unexported fields.
		if f.Anonymous || f.PkgPath != "" {
			continue
		}
		tagValue, name, disabled := parseTag(f, rt, f.Name)
		val := tagToValue[T](tagValue)
		var opts []ItemOption[T]
		if disabled {
			opts = append(opts, WithDisabled[T]())
		}
		items = append(items, ItemFrom(f.Name, name, val, opts...))

		// Set the struct field value so callers can use it as a constant.
		rv.Field(i).Set(reflect.ValueOf(val).Convert(f.Type))
	}
	s.enum = New(items...)
}

// ── delegation to Enum ───────────────────────────────────────────────

// Enum returns the underlying Enum[T]. When Struct is embedded in
// another struct that has its own exported fields, Go's JSON encoder
// ignores the embedded MarshalJSON/UnmarshalJSON — use .Enum() to
// serialize/deserialize directly:
//
//	json.NewEncoder(w).Encode(event.Events.Enum())       // marshal
//	json.NewDecoder(r).Decode(event.Events.Enum())       // unmarshal
func (s *Struct[T]) Enum() *Enum[T] { return s.enum }

// MarshalJSON / UnmarshalJSON delegate to the underlying enum.
// These are only effective when marshaling a bare Struct[T] (not embedded
// in a struct with exported fields). For the embedded case, use .Enum().
func (s *Struct[T]) MarshalJSON() ([]byte, error)    { return s.enum.MarshalJSON() }
func (s *Struct[T]) UnmarshalJSON(data []byte) error { return s.enum.UnmarshalJSON(data) }

func (s *Struct[T]) ByKey(name string) (Item[T], bool) { return s.enum.ByKey(name) }
func (s *Struct[T]) MustByKey(name string) Item[T]     { return s.enum.MustByKey(name) }
func (s *Struct[T]) ByValue(value T) (Item[T], bool)   { return s.enum.ByValue(value) }
func (s *Struct[T]) MustByValue(value T) Item[T]       { return s.enum.MustByValue(value) }
func (s *Struct[T]) Index(i int) (Item[T], bool)       { return s.enum.Index(i) }
func (s *Struct[T]) MustIndex(i int) Item[T]           { return s.enum.MustIndex(i) }
func (s *Struct[T]) Contains(value T) bool             { return s.enum.Contains(value) }
func (s *Struct[T]) All() []Item[T]                    { return s.enum.All() }
func (s *Struct[T]) Keys() []string                    { return s.enum.Keys() }
func (s *Struct[T]) Values() []T                       { return s.enum.Values() }
func (s *Struct[T]) Len() int                          { return s.enum.Len() }
func (s *Struct[T]) Range() iter.Seq2[string, Item[T]] { return s.enum.Range() }
func (s *Struct[T]) AddExt(itemKey, extKey, extValue string) {
	s.enum.AddExt(itemKey, extKey, extValue)
}
func (s *Struct[T]) GetExt(itemKey string) map[string]string { return s.enum.GetExt(itemKey) }
func (s *Struct[T]) ToMap() map[string]map[string]any        { return s.enum.ToMap() }

// ConvertValues return []R by to. Can be used >= go1.27
// func (s *Struct[T]) ConvertValues[R any](to func(T) R) []R {
// 	r := make([]R, 0, s.Len())
// 	for _, item := range s.Range() {
// 		r = append(r, to(item.Value()))
// 	}
// 	return r
// }

// ── helpers ──────────────────────────────────────────────────────────

// parseTag splits "<value>,<name>[,disabled]" — e.g. "0,待处理" or "0,待处理,disabled".
// When value is empty (",name"), it derives the value from the field name uppercased.
func parseTag(f reflect.StructField, rt reflect.Type, fieldName string) (value, name string, disabled bool) {
	tag := f.Tag.Get("enum")
	if tag == "" {
		panic(fmt.Sprintf("enum.Init: %s.%s missing enum tag", rt.Name(), fieldName))
	}
	parts := strings.SplitN(tag, ",", 3)
	if len(parts) < 2 || parts[1] == "" {
		panic(fmt.Sprintf("enum.Init: %s.%s invalid enum tag %q", rt.Name(), fieldName, tag))
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
// Supports integer and string underlying types. Custom types based on
// int or string are handled via Convert.
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
