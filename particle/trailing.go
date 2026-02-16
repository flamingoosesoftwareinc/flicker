package particle

import (
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/physics"
)

// TrailingEmitter spawns dust particles from a moving entity's specified position.
// Particles are bright white, drift with physics, and fade over their lifetime.
type TrailingEmitter struct {
	lastPos      fmath.Vec3
	initialized  bool
	EmitRate     float64    // particles per unit distance moved
	ParticleLife float64    // particle lifetime in seconds
	Offset       fmath.Vec2 // offset from entity position (e.g., bottom-left corner)
	Width        float64    // width of emission area (0 = single point, >0 = spread across width)
	Color        core.Color // particle color (typically bright white)
}

// NewTrailingEmitter creates a trailing emitter with default settings.
func NewTrailingEmitter(offset fmath.Vec2) *TrailingEmitter {
	return &TrailingEmitter{
		EmitRate:     0.5, // 1 particle per 2 pixels moved
		ParticleLife: 1.5, // 1.5 seconds lifetime
		Offset:       offset,
		Color:        core.Color{R: 255, G: 255, B: 255}, // Bright white
	}
}

// Enabled returns true (always active).
func (te *TrailingEmitter) Enabled() bool {
	return true
}

// Update spawns dust particles based on entity movement.
func (te *TrailingEmitter) Update(t core.Time, e core.Entity, w *core.World) {
	trans := w.Transform(e)
	if trans == nil {
		return
	}

	// Initialize on first update
	if !te.initialized {
		te.lastPos = trans.Position
		te.initialized = true
		return
	}

	// Calculate velocity (distance moved since last frame)
	delta := trans.Position.Sub(te.lastPos)
	distance := fmath.Vec2{X: delta.X, Y: delta.Y}.Length()

	// Only spawn if moving (threshold: 0.01 pixels - reduced from 0.1)
	if distance > 0.01 {
		// Number of particles to spawn based on distance
		count := int(distance * te.EmitRate)
		if count < 1 && distance > 0.01 {
			count = 1 // At least 1 particle if moving at all
		}

		// Spawn dust particles along the path
		for i := 0; i < count; i++ {
			// Interpolate position along movement path
			t := float64(i) / float64(count)
			particlePos := fmath.Vec2{
				X: te.lastPos.X + delta.X*t + te.Offset.X,
				Y: te.lastPos.Y + delta.Y*t + te.Offset.Y,
			}

			// Hash-based random offset for variety
			hash := (int(particlePos.X*73) ^ int(particlePos.Y*31) ^ int(t*100)) & 0xFF
			hashNorm := float64(hash) / 255.0

			// Spread particles across emission width (if specified)
			if te.Width > 0 {
				particlePos.X += hashNorm * te.Width
			}

			// Add slight vertical random offset
			randomOffset := fmath.Vec2{
				X: (hashNorm - 0.5) * 2.0,
				Y: (float64((hash*17)&0xFF)/255.0 - 0.5) * 1.5,
			}
			particlePos = particlePos.Add(randomOffset)

			te.spawnDustParticle(w, particlePos)
		}
	}

	te.lastPos = trans.Position
}

// spawnDustParticle creates a single dust particle entity.
func (te *TrailingEmitter) spawnDustParticle(w *core.World, pos fmath.Vec2) {
	particle := w.Spawn()

	// Position
	w.AddTransform(particle, &core.Transform{
		Position: fmath.Vec3{X: pos.X, Y: pos.Y, Z: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})

	// Render as small white dot
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, te.Color)
	w.AddDrawable(particle, &bitmap.Braille{Bitmap: pixel})

	// Age and lifetime
	w.AddAge(particle, &core.Age{
		Age:      0,
		Lifetime: te.ParticleLife,
	})

	// Physics - slight downward drift + random horizontal
	hash := (int(pos.X*13) ^ int(pos.Y*7)) & 0xFF
	randomVelX := (float64(hash)/255.0 - 0.5) * 3.0
	randomVelY := (float64((hash*23)&0xFF) / 255.0) * 2.0

	w.AddBody(particle, &core.Body{
		Velocity: fmath.Vec2{
			X: randomVelX,
			Y: 2.0 + randomVelY, // Drift downward
		},
	})

	// Add physics behaviors
	w.AddBehavior(particle, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
		// Euler integration for movement
		physics.EulerIntegration()(t, e, w)

		// Gravity (gentle downward acceleration)
		physics.Gravity(fmath.Vec2{X: 0, Y: 1.5})(t, e, w)

		// Turbulence for wobbly movement
		physics.Turbulence(0.3, 2.0)(t, e, w)

		// Drag to slow down
		physics.Drag(0.98)(t, e, w)

		// Age and despawn when lifetime expires
		AgeAndDespawn()(t, e, w)
	}))

	// Material: fade based on age
	w.AddMaterial(particle, func(f core.Fragment) core.Cell {
		c := f.Cell
		age := f.World.Age(f.Entity)
		if age != nil {
			// Fade out over lifetime
			progress := age.Age / age.Lifetime
			c.FGAlpha = 1.0 - progress // Fade from 1.0 to 0.0
			c.BGAlpha = 0.0            // Transparent background
		}
		return c
	})

	w.AddRoot(particle)
}
