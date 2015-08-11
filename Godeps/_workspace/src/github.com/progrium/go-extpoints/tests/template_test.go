package main

import (
	"testing"

	"github.com/progrium/go-extpoints/tests/extpoints"
)

var noops = extpoints.Noops
var noopFactories = extpoints.NoopFactories
var transformers = extpoints.StringTransformers

func TestLookupSuccess(t *testing.T) {
	ext := noops.Lookup("noop")
	if ext == nil {
		t.Fatal("Lookup returned not ok for registered extension")
	}
}

func TestLookupFail(t *testing.T) {
	ext := noops.Lookup("yesop")
	if ext != nil {
		t.Fatal("Lookup returned ok for non-existent extension")
	}
}

func TestSelect(t *testing.T) {
	exts := noops.Select([]string{"noop3", "noop", "noop2"})
	if len(exts) != 3 {
		t.Fatal("Select return wrong number of elements")
	}
	if exts[0] != nil {
		t.Fatal("Select non-existant element not nil")
	}
	if exts[2].Noop() != "noop2" {
		t.Fatal("Select elements not in expected order")
	}
}

func TestUsingExtension(t *testing.T) {
	upper := transformers.Lookup("upper")
	if upper.Transform("string") != "STRING" {
		t.Fatal("Used extension, but didn't work as expected")
	}
}

func TestUsingFuncComponent(t *testing.T) {
	factory := noopFactories.Lookup("noopFactory")
	n := factory()
	if _, ok := n.(extpoints.Noop); !ok {
		t.Fatal("Used extension, but didn't work as expected")
	}
}
