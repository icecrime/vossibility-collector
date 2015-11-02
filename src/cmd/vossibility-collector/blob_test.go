package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBlobAttributes(t *testing.T) {
	b := NewBlob("event", "id")
	attrs := map[string]interface{}{
		"k1": "value1",
		"k2": true,
	}

	for k, v := range attrs {
		if err := b.Push(k, v); err != nil {
			t.Fatalf("push (%v, %v) failed: %v", k, v, err)
		}
		if !b.HasAttribute(k) {
			t.Fatalf("missing attribute %v after push", k)
		}
	}

	var m map[string]interface{}
	if e, err := b.Encode(); err != nil {
		t.Fatalf("failed to encode blob: %v", err)
	} else if err := json.Unmarshal(e, &m); err != nil {
		t.Fatalf("failed to deserialize encoded result: %v", err)
	}

	for k, v := range attrs {
		if res, ok := m[k]; !ok {
			t.Fatalf("missing attribute %q in encoded blob", k)
		} else if res != v {
			t.Fatalf("unexpected value %v for attribute %q (expected: %v)", res, k, v)
		}
	}
}

func TestBlobSpecialAttributes(t *testing.T) {
	b := NewBlob("event", "id")
	if err := b.Push("_bad", "test"); err == nil || !strings.Contains(err.Error(), "invalid metadata field") {
		t.Fatalf(`expected "invalid metadata field" error, got %v`, err)
	}
	if err := b.Push(MetadataType, true); err == nil || !strings.Contains(err.Error(), "bad value") {
		t.Fatalf(`exepected "bad value" error. got %v`, err)
	}
	if err := b.Push(MetadataType, "test"); err != nil {
		t.Fatalf("failed to set type metadata: %v", err)
	}
}

func TestBlobSnapshot(t *testing.T) {
	b := NewBlob("event", "id")
	if s := b.Snapshot(); s != nil {
		t.Fatalf("snapshotting an empty blob returns non-nil result %#v", s)
	}

	payload := map[string]interface{}{"foo": "bar", "number": 123}
	b.Push("dummy", false)
	b.Push("snapshot_field", payload)
	b.Push(MetadataSnapshotID, "number")
	b.Push(MetadataSnapshotField, "snapshot_field")

	var s *Blob
	if s = b.Snapshot(); b == nil {
		t.Fatalf("nil snapshot returned for a valid blob")
	}
	if s.Type != b.Type {
		t.Fatalf("snapshot has unexpected type %v (expected %v)", s.Type, b.Type)
	}
	if s.Timestamp != b.Timestamp {
		t.Fatalf("snapshot has unexpected timestamp %v (expected %v)", s.Timestamp, b.Timestamp)
	}
	if expected := "123"; s.ID != expected {
		t.Fatalf("snapshot has unexpected ID %v (expected: %v)", s.ID, expected)
	}

	p, _ := json.Marshal(payload)
	r, _ := NewBlobFromPayload("event", "id", p)
	if rb, err := r.Encode(); err != nil {
		t.Fatalf("encoding blob failed: %v", err)
	} else if sb, err := s.Encode(); err != nil {
		t.Fatalf("encoding snapshot failed: %v", err)
	} else if bytes.Compare(rb, sb) != 0 {
		t.Fatalf("snapshot has unexpected payload %s (expected %s)", sb, rb)
	}
}
