package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"unicode/utf8"
)

func dshCanvasResultDigest(output string) string {
	sum := sha256.Sum256([]byte(output))
	return fmt.Sprintf("v1:%d:%x", len(output), sum)
}

func dshCanvasMarkClaim(effect *dshSideEffect, before string) error {
	if effect == nil {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
	if err != nil || claim.State != "dispatch_claimed" || claim.Kind != "" {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim.Kind, claim.CanvasBefore = "canvas", &before
	if err = writeConnectorClaim(effect.root, claim); err != nil {
		return errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return nil
}

// The existing desktop design bridge only accepts results up to 48 KiB.
// Persist a digest and byte length; the SDK history retains the exact result.
func dshCanvasRecordResult(effect *dshSideEffect, output string) error {
	if effect == nil || len(output) > 48<<10 || !utf8.ValidString(output) {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
	if err != nil || claim.State != "dispatch_claimed" || claim.Kind != "canvas" {
		return errors.New("dsh_connector_outcome_unknown")
	}
	digest := dshCanvasResultDigest(output)
	claim.Kind, claim.State, claim.Output = "canvas", "result_ready", &digest
	if err := writeConnectorClaim(effect.root, claim); err != nil {
		return errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return nil
}
