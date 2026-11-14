package model

import "testing"

func TestHasTagAddRemove(t *testing.T) {
	p := Project{}
	if p.HasTag("Work") {
		t.Fatal("expected no tag")
	}
	p.AddTag("Work")
	p.AddTag("Work") // duplicate
	if len(p.Tags) != 1 {
		t.Fatalf("dup not deduped: %v", p.Tags)
	}
	if !p.HasTag("Work") {
		t.Fatal("expected tag")
	}
	p.RemoveTag("Work")
	if p.HasTag("Work") {
		t.Fatal("remove failed")
	}
	p.RemoveTag("nope") // no-op
	if len(p.Tags) != 0 {
		t.Fatalf("unexpected: %v", p.Tags)
	}
}
