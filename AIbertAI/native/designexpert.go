package main

import (
	"embed"
	"encoding/json"
	"strings"
)

const designExpertID = "monkeyai-design" // Existing desktop profile and saved session identity.

//go:embed design-expert/*.md
var designExpertFiles embed.FS

func designExpertInstructions(design json.RawMessage) string {
	var task struct {
		CanvasID            string
		DesignType          string
		InitialArtifactType string
	}
	if json.Unmarshal(design, &task) != nil || !validUUID(task.CanvasID) {
		return ""
	}
	core, _ := designExpertFiles.ReadFile("design-expert/core.md")
	var medium string
	switch task.DesignType {
	case "prototype":
		medium = "prototype"
	case "slide_deck":
		medium = "deck"
	case "image":
		medium = "image"
	case "video":
		medium = "video"
	}
	if medium == "" && (task.InitialArtifactType == "poster" || task.InitialArtifactType == "logo") {
		medium = "image"
	}
	var instructions strings.Builder
	instructions.Write(core)
	if medium != "" {
		note, _ := designExpertFiles.ReadFile("design-expert/" + medium + ".md")
		instructions.WriteByte('\n')
		instructions.Write(note)
	}
	instructions.WriteString("\nRead the current selections from this workspace-relative path when needed: .aibertai/design/")
	instructions.WriteString(task.CanvasID)
	instructions.WriteString("/design-context.json")
	return instructions.String()
}
