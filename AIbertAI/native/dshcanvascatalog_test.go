package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestGuardedImageToolAdvertisesWorkspaceGeneration(t *testing.T) {
	const name = "media_generate_image"
	const caps = `{"version":4,"tools":["media_generate_image"]}`
	raw := json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string","minLength":1},"parameters":{"type":"object","properties":{"count":{"enum":[1]}}},"anchor_node_id":{"type":["string","null"]}},"required":["prompt"],"additionalProperties":false}`)
	catalog := []desktopDesignTool{{Name: name, Description: "anchor_node_id", Schema: raw}}
	selected := dshCanvasRestrictedCatalog(catalog, true, caps)
	if len(selected) != 1 || !strings.Contains(selected[0].Description, "source_path") || strings.Contains(selected[0].Description, "anchor_node_id") {
		t.Fatalf("unexpected image tool: %+v", selected)
	}
	var schema struct {
		Properties           map[string]json.RawMessage
		Required             []string
		AdditionalProperties bool
	}
	if err := json.Unmarshal(selected[0].Schema, &schema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(schema.Required, []string{"prompt"}) || len(schema.Properties) != 2 || schema.AdditionalProperties ||
		string(schema.Properties["prompt"]) != `{"minLength":1,"type":"string"}` ||
		string(schema.Properties["parameters"]) != `{"properties":{"count":{"enum":[1]}},"type":"object"}` {
		t.Fatalf("unexpected image schema: %s", selected[0].Schema)
	}
	if len(dshCanvasRestrictedCatalog(catalog, false, caps)) != 0 || len(dshCanvasRestrictedCatalog(catalog, true, "{}")) != 0 {
		t.Fatal("image generation requires write permission and worker guard support")
	}
	if !strings.Contains(string(catalog[0].Schema), "anchor_node_id") {
		t.Fatal("DSH selection mutated the worker's source schema")
	}
}
