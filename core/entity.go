package core

type Entity uint64

type World struct {
	next       Entity
	transforms map[Entity]*Transform
	drawables  map[Entity]Drawable
	children   map[Entity][]Entity
	roots      []Entity
}

func NewWorld() *World {
	return &World{
		transforms: make(map[Entity]*Transform),
		drawables:  make(map[Entity]Drawable),
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
