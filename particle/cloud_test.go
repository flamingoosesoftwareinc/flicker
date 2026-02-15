package particle

import (
	"math"
	"testing"

	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
)

func TestBitmapToCloud(t *testing.T) {
	// Create a 10x10 bitmap with 5 non-transparent pixels.
	bm := bitmap.New(10, 10)
	bm.Set(1, 1, core.Color{R: 255, G: 255, B: 255}, 1.0)
	bm.Set(2, 2, core.Color{R: 255, G: 255, B: 255}, 1.0)
	bm.Set(3, 3, core.Color{R: 255, G: 255, B: 255}, 1.0)
	bm.Set(4, 4, core.Color{R: 255, G: 255, B: 255}, 1.0)
	bm.Set(5, 5, core.Color{R: 255, G: 255, B: 255}, 1.0)

	cloud := BitmapToCloud(bm)

	if len(cloud) != 5 {
		t.Errorf("Expected cloud with 5 positions, got %d", len(cloud))
	}

	// Verify positions are correct.
	expected := []fmath.Vec2{
		{X: 1, Y: 1},
		{X: 2, Y: 2},
		{X: 3, Y: 3},
		{X: 4, Y: 4},
		{X: 5, Y: 5},
	}

	for i, pos := range expected {
		if math.Abs(cloud[i].X-pos.X) > 0.001 || math.Abs(cloud[i].Y-pos.Y) > 0.001 {
			t.Errorf("Expected cloud[%d]=%v, got %v", i, pos, cloud[i])
		}
	}
}

func TestBitmapToCloudEmpty(t *testing.T) {
	// Empty bitmap should produce empty cloud.
	bm := bitmap.New(10, 10)

	cloud := BitmapToCloud(bm)

	if len(cloud) != 0 {
		t.Errorf("Expected empty cloud, got %d positions", len(cloud))
	}
}

func TestDistributeTargets(t *testing.T) {
	w := core.NewWorld()

	// Create 10 entities.
	entities := make([]core.Entity, 10)
	for i := 0; i < 10; i++ {
		e := w.Spawn()
		entities[i] = e
		w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	}

	// Create a cloud with 3 positions (fewer than entities).
	cloud := []fmath.Vec2{
		{X: 10, Y: 10},
		{X: 20, Y: 20},
		{X: 30, Y: 30},
	}

	speed := 5.0

	// Distribute targets.
	result := DistributeTargets(entities, cloud, speed, w)

	// Should return same entities (no spawning when cloud < entities).
	if len(result) != len(entities) {
		t.Errorf("Expected %d entities, got %d", len(entities), len(result))
	}

	// Verify all entities have behaviors.
	for i, e := range result {
		if len(w.Behaviors(e)) == 0 {
			t.Errorf("Entity %d should have behavior", i)
		}
	}

	// Run behaviors to verify they move towards targets.
	// Entity 0 should move towards cloud[0], entity 1 towards cloud[1], etc.
	// Entity 3 should wrap to cloud[0], entity 4 to cloud[1], etc.
	for i := 0; i < 10; i++ {
		e := result[i]
		behavior := w.Behaviors(e)[0]
		behavior.Update(core.Time{Delta: 1.0}, e, w)

		transform := w.Transform(e)

		// After 1 second at speed 5, should have moved towards target.
		// Can't verify exact position (depends on distance), but should be closer.
		// Just verify it moved in the right direction.
		if i%3 == 0 {
			// Should move towards (10, 10).
			if transform.Position.X <= 0 {
				t.Errorf("Entity %d should have moved towards target", i)
			}
		}
	}
}

func TestDistributeTargetsEmptyCloud(t *testing.T) {
	w := core.NewWorld()

	entities := []core.Entity{w.Spawn()}
	cloud := []fmath.Vec2{}

	// Should not crash with empty cloud.
	result := DistributeTargets(entities, cloud, 1.0, w)

	// Should return original entities unchanged.
	if len(result) != len(entities) {
		t.Errorf("Expected %d entities, got %d", len(entities), len(result))
	}

	// No panic = success.
}

func TestDistributeTargetsSpawnsNewParticles(t *testing.T) {
	w := core.NewWorld()

	// Create 3 entities with full components.
	entities := make([]core.Entity, 3)
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

	for i := 0; i < 3; i++ {
		e := w.Spawn()
		entities[i] = e
		w.AddTransform(e, &core.Transform{
			Position: fmath.Vec3{X: float64(i), Y: 0},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddBody(e, &core.Body{})
		w.AddDrawable(e, &bitmap.Braille{Bitmap: pixel})
		w.AddLayer(e, 1)
		w.AddRoot(e)
	}

	// Create a cloud with 7 positions (more than entities).
	cloud := []fmath.Vec2{
		{X: 10, Y: 10},
		{X: 20, Y: 20},
		{X: 30, Y: 30},
		{X: 40, Y: 40},
		{X: 50, Y: 50},
		{X: 60, Y: 60},
		{X: 70, Y: 70},
	}

	speed := 5.0

	// Distribute targets - should spawn 4 new particles.
	result := DistributeTargets(entities, cloud, speed, w)

	// Should have 7 entities total now.
	if len(result) != 7 {
		t.Errorf("Expected 7 entities (3 original + 4 spawned), got %d", len(result))
	}

	// Verify all entities have behaviors and components.
	for i, e := range result {
		if len(w.Behaviors(e)) == 0 {
			t.Errorf("Entity %d should have behavior", i)
		}
		if w.Transform(e) == nil {
			t.Errorf("Entity %d should have transform", i)
		}
		if w.Body(e) == nil {
			t.Errorf("Entity %d should have body", i)
		}
		if w.Drawable(e) == nil {
			t.Errorf("Entity %d should have drawable", i)
		}
		if w.Layer(e) != 1 {
			t.Errorf("Entity %d should have layer 1, got %d", i, w.Layer(e))
		}
	}

	// Verify newly spawned particles are in roots.
	roots := w.Roots()
	if len(roots) != 7 {
		t.Errorf("Expected 7 root entities, got %d", len(roots))
	}
}
