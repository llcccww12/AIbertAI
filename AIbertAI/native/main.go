package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validUUID(value string) bool { return uuidPattern.MatchString(value) }

func verify(ctx context.Context, reader io.Reader, emit func(string, string) error) (err error) {
	if !validUUID(modelID) {
		return errors.New("打包时必须指定私有后端中唯一的模型 UUID")
	}
	c, err := newClient(backendOrigin)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(reader, (16<<10)+1))
	if err != nil {
		return err
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" || len(data) > 16<<10 {
		return errors.New("请输入 1 到 16384 字节的验收提示词")
	}

	loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	auth, user, err := c.login(loginCtx, func(link string) error { return emit("authorize", link) })
	cancel()
	if auth.Refresh != "" {
		defer func() {
			cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
			defer done()
			err = errors.Join(err, c.revoke(cleanup, auth))
		}()
	}
	if err != nil {
		return err
	}
	if err := emit("login", "授权成功"); err != nil {
		return err
	}
	machine, err := c.bindTestDevice(ctx, auth, user)
	if err != nil {
		return err
	}
	model, err := c.selectedModelConfig(ctx, auth)
	if err != nil {
		return err
	}
	key, err := c.createModelKey(ctx, auth)
	if validUUID(key.ID) {
		defer func() {
			cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
			defer done()
			err = errors.Join(err, c.json(cleanup, "DELETE", "/api/v1/api-keys/"+key.ID, auth.Access, "", nil, nil))
		}()
	}
	if err != nil {
		return err
	}

	session, err := newUUID()
	if err != nil {
		return err
	}
	started := time.Now().UTC()
	if err := c.registerSession(ctx, auth, machine, session, modelID, "", started); err != nil {
		return err
	}
	usage, finish, modelErr := c.chat(ctx, model.Protocol, key.Value, session, modelID, prompt, model.StreamMode != "off", func(chunk string) error { return emit("delta", chunk) })
	stop := "complete"
	switch {
	case modelErr != nil:
		stop = "error"
	case finish == "length":
		stop = "output_limit"
	}
	if err := c.report(ctx, auth, machine, session, started, time.Now().UTC(), stop, usage); err != nil {
		return errors.Join(err, modelErr)
	}
	if modelErr != nil {
		return modelErr
	}
	return emit("reported", session)
}

func verifyKnowledge(ctx context.Context, emit func(string, string) error) (err error) {
	c, err := newClient(backendOrigin)
	if err != nil {
		return err
	}
	loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	auth, _, err := c.login(loginCtx, func(link string) error { return emit("authorize", link) })
	cancel()
	if auth.Refresh != "" {
		defer func() {
			cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
			defer stop()
			err = errors.Join(err, c.revoke(cleanup, auth))
		}()
	}
	if err != nil {
		return err
	}
	catalog, err := c.knowledgeBases(ctx, auth, knowledgeFilter{Owner: "all", Page: 1, Size: 1})
	if err != nil {
		return err
	}
	return emit("knowledge", fmt.Sprintf("%d", catalog.TotalCount))
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "rollback-workspace-brand" {
		if err := rollbackWorkspaceBrand(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "status" || os.Args[1] == "login" || os.Args[1] == "serve") {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := cloudCommand(ctx, os.Args[1:], os.Stdout)
		stop()
		if err != nil {
			emitCloudError(os.Stdout, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "prepare-cloud-migration" {
		if err := prepareCloudMigration(os.Args[2:], os.Stdout); err != nil {
			emitCloudError(os.Stdout, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "verify-office-preview" {
		if err := verifyOfficePreview(os.Stdin, os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "verify-editable-pptx" {
		slides, _ := strconv.Atoi(os.Args[2])
		if err := verifyEditableDeck(os.Stdin, slides); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 3 && os.Args[1] == "desktop-source-login" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := desktopSourceLogin(ctx, os.Args[2], os.Stdin, os.Stdout)
		stop()
		if err != nil {
			reason := "login_failed"
			switch {
			case errors.Is(err, errSourceLoginCancelled), errors.Is(err, context.Canceled):
				reason = "cancelled"
			case errors.Is(err, context.DeadlineExceeded):
				reason = "timeout"
			case errors.Is(err, errDesktopCredential):
				reason = errDesktopCredential.Error()
			}
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"event": "error", "reason": reason})
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 5 && os.Args[1] == "desktop-source-logout" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := desktopSourceLogout(ctx, os.Args[2], os.Args[3], os.Args[4], os.Stdin)
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 4 && os.Args[1] == "desktop-source-runtime" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := desktopRuntime(ctx, os.Args[2], os.Args[3], os.Stdin, os.Stdout)
		stop()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 4 && os.Args[1] == "read-session-archive" {
		archive, err := readLegacySessionArchive(os.Args[2], os.Args[3])
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(archive)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "desktop-file-transfer" {
		if streamFileTransfer(context.Background(), os.Stdin, os.Stdout) != nil {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "desktop-knowledge" {
		writeDesktopKnowledge(context.Background(), os.Stdin, os.Stdout)
		return
	}
	remotePreview := len(os.Args) == 5 && os.Args[1] == "preview-remote-runtime"
	remoteVerify := len(os.Args) == 5 && (os.Args[1] == "verify-remote-runtime" || remotePreview)
	if !remoteVerify && (len(os.Args) != 2 || (os.Args[1] != "verify" && os.Args[1] != "verify-knowledge")) {
		fmt.Fprintln(os.Stderr, "Usage: aibertai-native verify | verify-knowledge | {verify-remote-runtime|preview-remote-runtime} PROFILE SESSION_ID APPROVED_FILE")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	events := json.NewEncoder(os.Stdout)
	emit := func(event, value string) error {
		return events.Encode(map[string]string{"event": event, "value": value})
	}
	var err error
	if remoteVerify {
		err = verifyRemoteRuntime(ctx, os.Args[2], os.Args[3], os.Args[4], remotePreview, emit)
	} else if os.Args[1] == "verify-knowledge" {
		err = verifyKnowledge(ctx, emit)
	} else {
		err = verify(ctx, os.Stdin, emit)
	}
	if err != nil {
		_ = events.Encode(map[string]string{"event": "error", "value": err.Error()})
		os.Exit(1)
	}
}
