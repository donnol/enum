package enum

import (
	"encoding/json"
	"fmt"
	"iter"
)

// EnumBase is the type constraint for enum value types.
// Only int/uint/string-based types are supported.
type EnumBase interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~string
}

// ParentExtKey is the ext key used by BuildTree to establish
// parent-child relationships. Items with ext[ParentExtKey] == "…"
// are treated as children of the item with the matching value.
const ParentExtKey = "parent"

// JSON field names shared between jsonItem struct tags and ToMap output.
const (
	NameKey     = "name"
	ValueKey    = "value"
	DisabledKey = "disabled"
	ExtKey      = "ext"
)

// Item is one member of an enum—a programmatic key, a human-readable name,
// and a value. Fields are private; use the accessor methods to read them.
type Item[T EnumBase] struct {
	key      string
	name     string
	value    T
	disabled bool
	ext      map[string]string
}

// ItemOption is a functional option for ItemFrom.
type ItemOption[T EnumBase] func(*Item[T])

// WithDisabled marks the item as disabled (display-only, not selectable).
func WithDisabled[T EnumBase]() ItemOption[T] {
	return func(i *Item[T]) { i.disabled = true }
}

// WithExt sets extension metadata on the item.
func WithExt[T EnumBase](m map[string]string) ItemOption[T] {
	return func(i *Item[T]) {
		if m == nil {
			return
		}
		if i.ext == nil {
			i.ext = make(map[string]string, len(m))
		}
		for k, v := range m {
			i.ext[k] = v
		}
	}
}

// ItemFrom creates an Item. key is the programmatic identifier (e.g. "Monday");
// name is the human-readable label (e.g. "周一"). Options like WithDisabled
// and WithExt are applied last.
func ItemFrom[T EnumBase](key, name string, value T, opts ...ItemOption[T]) Item[T] {
	item := Item[T]{key: key, name: name, value: value}
	for _, o := range opts {
		o(&item)
	}
	return item
}

func (i Item[T]) Key() string      { return i.key }
func (i Item[T]) Name() string     { return i.name }
func (i Item[T]) Value() T         { return i.value }
func (i Item[T]) IsDisabled() bool { return i.disabled }
func (i Item[T]) Ext() map[string]string {
	if i.ext == nil {
		return nil
	}
	out := make(map[string]string, len(i.ext))
	for k, v := range i.ext {
		out[k] = v
	}
	return out
}

// Enum is an ordered set of named values. It is logically immutable —
// items, names, and values are fixed after construction. The only
// exception is AddExt, which mutates the extension metadata of an
// existing item; all other fields remain stable.
//
// It is safe for concurrent use after construction, provided AddExt
// calls are not concurrent with reads.
type Enum[T EnumBase] struct {
	items     []Item[T]
	byKey     map[string]Item[T]
	byValue   map[T]Item[T]
	keyOffset map[string]int // item key → index in items, for O(1) ext writes
}

// New creates an Enum from the given items. Panics if no items are
// provided, or if any name or value is duplicated.
func New[T EnumBase](items ...Item[T]) *Enum[T] {
	e, err := newEnum(items)
	if err != nil {
		panic(err)
	}
	return e
}

func newEnum[T EnumBase](items []Item[T]) (*Enum[T], error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("enum.New: at least one item is required")
	}
	byKey := make(map[string]Item[T], len(items))
	byValue := make(map[T]Item[T], len(items))
	offsets := make(map[string]int, len(items))
	for i, item := range items {
		if _, ok := byKey[item.key]; ok {
			return nil, fmt.Errorf("enum.New: duplicate key %q", item.key)
		}
		byKey[item.key] = item
		if _, ok := byValue[item.value]; ok {
			return nil, fmt.Errorf("enum.New: duplicate value %v", item.value)
		}
		byValue[item.value] = item
		offsets[item.key] = i
	}
	return &Enum[T]{items: items, byKey: byKey, byValue: byValue, keyOffset: offsets}, nil
}

// ── lookup ───────────────────────────────────────────────────────────

// ByKey returns the item with the given key. The second return value
// is false when the key does not exist.
func (e *Enum[T]) ByKey(key string) (Item[T], bool) {
	item, ok := e.byKey[key]
	return item, ok
}

// MustByKey returns the item with the given key, panicking if it
// doesn't exist.
func (e *Enum[T]) MustByKey(key string) Item[T] {
	item, ok := e.byKey[key]
	if !ok {
		panic(fmt.Sprintf("enum: key %q not found", key))
	}
	return item
}

// ByValue returns the item with the given value. The second return value
// is false when the value does not exist.
func (e *Enum[T]) ByValue(value T) (Item[T], bool) {
	item, ok := e.byValue[value]
	return item, ok
}

// MustByValue returns the item with the given value, panicking if it
// doesn't exist.
func (e *Enum[T]) MustByValue(value T) Item[T] {
	item, ok := e.byValue[value]
	if !ok {
		panic(fmt.Sprintf("enum: value %v not found", value))
	}
	return item
}

// Index returns the item at the given position in definition order.
// The second return value is false when i is out of range.
func (e *Enum[T]) Index(i int) (Item[T], bool) {
	if i < 0 || i >= len(e.items) {
		return Item[T]{}, false
	}
	return e.items[i], true
}

