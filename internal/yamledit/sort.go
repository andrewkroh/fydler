package yamledit

import (
	"cmp"
	"slices"

	"github.com/goccy/go-yaml/ast"
)

// FieldAttributeOrder is an ast.Visitor that sorts maps keys for field definitions.
var FieldAttributeOrder = mapKeyOrder{
	"name":        -5,
	"type":        -4,
	"external":    -3,
	"description": -2,
}

type mapKeyOrder map[string]int

func (o mapKeyOrder) Compare(a, b string) int {
	ai, hasA := o[a]
	bi, hasB := o[b]

	if hasA && hasB {
		return ai - bi
	} else if hasA {
		return ai
	} else if hasB {
		return -1 * bi
	}
	return cmp.Compare(a, b)
}

func (o mapKeyOrder) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.MappingNode:
		slices.SortFunc(n.Values, func(a, b *ast.MappingValueNode) int {
			return o.Compare(a.Key.String(), b.Key.String())
		})
	}
	return o
}
