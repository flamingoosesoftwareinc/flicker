package particle

import (
	"math"

	"flicker/core"
	"flicker/fmath"
	"flicker/physics"
)

// PhaseContext provides the environment for a transition phase.
type PhaseContext struct {
	Particles []core.Entity
	Targets   []fmath.Vec2
	Duration  float64
	World     *core.World
}

// PhaseController manages a single phase of a transition.
type PhaseController interface {
	Update(elapsed float64) bool // Returns true when phase is complete
	End()                        // Cleanup when phase ends
}

// TransitionPhase defines one phase of a morph transition.
type TransitionPhase interface {
	Start(ctx PhaseContext) PhaseController
}

// ============================================================================
// Behavior-Based Phase (Simulation)
// ============================================================================

// BehaviorPhase uses ECS behaviors for particle movement.
type BehaviorPhase struct {
	CreateBehaviors func(PhaseContext) []core.Behavior
}

type behaviorController struct {
	behaviors []core.Behavior
	duration  float64
}

func (p *BehaviorPhase) Start(ctx PhaseContext) PhaseController {
	behaviors := p.CreateBehaviors(ctx)
	return &behaviorController{
		behaviors: behaviors,
		duration:  ctx.Duration,
	}
}

func (c *behaviorController) Update(elapsed float64) bool {
	return elapsed >= c.duration
}

func (c *behaviorController) End() {
	// Disable all behaviors
	for _, b := range c.behaviors {
		if fb, ok := b.(*core.FuncBehavior); ok {
			fb.SetEnabled(false)
		}
	}
}

// SeekPhase creates a behavior phase that moves particles to targets.
func SeekPhase() *BehaviorPhase {
	return &BehaviorPhase{
		CreateBehaviors: func(ctx PhaseContext) []core.Behavior {
			behaviors := make([]core.Behavior, 0, len(ctx.Particles))

			// Calculate speed for this phase
			speed := CalculateSpeedForDuration(
				ctx.Particles,
				ctx.Targets,
				ctx.Duration*0.9,
				ctx.World,
			)

			for i, p := range ctx.Particles {
				targetIdx := i % len(ctx.Targets)
				b := ctx.World.AddBehavior(
					p,
					core.NewBehavior(InterpolateToTarget(ctx.Targets[targetIdx], speed)),
				)
				behaviors = append(behaviors, b)
			}

			return behaviors
		},
	}
}

// burstOutwardEased creates a behavior that moves particles radially outward with easing.
func burstOutwardEased(
	center fmath.Vec2,
	distance, duration float64,
	easing func(float64) float64,
) core.BehaviorFunc {
	type particleState struct {
		startPos  fmath.Vec2
		direction fmath.Vec2
		startTime float64
	}
	state := &particleState{startTime: -1}

	return func(t core.Time, e core.Entity, w *core.World) {
		transform := w.Transform(e)
		body := w.Body(e)
		if transform == nil {
			return
		}

		// Initialize on first frame
		if state.startTime < 0 {
			state.startTime = t.Total
			state.startPos = fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}

			delta := state.startPos.Sub(center)
			if delta.X == 0 && delta.Y == 0 {
				delta = fmath.Vec2{X: 1, Y: 0}
			}
			state.direction = delta.Normalize()
		}

		// Calculate progress
		elapsed := t.Total - state.startTime
		progress := elapsed / duration
		if progress > 1.0 {
			progress = 1.0
		}

		// Apply easing to get target distance
		easedProgress := easing(progress)
		targetDistance := distance * easedProgress

		// Calculate target position
		targetPos := fmath.Vec2{
			X: state.startPos.X + state.direction.X*targetDistance,
			Y: state.startPos.Y + state.direction.Y*targetDistance,
		}

		// Update position
		oldX := transform.Position.X
		oldY := transform.Position.Y
		transform.Position.X = targetPos.X
		transform.Position.Y = targetPos.Y

		// Update velocity for directional materials
		if body != nil && t.Delta > 0 {
			body.Velocity = fmath.Vec2{
				X: (targetPos.X - oldX) / t.Delta,
				Y: (targetPos.Y - oldY) / t.Delta,
			}
		}
	}
}

