package enum

import "fmt"

// TreeNode is a recursive frontend node for multi-level dropdowns.
type TreeNode struct {
	Key      string     `json:"key"`
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Disabled bool       `json:"disabled,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// CascaderOption is the Ant Design <Cascader> / <TreeSelect> format,
// using label instead of name. Produced by Struct[T].TreeOptions().
type CascaderOption struct {
	Label    string            `json:"label"`
	Value    string            `json:"value"`
	Disabled bool              `json:"disabled,omitempty"`
	Children []CascaderOption  `json:"children,omitempty"`
}

// BuildTree converts a flat item list into a recursive tree keyed by
// ext["parent"]. Items without a parent (or whose parent doesn't match
// any item) become roots.
func BuildTree[T EnumBase](items []Item[T]) []TreeNode {
	byValue := make(map[string]Item[T], len(items))
	for _, it := range items {
		byValue[fmt.Sprint(it.Value())] = it
	}

	children := make(map[string][]Item[T])
	orphans := make([]Item[T], 0)
	for _, it := range items {
		parent, hasParent := it.Ext()[ParentExtKey]
		if !hasParent || parent == "" {
			orphans = append(orphans, it)
			continue
		}
		if _, exists := byValue[parent]; !exists {
			orphans = append(orphans, it)
			continue
		}
		children[parent] = append(children[parent], it)
	}

	nodes := make([]TreeNode, 0, len(orphans))
	for _, it := range orphans {
		nodes = append(nodes, buildNode(it, children))
	}
	return nodes
}

func buildNode[T EnumBase](it Item[T], children map[string][]Item[T]) TreeNode {
	node := TreeNode{
		Key:      it.Key(),
		Name:     it.Name(),
		Value:    fmt.Sprint(it.Value()),
		Disabled: it.IsDisabled(),
	}
	childList := children[fmt.Sprint(it.Value())]
	node.Children = make([]TreeNode, len(childList))
	for j, child := range childList {
		node.Children[j] = buildNode(child, children)
	}
	if len(node.Children) == 0 {
		node.Children = nil
	}
	return node
}

// AsStringItems converts a typed Item slice to []Item[string] using
// fmt.Sprint for the value. This allows combining items from different
// enum types into a single BuildTree input.
func AsStringItems[T EnumBase](items []Item[T]) []Item[string] {
	out := make([]Item[string], len(items))
	for i, it := range items {
		var opts []ItemOption[string]
		if it.IsDisabled() {
			opts = append(opts, WithDisabled[string]())
		}
		if ext := it.Ext(); len(ext) > 0 {
			opts = append(opts, WithExt[string](ext))
		}
		out[i] = ItemFrom(it.Key(), it.Name(), fmt.Sprint(it.Value()), opts...)
	}
	return out
}
