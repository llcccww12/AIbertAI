package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type dshGoalAdmissionKey struct{}

// The installation and journal root must be private, version pinned resources.
// This constructor does not activate the engine; its owner selects that policy.
func newDSHGatewayRuntime(installation dshInstalledRuntime, journalRoot string, gateway *dshGateway, model func(context.Context, acpSession) (selectedModelConfig, error)) (*dshRuntimeConfig, error) {
	if !filepath.IsAbs(journalRoot) || gateway == nil || model == nil || installation.Patch != "" {
		return nil, errors.New("invalid_dsh_gateway_runtime")
	}
	for _, env := range installation.Env {
		key, _, _ := strings.Cut(env, "=")
		if strings.EqualFold(key, "DSH_AIBERTAI_LOCAL_TOKEN") {
			return nil, errors.New("invalid_dsh_gateway_runtime")
		}
	}
	validate := func(ctx context.Context, session acpSession) (selectedModelConfig, error) {
		if !validUUID(session.ID) || session.ID != session.DSHSessionID {
			return selectedModelConfig{}, errors.New("invalid_dsh_gateway_session")
		}
		selected, err := model(ctx, session)
		if err != nil {
			return selectedModelConfig{}, err
		}
		if selected.ID != session.Model {
			return selectedModelConfig{}, errors.New("invalid_dsh_gateway_model")
		}
		if !selected.SupportsVision {
			// 只看这一轮有没有图。历史里有过图就整段拒绝，会让会话永久卡死：
			// 之后每条消息都撞在同一张旧图上，用户只能退出重开。
			if _, ok := ctx.Value(dshImageInputKey{}).(dshImageInput); ok {
				return selectedModelConfig{}, errors.New("model_vision_unavailable")
			}
		}
		return selected, nil
	}
	bind := func(ctx context.Context, session acpSession, operationID string, local *dshLocalGateway) error {
		var admission func(string) error
		if admit, ok := ctx.Value(dshGoalAdmissionKey{}).(func(string) error); ok && admit != nil {
			admission = admit
		} else if session.platformGoal() != nil && session.Goal.Current && session.Goal.Status == "active" {
			return errors.New("dsh_goal_budget_unavailable")
		}
		local.mu.Lock()
		defer local.mu.Unlock()
		if local.pending != nil {
			return errors.New("dsh_reconciliation_required")
		}
		local.admitGoalInvocation = admission
		return nil
	}
	resolvePlugins := func(ctx context.Context, expertID string) ([]string, error) {
		auth, err := gateway.authorization(ctx)
		if err != nil {
			return nil, err
		}
		expert, err := gateway.client.desktopExpertManifest(ctx, auth, expertID)
		if err != nil {
			return nil, err
		}
		return dshDisabledPlugins(expert.DSHPlugins)
	}
	experts := &dshTeamExpertCache{load: func(ctx context.Context) ([]dshTeamExpert, error) {
		auth, err := gateway.authorization(ctx)
		if err != nil {
			return nil, err
		}
		return gateway.client.dshTeamExperts(ctx, auth)
	}}
	return &dshRuntimeConfig{TeamExperts: experts.get, JournalRoot: journalRoot, SharedPluginHome: installation.Home, ResolveDisabledPlugins: resolvePlugins, ValidateLease: validate, BindLease: bind, Prepare: func(ctx context.Context, session acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
		selected, err := validate(ctx, session)
		if err != nil {
			return nil, "", "", nil, nil, err
		}
		if !selected.SupportsVision {
			if _, ok := ctx.Value(dshImageInputKey{}).(dshImageInput); ok {
				return nil, "", "", nil, nil, errors.New("model_vision_unavailable")
			}
		}
		journal := filepath.Join(journalRoot, session.ID)
		local, err := newDSHLocalGateway(gateway, session.ID, selected, journal)
		if err != nil {
			return nil, "", "", nil, nil, err
		}
		local.requireInvocationID = true
		if admit, ok := ctx.Value(dshGoalAdmissionKey{}).(func(string) error); ok && admit != nil {
			local.admitGoalInvocation = admit
		} else if session.platformGoal() != nil && session.Goal.Current && session.Goal.Status == "active" {
			return nil, "", "", nil, nil, errors.New("dsh_goal_budget_unavailable")
		}
		base, stop, err := local.Start()
		if err != nil {
			return nil, "", "", nil, nil, err
		}
		cleanup := func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = stop(stopCtx)
		}
		local.agentTeamEnabled = session.AgentTeamEnabled
		local.standardMode = session.ExpertID == ""
		if session.ExpertID == "" {
			if installation.PlatformSkillsDir != "" {
				local.platformSkillsDir = dshSessionSkillsDir(installation.PlatformSkillsDir, session.ID)
			}
			local.teamExperts = experts.get(ctx)
		}
		disabled, _ := ctx.Value(dshDisabledPluginsKey{}).([]string)
		patch, err := local.Patch(base, selected, disabled...)
		if err != nil {
			cleanup()
			return nil, "", "", nil, nil, err
		}
		file, err := os.CreateTemp(journal, ".sdk-patch-*.json")
		if err != nil {
			cleanup()
			return nil, "", "", nil, nil, err
		}
		path := file.Name()
		if err = file.Chmod(0600); err == nil {
			_, err = file.Write(patch)
		}
		err = errors.Join(err, file.Close())
		if err != nil {
			_ = os.Remove(path)
			cleanup()
			return nil, "", "", nil, nil, err
		}
		stopAll := func() { _ = os.Remove(path); cleanup() }
		copyInstall := installation
		copyInstall.Patch = path
		copyInstall.Provider = func(context.Context, acpSession) (string, string, error) { return "aibertai", selected.ID, nil }
		copyInstall.Env = append(append([]string{}, installation.Env...), "DSH_AIBERTAI_LOCAL_TOKEN="+local.token)
		cfg, err := copyInstall.config()
		if err != nil {
			stopAll()
			return nil, "", "", nil, nil, err
		}
		cmd, err := cfg.Command(ctx, session)
		if err != nil {
			stopAll()
			return nil, "", "", nil, nil, err
		}
		return cmd, "aibertai", selected.ID, local, stopAll, nil
	}}, nil
}
