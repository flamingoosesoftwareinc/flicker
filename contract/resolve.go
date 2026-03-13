package contract

import (
	"go/types"
	"reflect"
	"strings"
)

// resolveTypeShape converts a types.Type into a TypeShape, recursively
// resolving struct fields, slices, maps, pointers, and named types.
func resolveTypeShape(t types.Type) TypeShape {
	if t == nil {
		return TypeShape{Name: "<unknown>", Kind: "unknown"}
	}

	return resolveTypeShapeImpl(t, make(map[types.Type]bool))
}

func resolveTypeShapeImpl(t types.Type, seen map[types.Type]bool) TypeShape {
	// Prevent infinite recursion on recursive types.
	if seen[t] {
		return TypeShape{Name: t.String(), Kind: "recursive"}
	}

	switch typ := t.(type) {
	case *types.Named:
		seen[t] = true
		defer delete(seen, t)

		ts := TypeShape{
			Name: typ.Obj().Name(),
			Kind: kindFromUnderlying(typ.Underlying()),
		}
		if pkg := typ.Obj().Pkg(); pkg != nil {
			ts.Pkg = pkg.Path()
		}

		// For structs, resolve fields.
		if st, ok := typ.Underlying().(*types.Struct); ok {
			ts.Fields = resolveStructFields(st, seen)
		}

		// For slices/maps/pointers, resolve inner types.
		switch u := typ.Underlying().(type) {
		case *types.Slice:
			elem := resolveTypeShapeImpl(u.Elem(), seen)
			ts.Elem = &elem
		case *types.Map:
			key := resolveTypeShapeImpl(u.Key(), seen)
			val := resolveTypeShapeImpl(u.Elem(), seen)
			ts.Key = &key
			ts.Elem = &val
		case *types.Pointer:
			elem := resolveTypeShapeImpl(u.Elem(), seen)
			ts.Elem = &elem
		}

		return ts

	case *types.Struct:
		ts := TypeShape{
			Name:   "",
			Kind:   "struct",
			Fields: resolveStructFields(typ, seen),
		}
		return ts

	case *types.Pointer:
		elem := resolveTypeShapeImpl(typ.Elem(), seen)
		return TypeShape{
			Name: "*" + elem.Name,
			Kind: "pointer",
			Elem: &elem,
		}

	case *types.Slice:
		elem := resolveTypeShapeImpl(typ.Elem(), seen)
		return TypeShape{
			Name: "[]" + elem.Name,
			Kind: "slice",
			Elem: &elem,
		}

	case *types.Map:
		key := resolveTypeShapeImpl(typ.Key(), seen)
		val := resolveTypeShapeImpl(typ.Elem(), seen)
		return TypeShape{
			Name: "map[" + key.Name + "]" + val.Name,
			Kind: "map",
			Key:  &key,
			Elem: &val,
		}

	case *types.Basic:
		return TypeShape{
			Name: typ.Name(),
			Kind: typ.Name(),
		}

	case *types.Interface:
		if typ.Empty() {
			return TypeShape{Name: "any", Kind: "interface"}
		}
		return TypeShape{Name: t.String(), Kind: "interface"}

	default:
		return TypeShape{Name: t.String(), Kind: "unknown"}
	}
}

func kindFromUnderlying(t types.Type) string {
	switch t := t.(type) {
	case *types.Struct:
		return "struct"
	case *types.Basic:
		return t.Name()
	case *types.Slice:
		return "slice"
	case *types.Map:
		return "map"
	case *types.Pointer:
		return "pointer"
	case *types.Interface:
		return "interface"
	default:
		return "unknown"
	}
}

func resolveStructFields(st *types.Struct, seen map[types.Type]bool) []FieldShape {
	var fields []FieldShape

	for i := range st.NumFields() {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}

		fs := FieldShape{
			Name: f.Name(),
			Type: resolveTypeShapeImpl(f.Type(), seen),
		}

		// Extract JSON tag.
		tag := st.Tag(i)
		if tag != "" {
			jsonTag := reflect.StructTag(tag).Get("json")
			if jsonTag != "" && jsonTag != "-" {
				// Take just the name part before any comma.
				if idx := strings.Index(jsonTag, ","); idx != -1 {
					jsonTag = jsonTag[:idx]
				}
				fs.JSONTag = jsonTag
			}
		}

		fields = append(fields, fs)
	}

	return fields
}