// MustIndex returns the item at position i, panicking if out of range.
func (e *Enum[T]) MustIndex(i int) Item[T] {
	if i < 0 || i >= len(e.items) {
		panic(fmt.Sprintf("enum: index %d out of range [0, %d)", i, len(e.items)))
	}
	return e.items[i]
}

// Contains reports whether the given value is a member of the enum.
func (e *Enum[T]) Contains(value T) bool {
	_, ok := e.byValue[value]
	return ok
}

// ── iteration ────────────────────────────────────────────────────────

// All returns a copy of all items in definition order.
func (e *Enum[T]) All() []Item[T] {
	out := make([]Item[T], len(e.items))
	copy(out, e.items)
	return out
}

// Keys returns all programmatic keys in definition order.
func (e *Enum[T]) Keys() []string {
	out := make([]string, len(e.items))
	for i, it := range e.items {
		out[i] = it.key
	}
	return out
}

// Values returns all values in definition order.
func (e *Enum[T]) Values() []T {
	values := make([]T, len(e.items))
	for i, item := range e.items {
		values[i] = item.value
	}
	return values
}

// Len returns the number of items.
func (e *Enum[T]) Len() int {
	return len(e.items)
}

// Range iterates over all items in definition order, yielding (key, item).
func (e *Enum[T]) Range() iter.Seq2[string, Item[T]] {
	return func(yield func(string, Item[T]) bool) {
		for _, item := range e.items {
			if !yield(item.key, item) {
				return
			}
		}
	}
}

// ── ext ──────────────────────────────────────────────────────────────

// AddExt adds a key-value pair to the named item's extension metadata.
// No-op when the item is not found. The item's ext map is created on
// first use. Updates both the items slice and the lookup maps so
// subsequent ByKey/ByValue calls reflect the change.
// Panics if the item does not exist — a typo'd key would otherwise be
// silently ignored, hiding a bug from the caller.
func (e *Enum[T]) AddExt(itemKey, extKey, extValue string) {
	idx, ok := e.keyOffset[itemKey]
	if !ok {
		panic(fmt.Sprintf("enum.AddExt: item %q not found", itemKey))
	}
	if e.items[idx].ext == nil {
		e.items[idx].ext = map[string]string{extKey: extValue}
	} else {
		e.items[idx].ext[extKey] = extValue
	}
	it := e.items[idx]
	e.byKey[itemKey] = it
	e.byValue[it.value] = it
}

// GetExt returns a copy of the named item's extension metadata, or nil
// when the item is not found.
func (e *Enum[T]) GetExt(itemKey string) map[string]string {
	item, ok := e.byKey[itemKey]
	if !ok {
		return nil
	}
	return item.Ext()
}

// ToMap returns each item keyed by its key. Useful when the frontend
// needs O(1) lookup by key instead of array iteration:
//
//	json.NewEncoder(w).Encode(event.Events.ToMap())
//	// → {UserCreated: {name:"用户创建", value:"user.created"}, ...}
func (e *Enum[T]) ToMap() map[string]map[string]any {
	m := make(map[string]map[string]any, len(e.items))
	for _, it := range e.items {
		entry := map[string]any{
			NameKey:  it.name,
			ValueKey: it.value,
		}
		if it.disabled {
			entry[DisabledKey] = true
		}
		if ext := it.Ext(); len(ext) > 0 {
			entry[ExtKey] = ext
		}
		m[it.key] = entry
	}
	return m
}

// ConvertValues return []R by to. Can be used >= go1.27
// func (e *Enum[T]) ConvertValues[R any](to func(T) R) []R {
// 	r := make([]R, 0, e.Len())
// 	for _, item := range e.Range() {
// 		r = append(r, to(item.Value()))
// 	}
// 	return r
// }

// ── JSON ──────────────────────────────────────────────────────────────

type jsonItem[T EnumBase] struct {
	Key      string            `json:"key"`
	Name     string            `json:"name"`
	Value    T                 `json:"value"`
	Disabled bool              `json:"disabled,omitempty"`
	Ext      map[string]string `json:"ext,omitempty"`
}

// MarshalJSON serializes the enum as an ordered array of {key, name, value, disabled?, ext?}.
func (e *Enum[T]) MarshalJSON() ([]byte, error) {
	items := make([]jsonItem[T], len(e.items))
	for i, it := range e.items {
		items[i] = jsonItem[T]{
			Key:      it.key,
			Name:     it.name,
			Value:    it.value,
			Disabled: it.disabled,
			// Copy via Ext() to read a snapshot instead of the live map.
			Ext: it.Ext(),
		}
	}
	return json.Marshal(items)
}

// UnmarshalJSON reconstructs the enum from a JSON array.
// Returns an error on duplicate names/values or invalid JSON.
func (e *Enum[T]) UnmarshalJSON(data []byte) error {
	var raw []jsonItem[T]
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	items := make([]Item[T], len(raw))
	for i, r := range raw {
		items[i] = Item[T]{
			key:      r.Key,
			name:     r.Name,
			value:    r.Value,
			disabled: r.Disabled,
			ext:      r.Ext,
		}
	}
	rebuilt, err := newEnum(items)
	if err != nil {
		return err
	}
	*e = *rebuilt
	return nil
}
