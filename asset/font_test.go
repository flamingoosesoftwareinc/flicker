package asset

import "testing"

func TestLoadFont(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}
	m := f.Metrics(32)
	if m.Ascent <= 0 {
		t.Errorf("expected Ascent > 0, got %f", m.Ascent)
	}
	if m.Descent <= 0 {
		t.Errorf("expected Descent > 0, got %f", m.Descent)
	}
	if m.Height != m.Ascent+m.Descent {
		t.Errorf("Height (%f) != Ascent (%f) + Descent (%f)", m.Height, m.Ascent, m.Descent)
	}
}

func TestLoadFontNotFound(t *testing.T) {
	_, err := LoadFont("nonexistent.ttf")
	if err == nil {
		t.Fatalf("expected error for nonexistent path, got nil")
	}
}

func TestCacheGetOrLoadFont(t *testing.T) {
	c := NewCache()
	f1, err := c.GetOrLoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("GetOrLoadFont: %v", err)
	}
	f2, err := c.GetOrLoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("GetOrLoadFont (cached): %v", err)
	}
	if f1 != f2 {
		t.Errorf("cache miss: expected same pointer on second call")
	}
}
