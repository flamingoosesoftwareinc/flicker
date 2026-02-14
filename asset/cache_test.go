package asset

import "testing"

func TestCacheMiss(t *testing.T) {
	c := NewCache()
	v, ok := c.Get("nope")
	if ok || v != nil {
		t.Errorf("miss should return nil/false, got %v, %v", v, ok)
	}
}

func TestCachePutGet(t *testing.T) {
	c := NewCache()
	c.Put("a", 42)
	v, ok := c.Get("a")
	if !ok || v != 42 {
		t.Errorf("roundtrip: got %v, %v", v, ok)
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := NewCache()
	c.Put("k", "first")
	c.Put("k", "second")
	v, ok := c.Get("k")
	if !ok || v != "second" {
		t.Errorf("overwrite: got %v, %v", v, ok)
	}
}

func TestCacheMixedTypes(t *testing.T) {
	c := NewCache()
	c.Put("int", 1)
	c.Put("str", "hello")
	c.Put("slice", []int{1, 2, 3})

	if v, ok := c.Get("int"); !ok || v != 1 {
		t.Errorf("int: got %v, %v", v, ok)
	}
	if v, ok := c.Get("str"); !ok || v != "hello" {
		t.Errorf("str: got %v, %v", v, ok)
	}
	if v, ok := c.Get("slice"); !ok {
		t.Errorf("slice: got %v, %v", v, ok)
	}
}
