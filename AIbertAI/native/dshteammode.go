package main

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
)

func (expert desktopExpert) allowsAgentTeam() bool {
	return expert.DSHPlugins == nil || slices.Contains(expert.DSHPlugins, "tool-agent-team")
}

type desktopAgentTeamMode struct {
	Enabled   bool `json:"agentTeamEnabled"`
	Available bool `json:"agentTeamAvailable"`
}

func (s acpSession) addAgentTeamMetadata(metadata map[string]any) {
	if s.Engine != "dsh" {
		return
	}
	metadata["dshJobsAvailable"] = true
	if s.AgentTeamAvailable != nil {
		metadata["agentTeamAvailable"] = *s.AgentTeamAvailable
	}
	if s.AgentTeamEnabled != nil {
		metadata["agentTeamEnabled"] = *s.AgentTeamEnabled && (s.AgentTeamAvailable == nil || *s.AgentTeamAvailable)
	}
}

func (h *relaySessionHost) sessionAgentTeam(ctx context.Context, raw json.RawMessage) (any, error) {
	var args struct {
		SessionID string  `json:"sessionId"`
		ExpertID  *string `json:"expertId"`
		Enabled   *bool   `json:"enabled"`
	}
	if json.Unmarshal(raw, &args) != nil {
		return nil, errors.New("invalid_session_parameters")
	}
	if args.ExpertID != nil {
		if args.SessionID != "" || args.Enabled != nil {
			return nil, errors.New("invalid_session_parameters")
		}
		expert, err := h.resolveDesktopExpert(ctx, *args.ExpertID)
		if err != nil {
			return nil, err
		}
		return desktopAgentTeamMode{Available: expert.allowsAgentTeam()}, nil
	}
	l := h.live
	row, err := l.load(args.SessionID)
	if err != nil {
		return nil, err
	}
	if row.Engine != "dsh" || row.DSHSessionID != row.ID {
		if args.Enabled == nil {
			return map[string]any{}, nil
		}
		return nil, errors.New("agent_team_unavailable")
	}
	expert, err := h.resolveDesktopExpert(ctx, row.ExpertID)
	if err != nil {
		return nil, err
	}
	mode := desktopAgentTeamMode{Available: expert.allowsAgentTeam()}
	if !mode.Available {
		if args.Enabled != nil && *args.Enabled {
			return nil, errors.New("agent_team_unavailable")
		}
		return mode, nil
	}
	l.mu.Lock()
	current, exists := l.sessions[row.ID]
	if !exists || current.ExpertID != row.ExpertID || current.ResourceRevision != row.ResourceRevision {
		l.mu.Unlock()
		return nil, errors.New("configuration_conflict")
	}
	row = current
	if l.restarting || l.compacting[row.ID] || row.State == "running" || l.activeTurns[row.ID] != nil || desktopGoalAdvancing(row.Goal) {
		l.mu.Unlock()
		if args.Enabled == nil && row.AgentTeamEnabled != nil {
			mode.Enabled = *row.AgentTeamEnabled
			return mode, nil
		}
		return nil, errors.New("session_busy")
	}
	if len(row.Turns) == 0 {
		// No SDK journal exists until the first prompt. Keep the previous
		// employee default for clients that omitted a creation preference.
		mode.Enabled = row.AgentTeamEnabled == nil || *row.AgentTeamEnabled
		if args.Enabled != nil {
			mode.Enabled = *args.Enabled
			row.AgentTeamEnabled, row.AgentTeamAvailable = &mode.Enabled, &mode.Available
			err = l.save(row)
		}
		l.mu.Unlock()
		return mode, err
	}
	if l.compacting == nil {
		l.compacting = make(map[string]bool)
	}
	l.compacting[row.ID] = true
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		delete(l.compacting, row.ID)
		l.mu.Unlock()
	}()

	disabled, err := dshDisabledPlugins(expert.DSHPlugins)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, dshDisabledPluginsKey{}, disabled)
	opts := dshSDKOptions{InspectSession: func(ctx context.Context, p *dshProcess) error {
		params := map[string]any{"sessionId": row.ID}
		if args.Enabled != nil {
			params["enabled"] = *args.Enabled
		}
		reply, err := p.call(ctx, "aibertai/agent-team", params, nil)
		if err != nil {
			return err
		}
		var state struct{ Enabled *bool }
		if json.Unmarshal(reply, &state) != nil || state.Enabled == nil || args.Enabled != nil && *state.Enabled != *args.Enabled {
			return errors.New("invalid_agent_team_response")
		}
		mode.Enabled = *state.Enabled
		return nil
	}}
	if err := reconcileDSHSession(context.WithValue(ctx, dshSDKOptionsKey{}, opts), l.dsh, row); err != nil {
		return nil, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	current, exists = l.sessions[row.ID]
	if !exists || current.ExpertID != row.ExpertID {
		return nil, errors.New("configuration_conflict")
	}
	if current.AgentTeamEnabled == nil || *current.AgentTeamEnabled != mode.Enabled || current.AgentTeamAvailable == nil || !*current.AgentTeamAvailable {
		current.AgentTeamEnabled, current.AgentTeamAvailable = &mode.Enabled, &mode.Available
		if err := l.save(current); err != nil {
			return nil, err
		}
	}
	return mode, nil
}
