package core

type Entity uint64

// Material is a per-entity fragment shader applied after drawing.
type Material func(f Fragment) Cell

// ComposeMaterials combines multiple materials into a single material that
// applies them in sequence. Each material receives the output of the previous
// one as f.Cell. Use this when multiple effects need to modify the same entity.
// Returns nil if no materials are provided, or the single material if only one.
func ComposeMaterials(materials ...Material) Material {
	// Filter out nil materials.
	filtered := make([]Material, 0, len(materials))
	for _, m := range materials {
		if m != nil {
			filtered = append(filtered, m)
		}
	}

	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}

	return func(f Fragment) Cell {
		cell := f.Cell
		for _, mat := range filtered {
			f.Cell = cell
			cell = mat(f)
		}
		return cell
	}
}

type World struct {
	next         Entity
	transforms   map[Entity]*Transform
	drawables    map[Entity]Drawable
	behaviors    map[Entity]Behavior
	materials    map[Entity]Material
	layers       map[Entity]int
	cameras      map[Entity]*Camera
	children     map[Entity][]Entity
	roots        []Entity
	activeCamera Entity // 0 = no camera; safe because Spawn() starts at 1
}

func NewWorld() *World {
	return &World{
		transforms: make(map[Entity]*Transform),
		drawables:  make(map[Entity]Drawable),
		behaviors:  make(map[Entity]Behavior),
		materials:  make(map[Entity]Material),
		layers:     make(map[Entity]int),
		cameras:    make(map[Entity]*Camera),
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

func (w *World) AddLayer(e Entity, layer int) {
	w.layers[e] = layer
}

func (w *World) Layer(e Entity) int {
	return w.layers[e]
}

func (w *World) AddCamera(e Entity, c *Camera) {
	w.cameras[e] = c
}

func (w *World) Camera(e Entity) *Camera {
	return w.cameras[e]
}

func (w *World) SetActiveCamera(e Entity) {
	w.activeCamera = e
}

func (w *World) ActiveCamera() Entity {
	return w.activeCamera
}