// BurstPhase creates a phase that bursts particles outward from centroid with easing.
func BurstPhase(distance float64) *BehaviorPhase {
	return BurstPhaseWithEasing(distance, EaseInOutQuad)
}

// BurstPhaseWithEasing creates a burst phase with custom easing function.
func BurstPhaseWithEasing(distance float64, easing func(float64) float64) *BehaviorPhase {
	return &BehaviorPhase{
		CreateBehaviors: func(ctx PhaseContext) []core.Behavior {
			behaviors := make([]core.Behavior, 0, len(ctx.Particles))

			// Calculate centroid
			center := fmath.Vec2{X: 0, Y: 0}
			for _, p := range ctx.Particles {
				if tr := ctx.World.Transform(p); tr != nil {
					center.X += tr.Position.X
					center.Y += tr.Position.Y
				}
			}
			if len(ctx.Particles) > 0 {
				center.X /= float64(len(ctx.Particles))
				center.Y /= float64(len(ctx.Particles))
			}

			// Create eased burst behavior for each particle
			for _, p := range ctx.Particles {
				b := ctx.World.AddBehavior(
					p,
					core.NewBehavior(burstOutwardEased(center, distance, ctx.Duration, easing)),
				)
				behaviors = append(behaviors, b)
			}

			return behaviors
		},
	}
}

// TurbulencePhase adds turbulent motion.
func TurbulencePhase(scale, strength float64) *BehaviorPhase {
	return &BehaviorPhase{
		CreateBehaviors: func(ctx PhaseContext) []core.Behavior {
			behaviors := make([]core.Behavior, 0, len(ctx.Particles))

			for _, p := range ctx.Particles {
				b := ctx.World.AddBehavior(
					p,
					core.NewBehavior(physics.Turbulence(scale, strength)),
				)
				behaviors = append(behaviors, b)
			}

			return behaviors
		},
	}
}

// ============================================================================
// Keyframe-Based Phase (Pre-animated)
// ============================================================================

// KeyframePhase directly manipulates transforms using easing functions.
type KeyframePhase struct {
	Easing func(t float64) float64 // Easing function (0.0 to 1.0)
}

type keyframeController struct {
	particles []core.Entity
	startPos  []fmath.Vec2
	targetPos []fmath.Vec2
	duration  float64
	easing    func(float64) float64
	world     *core.World
}

func (p *KeyframePhase) Start(ctx PhaseContext) PhaseController {
	controller := &keyframeController{
		particles: ctx.Particles,
		startPos:  make([]fmath.Vec2, len(ctx.Particles)),
		targetPos: make([]fmath.Vec2, len(ctx.Targets)),
		duration:  ctx.Duration,
		easing:    p.Easing,
		world:     ctx.World,
	}

	// Capture starting positions
	for i, p := range ctx.Particles {
		if tr := ctx.World.Transform(p); tr != nil {
			controller.startPos[i] = fmath.Vec2{X: tr.Position.X, Y: tr.Position.Y}
		}
	}

	// Copy target positions
	copy(controller.targetPos, ctx.Targets)

	return controller
}

func (c *keyframeController) Update(elapsed float64) bool {
	t := elapsed / c.duration
	if t > 1.0 {
		t = 1.0
	}

	// Apply easing
	easedT := c.easing(t)

	// Directly set positions
	for i, p := range c.particles {
		targetIdx := i % len(c.targetPos)
		start := c.startPos[i]
		target := c.targetPos[targetIdx]

		newPos := fmath.Vec2{
			X: fmath.Lerp(start.X, target.X, easedT),
			Y: fmath.Lerp(start.Y, target.Y, easedT),
		}

		if tr := c.world.Transform(p); tr != nil {
			tr.Position.X = newPos.X
			tr.Position.Y = newPos.Y
		}

		// Update velocity for directional materials
		if body := c.world.Body(p); body != nil {
			if elapsed > 0 {
				velocity := fmath.Vec2{
					X: (newPos.X - start.X) / elapsed,
					Y: (newPos.Y - start.Y) / elapsed,
				}
				body.Velocity = velocity
			}
		}
	}

	return elapsed >= c.duration
}

