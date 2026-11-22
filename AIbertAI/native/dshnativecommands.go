package main

import (
	"context"
	"strings"
)

// Route read-only goal inspection and manual compaction through the SDK's
// command dispatcher. Goal mutations remain owned by the platform goal flow.
func runDSHInteractiveTurn(ctx context.Context, cfg *dshRuntimeConfig, session acpSession, turn *acpTurn, resume bool, project func(map[string]any) error) error {
	if turn.DSHMemberID != "" {
		return runDSHTurn(ctx, cfg, session, turn, resume, project)
	}
	command := strings.TrimSpace(turn.Prompt)
	if turn.ModelInput == "" && command == "/compact" {
		ctx = context.WithValue(ctx, dshSDKOptionsKey{}, dshCompactOptions(session, turn, project))
	} else if turn.ModelInput == "" && command == "/goal" {
		ctx = context.WithValue(ctx, dshSDKOptionsKey{}, dshSDKOptions{Query: true})
	}
	return runDSHTurn(ctx, cfg, session, turn, resume, project)
}
