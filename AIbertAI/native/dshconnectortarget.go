package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"
)

// No credential identifier, bearer token, URL, or payload is retained. The
// endpoint digest includes the actual credential route without disclosing it.
type dshConnectorTarget struct {
	Version          int    `json:"version"`
	ConnectorID      string `json:"connectorId"`
	Tool             string `json:"tool"`
	EndpointDigest   string `json:"endpointDigest"`
	ResourceRevision int    `json:"resourceRevision"`
	Step             int    `json:"step"`
}

func validDSHConnectorTarget(target dshConnectorTarget) bool {
	if target.Version != 1 || !validUUID(target.ConnectorID) || target.Tool == "" || len(target.Tool) > 1024 || !utf8.ValidString(target.Tool) || strings.ContainsRune(target.Tool, 0) || target.ResourceRevision < 0 || target.Step < 0 || len(target.EndpointDigest) != 64 {
		return false
	}
	bytes, err := hex.DecodeString(target.EndpointDigest)
	return err == nil && len(bytes) == sha256.Size && hex.EncodeToString(bytes) == target.EndpointDigest
}

func newDSHConnectorTarget(origin string, tool connectorTool, revision, step int) (dshConnectorTarget, error) {
	fail := func() (dshConnectorTarget, error) {
		return dshConnectorTarget{}, errors.New("dsh_connector_target_invalid")
	}
	if tool.Local || !validUUID(tool.ConnectorID) || tool.CredentialID != "" && !validUUID(tool.CredentialID) {
		return fail()
	}
	u, err := url.Parse(origin)
	if err != nil || u.User != nil || u.Hostname() == "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Scheme != "http" && u.Scheme != "https" {
		return fail()
	}
	host, port := strings.ToLower(u.Hostname()), u.Port()
	if port != "" && !(u.Scheme == "http" && port == "80") && !(u.Scheme == "https" && port == "443") {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	endpoint := "POST " + u.Scheme + "://" + host + connectorPath("/mcp/connectors/", tool.ConnectorID, tool.CredentialID)
	hash := sha256.Sum256([]byte(endpoint))
	target := dshConnectorTarget{Version: 1, ConnectorID: tool.ConnectorID, Tool: tool.Name, EndpointDigest: hex.EncodeToString(hash[:]), ResourceRevision: revision, Step: step}
	if !validDSHConnectorTarget(target) {
		return fail()
	}
	return target, nil
}

// Persist identity after approval and before any remote request is made.
func (effect *dshSideEffect) bindConnectorTarget(target dshConnectorTarget) error {
	if effect == nil || !validDSHConnectorTarget(target) {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim, err := connectorPending(effect.root, effect.session, effect.operation, effect.call)
	if err != nil || claim.State != "dispatch_claimed" || claim.Target != nil {
		return errors.New("dsh_connector_outcome_unknown")
	}
	claim.Target = &target
	if err = writeConnectorClaim(effect.root, claim); err != nil {
		return errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
	}
	return nil
}