func (c *keyframeController) End() {
	// Ensure all particles reached their targets
	for i, p := range c.particles {
		targetIdx := i % len(c.targetPos)
		if tr := c.world.Transform(p); tr != nil {
			tr.Position.X = c.targetPos[targetIdx].X
			tr.Position.Y = c.targetPos[targetIdx].Y
		}
		if body := c.world.Body(p); body != nil {
			body.Velocity = fmath.Vec2{X: 0, Y: 0}
		}
	}
}

// Easing functions
func EaseLinear(t float64) float64 {
	return t
}

func EaseInQuad(t float64) float64 {
	return t * t
}

func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func EaseInCubic(t float64) float64 {
	return t * t * t
}

func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

// ============================================================================
// Curve-Based Phase (Procedural)
// ============================================================================

// CurvePhase uses Bezier curves for smooth arcing motion.
type CurvePhase struct {
	ArcHeight float64 // Height of arc above midpoint
}

type curveController struct {
	particles []core.Entity
	curves    []bezierCurve
	duration  float64
	world     *core.World
}

type bezierCurve struct {
	p0, p1, p2, p3 fmath.Vec2
}

func (c bezierCurve) sample(t float64) fmath.Vec2 {
	// Cubic Bezier: B(t) = (1-t)³P₀ + 3(1-t)²tP₁ + 3(1-t)t²P₂ + t³P₃
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt
	t2 := t * t
	t3 := t2 * t

	return fmath.Vec2{
		X: mt3*c.p0.X + 3*mt2*t*c.p1.X + 3*mt*t2*c.p2.X + t3*c.p3.X,
		Y: mt3*c.p0.Y + 3*mt2*t*c.p1.Y + 3*mt*t2*c.p2.Y + t3*c.p3.Y,
	}
}

func (p *CurvePhase) Start(ctx PhaseContext) PhaseController {
	controller := &curveController{
		particles: ctx.Particles,
		curves:    make([]bezierCurve, len(ctx.Particles)),
		duration:  ctx.Duration,
		world:     ctx.World,
	}

	// Build Bezier curve for each particle
	for i, particle := range ctx.Particles {
		targetIdx := i % len(ctx.Targets)
		target := ctx.Targets[targetIdx]

		var start fmath.Vec2
		if tr := ctx.World.Transform(particle); tr != nil {
			start = fmath.Vec2{X: tr.Position.X, Y: tr.Position.Y}
		}

		// Create control points for arc
		mid := fmath.Vec2{
			X: (start.X + target.X) / 2,
			Y: (start.Y + target.Y) / 2,
		}

		// Perpendicular offset for arc
		dx := target.X - start.X
		dy := target.Y - start.Y
		perpX := -dy
		perpY := dx
		length := math.Sqrt(perpX*perpX + perpY*perpY)
		if length > 0 {
			perpX /= length
			perpY /= length
		}

		// Control points create arc
		controller.curves[i] = bezierCurve{
			p0: start,
			p1: fmath.Vec2{X: mid.X + perpX*p.ArcHeight, Y: mid.Y + perpY*p.ArcHeight},
			p2: fmath.Vec2{X: mid.X + perpX*p.ArcHeight, Y: mid.Y + perpY*p.ArcHeight},
			p3: target,
		}
	}

	return controller
}

func (c *curveController) Update(elapsed float64) bool {
	t := elapsed / c.duration
	if t > 1.0 {
		t = 1.0
	}

	for i, p := range c.particles {
		pos := c.curves[i].sample(t)

		if tr := c.world.Transform(p); tr != nil {
			oldX := tr.Position.X
			oldY := tr.Position.Y
			tr.Position.X = pos.X
			tr.Position.Y = pos.Y

			// Update velocity for directional materials
			if body := c.world.Body(p); body != nil && elapsed > 0 {
				body.Velocity = fmath.Vec2{
					X: (pos.X - oldX) / 0.016, // Approximate frame time
					Y: (pos.Y - oldY) / 0.016,
				}
			}
		}
	}

	return elapsed >= c.duration
}

func (c *curveController) End() {
	// Ensure particles reached targets
	for i, p := range c.particles {
		pos := c.curves[i].p3
		if tr := c.world.Transform(p); tr != nil {
			tr.Position.X = pos.X
			tr.Position.Y = pos.Y
		}
		if body := c.world.Body(p); body != nil {
			body.Velocity = fmath.Vec2{X: 0, Y: 0}
		}
	}
}
