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

// Video, logo, image editing and artifact assets run under the host claim, so
// they need write permission but not the worker's atomic guard. Tools with no
// recovery story (canvas_production) stay out of DSH sessions.
func TestDSHCanvasCatalogExposesClaimedTools(t *testing.T) {
	object := json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
	names := []string{"media_get_video_model", "media_generate_video", "canvas_create_video", "media_edit_image",
		"canvas_add_artifact_asset", "design_catalog_logo_batch", "canvas_update_node", "canvas_production"}
	catalog := make([]desktopDesignTool, 0, len(names))
	for _, name := range names {
		catalog = append(catalog, desktopDesignTool{Name: name, Schema: object})
	}
	got := func(canWrite bool) []string {
		var out []string
		for _, tool := range dshCanvasRestrictedCatalog(catalog, canWrite, "{}") {
			out = append(out, tool.Name)
		}
		return out
	}
	if want := names[:len(names)-1]; !reflect.DeepEqual(got(true), want) {
		t.Fatalf("writable catalog = %v, want %v", got(true), want)
	}
	if want := []string{"media_get_video_model"}; !reflect.DeepEqual(got(false), want) {
		t.Fatalf("read-only catalog = %v, want %v", got(false), want)
	}
}
