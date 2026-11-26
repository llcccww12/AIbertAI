package main

import (
	"encoding/json"
)

// dshCanvasRestrictedCatalog filters the platform's current catalog before it
// is exposed to the SDK. Guarded tools need the connected desktop's capability
// response. Canvas-only writes, retry-safe copies/previews and billed media run
// under the host claim instead (see dshCanvasEvidenceWrite, dshCanvasRetrySafe,
// dshMediaWrite) and need write permission. Guarded selection/document reads
// use the desktop's explicit no-migration capability and need no write permission.
func dshCanvasRestrictedCatalog(catalog []desktopDesignTool, canWrite bool, capabilities string) []desktopDesignTool {
	var result []desktopDesignTool
	seen := make(map[string]bool)
	for _, tool := range catalog {
		var schema struct {
			Type       string
			Properties map[string]json.RawMessage
		}
		if seen[tool.Name] || !designTool(tool.Name) || json.Unmarshal(tool.Schema, &schema) != nil || schema.Type != "object" {
			continue
		}
		guarded := dshCanvasGuarded(tool.Name) && (!designWriteTool(tool.Name) || canWrite) && dshCanvasHasGuard(capabilities, tool.Name)
		claimed := canWrite && (dshCanvasEvidenceWrite(tool.Name) || dshCanvasRetrySafe(tool.Name) || dshMediaWrite(tool.Name) && !dshCanvasGuarded(tool.Name))
		if !dshCanvasReadOnly(tool.Name) && !guarded && !claimed {
			continue
		}
		if tool.Name == "media_generate_image" || tool.Name == "canvas_create_image" {
			required, optional := "prompt", "parameters"
			tool.Description = "用当前会话的图片模型生成图片并保存到工作区，返回 source_path。生成前可用 media_get_image_model 查询所选模型支持的参数。"
			if tool.Name == "canvas_create_image" {
				required, optional = "source_path", "title"
				tool.Description = "将本会话生成或编辑的图片作为新节点添加到画布。请原样使用 media_generate_image 或 media_edit_image 返回的 source_path，可选填图片标题。"
				if schema.Properties[required] == nil {
					continue
				}
				schema.Properties[required], _ = json.Marshal(map[string]string{"type": "string", "pattern": dshGeneratedImagePattern})
				if schema.Properties[optional] != nil {
					schema.Properties[optional] = json.RawMessage(`{"type":"string","pattern":"\\S","maxLength":160}`)
				}
			}
			if schema.Properties[required] == nil {
				continue
			}
			for name := range schema.Properties {
				if name != required && name != optional {
					delete(schema.Properties, name)
				}
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(tool.Schema, &fields); err != nil {
				continue
			}
			fields["properties"], _ = json.Marshal(schema.Properties)
			fields["additionalProperties"] = json.RawMessage("false")
			tool.Schema, _ = json.Marshal(fields)

		}
		strict, err := dshStrictToolSchema(tool.Schema)
		if err != nil {
			continue
		}
		seen[tool.Name] = true
		tool.Schema = strict
		result = append(result, tool)
	}
	return result
}
