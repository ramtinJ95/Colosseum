package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeDashboardSessionController struct {
	sessionExists    bool
	currentSession   string
	lastSession      string
	currentErr       error
	lastErr          error
	createErr        error
	switchErr        error
	attachErr        error
	createCalls      []fakeCreateDashboardCall
	switchCalls      []string
	attachCalls      []string
	currentCalls     int
	sessionExistHits []string
}

type fakeCreateDashboardCall struct {
	name     string
	startDir string
	command  []string
}

func (f *fakeDashboardSessionController) SessionExists(_ context.Context, name string) bool {
	f.sessionExistHits = append(f.sessionExistHits, name)
	return f.sessionExists
}

func (f *fakeDashboardSessionController) CurrentSession(_ context.Context) (string, error) {
	f.currentCalls++
	if f.currentErr != nil {
		return "", f.currentErr
	}
	return f.currentSession, nil
}

func (f *fakeDashboardSessionController) LastSession(_ context.Context) (string, error) {
	if f.lastErr != nil {
		return "", f.lastErr
	}
	return f.lastSession, nil
}

func (f *fakeDashboardSessionController) CreateDetachedSessionWithCommand(_ context.Context, name string, startDir string, command []string) error {
	f.createCalls = append(f.createCalls, fakeCreateDashboardCall{
		name:     name,
		startDir: startDir,
		command:  append([]string(nil), command...),
	})
	return f.createErr
}

func (f *fakeDashboardSessionController) SwitchSession(_ context.Context, name string) error {
	f.switchCalls = append(f.switchCalls, name)
	return f.switchErr
}

func (f *fakeDashboardSessionController) AttachSession(_ context.Context, name string) error {
	f.attachCalls = append(f.attachCalls, name)
	return f.attachErr
}

func TestDashboardBootstrapRunsInlineForInternalProcess(t *testing.T) {
	controller := &fakeDashboardSessionController{}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv: func(key string) string {
			if key == dashboardInternalEnv {
				return "1"
			}
			return ""
		},
	}

	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if handled {
		t.Fatal("handled = true, want false")
	}
	if controller.currentCalls != 0 {
		t.Fatalf("currentCalls = %d, want 0", controller.currentCalls)
	}
	if len(controller.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(controller.createCalls))
	}
}

func TestDashboardBootstrapCreatesAndAttachesOutsideTmux(t *testing.T) {
	controller := &fakeDashboardSessionController{}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv:         func(string) string { return "" },
	}

	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if len(controller.createCalls) != 1 {
		t.Fatalf("createCalls = %d, want 1", len(controller.createCalls))
	}
	if len(controller.attachCalls) != 1 || controller.attachCalls[0] != "colo-dashboard" {
		t.Fatalf("attachCalls = %v, want [colo-dashboard]", controller.attachCalls)
	}
	if len(controller.switchCalls) != 0 {
		t.Fatalf("switchCalls = %v, want none", controller.switchCalls)
	}

	expectedCommand := []string{"env", dashboardInternalEnv + "=1", "/tmp/colosseum/bin"}
	if !reflect.DeepEqual(controller.createCalls[0].command, expectedCommand) {
		t.Fatalf("command = %v, want %v", controller.createCalls[0].command, expectedCommand)
	}
}

func TestDashboardBootstrapCreatesAndSwitchesInsideTmux(t *testing.T) {
	controller := &fakeDashboardSessionController{currentSession: "colo-feature"}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv: func(key string) string {
			if key == "TMUX" {
				return "/tmp/tmux-1000/default,123,0"
			}
			return ""
		},
	}

	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if controller.currentCalls != 1 {
		t.Fatalf("currentCalls = %d, want 1", controller.currentCalls)
	}
	if len(controller.createCalls) != 1 {
		t.Fatalf("createCalls = %d, want 1", len(controller.createCalls))
	}
	if len(controller.switchCalls) != 1 || controller.switchCalls[0] != "colo-dashboard" {
		t.Fatalf("switchCalls = %v, want [colo-dashboard]", controller.switchCalls)
	}
	if len(controller.attachCalls) != 0 {
		t.Fatalf("attachCalls = %v, want none", controller.attachCalls)
	}

	expectedCommand := []string{"env", dashboardInternalEnv + "=1", "/tmp/colosseum/bin"}
	if !reflect.DeepEqual(controller.createCalls[0].command, expectedCommand) {
		t.Fatalf("command = %v, want %v", controller.createCalls[0].command, expectedCommand)
	}
}

