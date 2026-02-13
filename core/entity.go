package core

type Entity uint64

type World struct {
	next       Entity
	transforms map[Entity]*Transform
	geometries map[Entity]*Geometry
	children   map[Entity][]Entity
	roots      []Entity
}

func NewWorld() *World {
	return &World{
		transforms: make(map[Entity]*Transform),
		geometries: make(map[Entity]*Geometry),
		children:   make(map[Entity][]Entity),
	}
}

func (w *World) Spawn() Entity {
	w.next++
	return w.next
}

func (w *World) AddRoot(e Entity) {
	w.roots = append(w.roots, e)
}

func (w *World) Attach(child, parent Entity) {
	w.children[parent] = append(w.children[parent], child)
}

func (w *World) AddTransform(e Entity, t *Transform) {
	w.transforms[e] = t
}

func (w *World) AddGeometry(e Entity, g *Geometry) {
	w.geometries[e] = g
}

func (w *World) Roots() []Entity {
	return w.roots
}

func (w *World) Children(e Entity) []Entity {
	return w.children[e]
}

func (w *World) Transform(e Entity) *Transform {
	return w.transforms[e]
}

func (w *World) Geometry(e Entity) *Geometry {
	return w.geometries[e]
}
