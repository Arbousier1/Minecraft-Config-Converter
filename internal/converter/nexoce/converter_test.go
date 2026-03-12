package nexoce

import "testing"

func TestConvertItemWritesCustomModelDataAtTopLevel(t *testing.T) {
	converter := New("demo")

	converter.convertItem("blade", map[string]any{
		"material": "STONE",
		"model":    4321,
	})

	item := converter.config.Items["demo:blade"]
	if got, want := item["custom-model-data"], 4321; got != want {
		t.Fatalf("custom-model-data mismatch: got %v want %v", got, want)
	}

	data := item["data"].(map[string]any)
	if _, exists := data["custom-model-data"]; exists {
		t.Fatalf("custom-model-data should not be nested in data: %#v", data)
	}
}
