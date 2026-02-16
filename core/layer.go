package core

import "sort"

// LayerPostProcess is a per-cell fragment shader applied to a layer canvas
// before compositing. Source points to a snapshot so reads are stable.
type LayerPostProcess func(f Fragment) Cell

// LayerPreProcess is a per-cell fragment shader applied to a layer canvas
// before rendering entities. Used for trail effects - instead of clearing,
// the previous frame is transformed (faded, blurred, etc.). Source points
// to a snapshot so reads are stable.
type LayerPreProcess func(f Fragment) Cell

// Compositor owns per-layer canvases and composites them back-to-front.
type Compositor struct {
	width, height int
	layers        map[int]*Canvas
	preProcess    map[int]LayerPreProcess
	postProcess   map[int]LayerPostProcess
	blends        map[int]ColorBlend
	scratch       *Canvas // reusable snapshot buffer for post-process
	scratch2      *Canvas // second snapshot buffer for pre-process
}

// NewCompositor creates a compositor for the given canvas dimensions.
func NewCompositor(w, h int) *Compositor {
	return &Compositor{
		width:       w,
		height:      h,
		layers:      make(map[int]*Canvas),
		preProcess:  make(map[int]LayerPreProcess),
		postProcess: make(map[int]LayerPostProcess),
		blends:      make(map[int]ColorBlend),
	}
}

// SetPreProcess registers a pre-process pass for the given layer.
// When set, the layer canvas is NOT cleared - instead the pre-process
// shader is applied to transform the previous frame (for trail effects).
func (c *Compositor) SetPreProcess(layer int, pp LayerPreProcess) {
	c.preProcess[layer] = pp
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
	// Clear or pre-process all layer canvases.
	for idx, lc := range c.layers {
		if pp, ok := c.preProcess[idx]; ok {
			// Apply pre-process shader instead of clearing (for trails)
			if c.scratch2 == nil {
				c.scratch2 = NewCanvas(c.width, c.height)
			}
			lc.CopyInto(c.scratch2)
			for y := range lc.Height {
				for x := range lc.Width {
					f := Fragment{
						X: x, Y: y,
						ScreenX: x, ScreenY: y,
						Time:   t,
						Cell:   c.scratch2.Get(x, y),
						Source: c.scratch2,
						World:  world,
						Entity: 0, // No specific entity for layer pre-process
					}
					lc.Set(x, y, pp(f))
				}
			}
		} else {
			// Normal clear
			lc.Clear()
		}
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
						World:  world,
						Entity: 0, // No specific entity for layer post-process
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
