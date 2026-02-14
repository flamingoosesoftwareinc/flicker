package core

import "sort"

// LayerPostProcess is a per-cell fragment shader applied to a layer canvas
// before compositing. Source points to a snapshot so reads are stable.
type LayerPostProcess func(f Fragment) Cell

// Compositor owns per-layer canvases and composites them back-to-front.
type Compositor struct {
	width, height int
	layers        map[int]*Canvas
	postProcess   map[int]LayerPostProcess
	blends        map[int]ColorBlend
	scratch       *Canvas // reusable snapshot buffer for post-process
}

// NewCompositor creates a compositor for the given canvas dimensions.
func NewCompositor(w, h int) *Compositor {
	return &Compositor{
		width:       w,
		height:      h,
		layers:      make(map[int]*Canvas),
		postProcess: make(map[int]LayerPostProcess),
		blends:      make(map[int]ColorBlend),
	}
}

// SetPostProcess registers a post-process pass for the given layer.
func (c *Compositor) SetPostProcess(layer int, pp LayerPostProcess) {
	c.postProcess[layer] = pp
}

// SetBlend configures a ColorBlend for the given layer. Layers without
// an explicit blend use NormalColorBlend.
func (c *Compositor) SetBlend(layer int, blend ColorBlend) {
	c.blends[layer] = blend
}

// getLayer returns the canvas for the given layer index, creating it if needed.
func (c *Compositor) getLayer(idx int) *Canvas {
	if lc, ok := c.layers[idx]; ok {
		return lc
	}
	lc := NewCanvas(c.width, c.height)
	c.layers[idx] = lc
	return lc
}

// Composite renders all root entities into per-layer canvases, applies
// post-process passes, and composites layers back-to-front onto dst.
func (c *Compositor) Composite(world *World, dst *Canvas, t Time) {
	// Clear all layer canvases.
	for _, lc := range c.layers {
		lc.Clear()
	}

	// Compute view matrix once for all layers.
	view := viewMatrix(world, c.width, c.height)

	// Render each root's entity tree into its layer canvas.
	for _, root := range world.Roots() {
		idx := world.Layer(root)
		lc := c.getLayer(idx)
		renderEntity(world, lc, root, view, t)
	}

	// Collect and sort layer indices ascending (back-to-front).
	indices := make([]int, 0, len(c.layers))
	for idx := range c.layers {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	// Apply post-process per layer, then composite onto dst.
	for _, idx := range indices {
		lc := c.layers[idx]
		if pp, ok := c.postProcess[idx]; ok {
			if c.scratch == nil {
				c.scratch = NewCanvas(c.width, c.height)
			}
			lc.CopyInto(c.scratch)
			for y := range lc.Height {
				for x := range lc.Width {
					f := Fragment{
						X: x, Y: y,
						ScreenX: x, ScreenY: y,
						Time:   t,
						Cell:   c.scratch.Get(x, y),
						Source: c.scratch,
					}
					lc.Set(x, y, pp(f))
				}
			}
		}
		blend := NormalColorBlend
		if b, ok := c.blends[idx]; ok {
			blend = b
		}
		dst.Composite(lc, blend)
	}
}
