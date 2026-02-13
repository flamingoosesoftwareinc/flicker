package core

type Entity uint64

// Material is a per-entity fragment shader applied after drawing.
// x, y are local coords relative to the drawable origin.
type Material func(x, y int, t Time, cell Cell) Cell

type World struct {
	next       Entity
	transforms map[Entity]*Transform
	drawables  map[Entity]Drawable
	behaviors  map[Entity]Behavior
	materials  map[Entity]Material
	children   map[Entity][]Entity
	roots      []Entity
}

func NewWorld() *World {
	return &World{
		transforms: make(map[Entity]*Transform),
		drawables:  make(map[Entity]Drawable),
		behaviors:  make(map[Entity]Behavior),
		materials:  make(map[Entity]Material),
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

func (w *World) AddDrawable(e Entity, d Drawable) {
	w.drawables[e] = d
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

func (w *World) Drawable(e Entity) Drawable {
	return w.drawables[e]
}

func (w *World) AddBehavior(e Entity, b Behavior) {
	w.behaviors[e] = b
}

func (w *World) Behavior(e Entity) Behavior {
	return w.behaviors[e]
}

func (w *World) AddMaterial(e Entity, m Material) {
	w.materials[e] = m
}

func (w *World) Material(e Entity) Material {
	return w.materials[e]
}
