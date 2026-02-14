package core

import "sort"

// LayerPostProcess is applied to an entire layer canvas before compositing.
type LayerPostProcess func(canvas *Canvas, t Time)

// Compositor owns per-layer canvases and composites them back-to-front.
type Compositor struct {
	width, height int
	layers        map[int]*Canvas
	postProcess   map[int]LayerPostProcess
	blends        map[int]ColorBlend
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

	// Render each root's entity tree into its layer canvas.
	for _, root := range world.Roots() {
		idx := world.Layer(root)
		lc := c.getLayer(idx)
		renderEntity(world, lc, root, 0, 0, 0, t)
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
			pp(lc, t)
		}
		blend := NormalColorBlend
		if b, ok := c.blends[idx]; ok {
			blend = b
		}
		dst.Composite(lc, blend)
	}
}
