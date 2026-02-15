package outfmt

import (
	"bytes"
	"context"
	"testing"
)

func TestFromFlags_Valid(t *testing.T) {
	mode, err := FromFlags(true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mode.JSON {
		t.Error("expected JSON=true")
	}

	if mode.Plain {
		t.Error("expected Plain=false")
	}
}

func TestFromFlags_BothSet(t *testing.T) {
	_, err := FromFlags(true, true)
	if err == nil {
		t.Fatal("expected error when both JSON and Plain are set")
	}
}

func TestContextRoundTrip(t *testing.T) {
	mode := Mode{JSON: true}
	ctx := WithMode(context.Background(), mode)

	got := FromContext(ctx)
	if !got.JSON {
		t.Error("expected JSON=true from context")
	}
}

func TestFromContext_Empty(t *testing.T) {
	mode := FromContext(context.Background())
	if mode.JSON || mode.Plain {
		t.Error("expected zero Mode from empty context")
	}
}

func TestIsJSON(t *testing.T) {
	ctx := WithMode(context.Background(), Mode{JSON: true})

	if !IsJSON(ctx) {
		t.Error("expected IsJSON=true")
	}

	if IsPlain(ctx) {
		t.Error("expected IsPlain=false")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer

	err := WriteJSON(&buf, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestKeyValuePayload(t *testing.T) {
	result := KeyValuePayload("name", "test")

	if result["key"] != "name" {
		t.Errorf("expected key 'name', got %q", result["key"])
	}

	if result["value"] != "test" {
		t.Errorf("expected value 'test', got %q", result["value"])
	}
}
