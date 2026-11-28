package main

import (
	"context"
	"strings"
)

// Route plan mode, goal inspection and manual compaction through the SDK's
// command dispatcher. Goal mutations remain owned by the platform goal flow.
func runDSHInteractiveTurn(ctx context.Context, cfg *dshRuntimeConfig, session acpSession, turn *acpTurn, resume bool, project func(map[string]any) error) error {
	if turn.DSHMemberID != "" {
		return runDSHTurn(ctx, cfg, session, turn, resume, project)
	}
	command := strings.TrimSpace(turn.Prompt)
	planCommand := command == "/plan" || strings.HasPrefix(command, "/plan ")
	opts := dshSDKOptions{
		AllowPreviousFailure: cfg != nil && cfg.ContinueAfterFailedTurn,
		AllowMissingSession:  resume && session.canStartMissingDSHSession(turn.Operation),
	}
	if turn.ModelInput == "" {
		switch {
		case command == "/compact":
			opts = dshCompactOptions(session, turn, project)
		case command == "/goal":
			opts = dshSDKOptions{Query: true}
		case planCommand:
			opts.Query = true
		}
	}
	return runDSHTurn(context.WithValue(ctx, dshSDKOptionsKey{}, opts), cfg, session, turn, resume, project)
}

// Older desktop versions saved pre-dispatch failures as if the SDK had seen
// them. Allow first creation only if every earlier turn failed without any
// recorded SDK work, and only after the SDK confirms the session is absent.
func (s acpSession) canStartMissingDSHSession(operation string) bool {
	if s.Goal != nil || len(s.Turns) < 2 || s.Turns[len(s.Turns)-1].Operation != operation {
		return false
	}
	for _, prior := range s.Turns[:len(s.Turns)-1] {
		if prior.DSHInputNotAdmitted {
			continue
		}
		switch prior.FailureCode {
		case "session_turn_failed", "dsh_runtime_exited", "backend_response_interrupted", "backend_request_timeout":
		default:
			return false
		}
		if !prior.Failed ||
			prior.DSHMemberID != "" || prior.hasHistoryResponse() || prior.DSHStreamed ||
			prior.DSHProjectionSeq != nil || prior.Usage != (modelUsage{}) || prior.Tokens != (turnTokenFacts{}) {
			return false
		}
	}
	return true
}

func (s acpSession) addDSHPlanMetadata(metadata map[string]any) {
	if s.Engine != "dsh" {
		return
	}
	for i := len(s.Turns) - 1; i >= 0; i-- {
		if active := s.Turns[i].DSHPlanActive; active != nil {
			metadata["dshPlanActive"] = *active
			return
		}
	}
}
