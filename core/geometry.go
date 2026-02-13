package core

type GeometryKind int

const (
	GeoRect GeometryKind = iota
)

type Geometry struct {
	Kind   GeometryKind
	Width  int
	Height int
	Rune   rune
}
