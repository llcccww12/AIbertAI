package main

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

var skillCommandName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var errSkillCommandUnavailable = errors.New("skill_command_unavailable")

type desktopSkillPackage struct {
	ID, Name, SHA string
}

func (h *relaySessionHost) desktopSkillInstructions(ctx context.Context, prompt string) (string, string, *desktopSkillPackage, error) {
	commandLine := strings.TrimLeftFunc(prompt, unicode.IsSpace)
	if !strings.HasPrefix(commandLine, "/") {
		return prompt, "", nil, nil
	}
	end := strings.IndexFunc(commandLine, unicode.IsSpace)
	if end < 0 {
		end = len(commandLine)
	}
	command := commandLine[1:end]
	if !skillCommandName.MatchString(command) || strings.EqualFold(command, "goal") {
		return prompt, "", nil, nil
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return "", "", nil, err
	}
	skills, err := h.live.client.desktopSkillCatalog(ctx, auth)
	if err != nil {
		return "", "", nil, err
	}
	var selected *desktopSkillPackage
	for _, skill := range skills {
		name, _ := skill["command_name"].(string)
		if !strings.EqualFold(command, name) {
			continue
		}
		if selected != nil {
			return "", "", nil, errSkillCommandUnavailable
		}
		sha, _ := skill["package_sha256"].(string)
		selected = &desktopSkillPackage{ID: skill["id"].(string), Name: name, SHA: sha}
	}
	if selected == nil {
		return "", "", nil, errSkillCommandUnavailable
	}
	var skill struct {
		Name, Content string
		SHA           string `json:"package_sha256"`
		FileCount     int    `json:"file_count"`
	}
	if err := h.live.client.json(ctx, http.MethodGet, "/api/v1/skills/"+selected.ID+"/content", auth.Access, "", nil, &skill); err != nil {
		return "", "", nil, errSkillCommandUnavailable
	}
	if !strings.EqualFold(skill.Name, command) || strings.TrimSpace(skill.Content) == "" || len(skill.Content) > 128<<10 {
		return "", "", nil, errSkillCommandUnavailable
	}
	task := strings.TrimSpace(commandLine[end:])
	if task == "" {
		task = "Use the selected skill. Ask for any required input that is missing."
	}
	instructions := "The user explicitly selected the skill " + skill.Name + " for this turn. Apply these skill instructions to their request:\n\n" + skill.Content
	if skill.FileCount > 1 {
		if !skillPackageSHA.MatchString(selected.SHA) || skill.SHA != selected.SHA {
			return "", "", nil, errSkillCommandUnavailable
		}
		instructions += "\n\nIf a packaged script is needed, call run_skill_script with the package-relative path and individual arguments. The working directory is the session workspace. Do not report execution unless the tool result confirms it."
	} else {
		selected = nil
	}
	return task, instructions, selected, nil
}
