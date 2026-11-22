package main

import (
	"encoding/json"
	"errors"
	"strings"
)

// dshStrictToolSchema preserves validation while making required-only union
// branches acceptable to strict JSON Schema validators. Unsafe or unknown
// compositions are refused rather than weakening validation.
func dshStrictToolSchema(raw json.RawMessage) (json.RawMessage, error) {
	var root map[string]any
	if len(raw) > 48<<10 || json.Unmarshal(raw, &root) != nil || root == nil {
		return nil, errors.New("dsh_tool_schema_invalid")
	}
	if err := dshExpandLocalSchemaRefs(root); err != nil {
		return nil, err
	}
	var visit func(map[string]any, map[string]any) error
	visit = func(schema map[string]any, inherited map[string]any) error {
		if _, ok := schema["unevaluatedProperties"]; ok {
			return errors.New("dsh_tool_schema_unsupported")
		}
		if _, ok := schema["$ref"]; ok {
			return errors.New("dsh_tool_schema_unsupported")
		}
		local := map[string]any{}
		if p, exists := schema["properties"]; exists {
			var ok bool
			local, ok = p.(map[string]any)
			if !ok {
				return errors.New("dsh_tool_schema_invalid")
			}
		}
		if value, exists := schema["required"]; exists {
			names, ok := value.([]any)
			if !ok {
				return errors.New("dsh_tool_schema_invalid")
			}
			for _, item := range names {
				name, ok := item.(string)
				if !ok {
					return errors.New("dsh_tool_schema_invalid")
				}
				if _, exists := local[name]; exists {
					continue
				}
				if _, exists := inherited[name]; !exists {
					return errors.New("dsh_tool_schema_unsupported")
				}
				// An additionalProperties rule on this exact branch would change meaning
				// if a new property were introduced there.
				if _, exists := schema["additionalProperties"]; exists {
					return errors.New("dsh_tool_schema_unsupported")
				}
				local[name] = map[string]any{}
				schema["properties"] = local
			}
		}
		merged := map[string]any{}
		for key, value := range inherited {
			merged[key] = value
		}
		for key, value := range local {
			merged[key] = value
		}
		for _, kind := range []string{"anyOf", "oneOf", "allOf"} {
			if rawBranches, exists := schema[kind]; exists {
				branches, ok := rawBranches.([]any)
				if !ok {
					return errors.New("dsh_tool_schema_invalid")
				}
				for _, rawBranch := range branches {
					branch, ok := rawBranch.(map[string]any)
					if !ok {
						return errors.New("dsh_tool_schema_unsupported")
					}
					if err := visit(branch, merged); err != nil {
						return err
					}
				}
			}
		}
		for _, rawProperty := range local {
			if child, ok := rawProperty.(map[string]any); ok {
				if err := visit(child, nil); err != nil {
					return err
				}
			}
		}
		if rawItems, exists := schema["items"]; exists {
			child, ok := rawItems.(map[string]any)
			if !ok {
				return errors.New("dsh_tool_schema_unsupported")
			}
			if err := visit(child, nil); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root, nil); err != nil {
		return nil, err
	}
	out, err := json.Marshal(root)
	if err != nil || len(out) > 48<<10 {
		return nil, errors.New("dsh_tool_schema_invalid")
	}
	return out, nil
}

// dshExpandLocalSchemaRefs accepts only acyclic local definition references.
// Ref siblings and unsupported locations are rejected to avoid changing meaning.
func dshExpandLocalSchemaRefs(root map[string]any) error {
	unsupported := errors.New("dsh_tool_schema_unsupported")
	definitions := map[string]map[string]any{}
	for _, key := range []string{"$defs", "definitions"} {
		if value, ok := root[key]; ok {
			defs, ok := value.(map[string]any)
			if !ok {
				return unsupported
			}
			definitions[key] = defs
		}
	}
	count := 0
	active := map[string]bool{}
	usedRef := false
	unsafeBase := false
	var expand func(map[string]any, int) error
	expand = func(schema map[string]any, depth int) error {
		count++
		for _, key := range []string{"$id", "id", "$anchor", "$dynamicAnchor", "$dynamicRef", "$recursiveRef"} {
			if _, ok := schema[key]; ok {
				unsafeBase = true
			}
		}
		if depth > 32 || count > 2048 {
			return unsupported
		}
		if ref, exists := schema["$ref"]; exists {
			usedRef = true
			path, ok := ref.(string)
			parts := strings.Split(path, "/")
			if !ok || len(schema) != 1 || len(parts) != 3 || parts[0] != "#" || (parts[1] != "$defs" && parts[1] != "definitions") {
				return unsupported
			}
			name := strings.ReplaceAll(strings.ReplaceAll(parts[2], "~1", "/"), "~0", "~")
			if strings.Contains(strings.ReplaceAll(strings.ReplaceAll(parts[2], "~1", ""), "~0", ""), "~") {
				return unsupported
			}
			target, ok := definitions[parts[1]][name].(map[string]any)
			if !ok || active[path] {
				return unsupported
			}
			data, err := json.Marshal(target)
			if err != nil || len(data) > 48<<10 {
				return unsupported
			}
			var copy map[string]any
			if json.Unmarshal(data, &copy) != nil {
				return unsupported
			}
			active[path] = true
			err = expand(copy, depth+1)
			delete(active, path)
			if err != nil {
				return err
			}
			delete(schema, "$ref")
			for key, value := range copy {
				schema[key] = value
			}
			return nil
		}
		for _, key := range []string{"properties", "patternProperties"} {
			if children, ok := schema[key].(map[string]any); ok {
				for _, value := range children {
					if child, ok := value.(map[string]any); ok {
						if err := expand(child, depth+1); err != nil {
							return err
						}
					}
				}
			}
		}
		for _, key := range []string{"items", "additionalProperties", "not", "if", "then", "else", "contains"} {
			if child, ok := schema[key].(map[string]any); ok {
				if err := expand(child, depth+1); err != nil {
					return err
				}
			}
		}
		for _, key := range []string{"allOf", "anyOf", "oneOf"} {
			if children, ok := schema[key].([]any); ok {
				for _, value := range children {
					if child, ok := value.(map[string]any); ok {
						if err := expand(child, depth+1); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}
	if err := expand(root, 0); err != nil {
		return err
	}
	if usedRef && unsafeBase {
		return unsupported
	}
	delete(root, "$defs")
	delete(root, "definitions")
	var rejectRefs func(any) error
	rejectRefs = func(value any) error {
		switch value := value.(type) {
		case map[string]any:
			if _, ok := value["$ref"]; ok {
				return unsupported
			}
			for _, child := range value {
				if err := rejectRefs(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range value {
				if err := rejectRefs(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return rejectRefs(root)
}