func TestDashboardBootstrapReusesExistingSessionInsideTmux(t *testing.T) {
	controller := &fakeDashboardSessionController{
		sessionExists:  true,
		currentSession: "colo-feature",
	}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv: func(key string) string {
			if key == "TMUX" {
				return "/tmp/tmux-1000/default,123,0"
			}
			return ""
		},
	}

	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if len(controller.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(controller.createCalls))
	}
	if len(controller.switchCalls) != 1 || controller.switchCalls[0] != "colo-dashboard" {
		t.Fatalf("switchCalls = %v, want [colo-dashboard]", controller.switchCalls)
	}
}

func TestDashboardBootstrapRunsInlineInsideDashboardSession(t *testing.T) {
	controller := &fakeDashboardSessionController{currentSession: "colo-dashboard"}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv: func(key string) string {
			if key == "TMUX" {
				return "/tmp/tmux-1000/default,123,0"
			}
			return ""
		},
	}

	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if handled {
		t.Fatal("handled = true, want false")
	}
	if controller.currentCalls != 1 {
		t.Fatalf("currentCalls = %d, want 1", controller.currentCalls)
	}
	if len(controller.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(controller.createCalls))
	}
}

func TestDashboardBootstrapSurfacesCurrentSessionError(t *testing.T) {
	controller := &fakeDashboardSessionController{currentErr: errors.New("no client")}
	bootstrap := dashboardBootstrap{
		client:         controller,
		sessionName:    "colo-dashboard",
		currentDir:     "/tmp/colosseum",
		executablePath: "/tmp/colosseum/bin",
		getenv: func(key string) string {
			if key == "TMUX" {
				return "/tmp/tmux-1000/default,123,0"
			}
			return ""
		},
	}

	if _, err := bootstrap.Bootstrap(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreDashboardSessionSwitchesBackToOrigin(t *testing.T) {
	controller := &fakeDashboardSessionController{
		sessionExists: true,
		lastSession:   "colo-feature",
	}

	err := restoreDashboardSession(context.Background(), controller, func(key string) string {
		if key == "TMUX" {
			return "/tmp/tmux-1000/default,123,0"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("restoreDashboardSession: %v", err)
	}
	if len(controller.switchCalls) != 1 || controller.switchCalls[0] != "colo-feature" {
		t.Fatalf("switchCalls = %v, want [colo-feature]", controller.switchCalls)
	}
}

func TestRestoreDashboardSessionSkipsWhenOriginSessionMissing(t *testing.T) {
	controller := &fakeDashboardSessionController{lastSession: "colo-feature"}

	err := restoreDashboardSession(context.Background(), controller, func(key string) string {
		if key == "TMUX" {
			return "/tmp/tmux-1000/default,123,0"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("restoreDashboardSession: %v", err)
	}
	if len(controller.switchCalls) != 0 {
		t.Fatalf("switchCalls = %v, want none", controller.switchCalls)
	}
}

func TestRestoreDashboardSessionSkipsWithoutLastSession(t *testing.T) {
	controller := &fakeDashboardSessionController{}

	err := restoreDashboardSession(context.Background(), controller, func(key string) string {
		if key == "TMUX" {
			return "/tmp/tmux-1000/default,123,0"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("restoreDashboardSession: %v", err)
	}
	if len(controller.switchCalls) != 0 {
		t.Fatalf("switchCalls = %v, want none", controller.switchCalls)
	}
}
