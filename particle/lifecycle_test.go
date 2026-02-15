package particle

import (
	"math"
	"testing"

	"flicker/core"
)

func TestAgeAndDespawnIncrementsAge(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	w.AddAge(e, &core.Age{Age: 0.0, Lifetime: 0.0}) // Lifetime=0 means infinite.

	behavior := AgeAndDespawn()

	// Step 1: dt=0.1
	behavior(core.Time{Delta: 0.1}, e, w)

	age := w.Age(e)
	if age == nil {
		t.Fatal("Entity was despawned unexpectedly")
	}
	if math.Abs(age.Age-0.1) > 0.001 {
		t.Errorf("Expected age=0.1, got %f", age.Age)
	}

	// Step 2: dt=0.2
	behavior(core.Time{Delta: 0.2}, e, w)

	age = w.Age(e)
	if age == nil {
		t.Fatal("Entity was despawned unexpectedly")
	}
	if math.Abs(age.Age-0.3) > 0.001 {
		t.Errorf("Expected age=0.3, got %f", age.Age)
	}
}

func TestAgeAndDespawnLifetime(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	w.AddAge(e, &core.Age{Age: 0.0, Lifetime: 1.0})

	behavior := AgeAndDespawn()

	// Step 1: age to 0.5 (below lifetime).
	behavior(core.Time{Delta: 0.5}, e, w)

	age := w.Age(e)
	if age == nil {
		t.Fatal("Entity was despawned too early")
	}

	// Step 2: age to 1.5 (exceeds lifetime).
	behavior(core.Time{Delta: 1.0}, e, w)

	age = w.Age(e)
	if age != nil {
		t.Errorf("Entity should have been despawned, but age component still exists")
	}
}

func TestAgeAndDespawnWithoutAge(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity without Age component should not crash.
	behavior := AgeAndDespawn()
	behavior(core.Time{Delta: 0.1}, e, w)

	// No panic = success.
}
