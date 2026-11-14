package ipc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/seamanticus/aerc/lib/xdg"
)

// fakeHandler stands in for the aerc main loop: functions passed to
// Schedule only run when the test calls runQueued.
type fakeHandler struct {
	err      error
	queue    chan func()
	reported chan error
	commands chan []string
}

func newFakeHandler(err error) *fakeHandler {
	return &fakeHandler{
		err:      err,
		queue:    make(chan func(), 10),
		reported: make(chan error, 10),
		commands: make(chan []string, 10),
	}
}

func (h *fakeHandler) Command(args []string) error {
	h.commands <- args
	return h.err
}

func (h *fakeHandler) Schedule(fn func()) { h.queue <- fn }

func (h *fakeHandler) ReportError(err error) { h.reported <- err }

// runQueued runs the next scheduled function, as the main loop would.
func (h *fakeHandler) runQueued(t *testing.T) {
	t.Helper()
	select {
	case fn := <-h.queue:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("no command was scheduled")
	}
}

// startTestServer serves h on aerc.sock in a temporary XDG_RUNTIME_DIR, which
// is also where ConnectAndExec looks for it.
func startTestServer(t *testing.T, h Handler, pickup time.Duration) {
	t.Helper()
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv(NoWaitEnv, "")
	l, err := net.Listen("unix", xdg.RuntimePath("aerc.sock"))
	if err != nil {
		t.Fatal(err)
	}
	startup, ready := context.WithCancel(context.Background())
	ready()
	as := &AercServer{listener: l, handler: h, startup: startup, pickup: pickup}
	go as.Serve()
	t.Cleanup(as.Close)
}

type execResult struct {
	resp *Response
	err  error
}

func execAsync(args []string) <-chan execResult {
	ch := make(chan execResult, 1)
	go func() {
		resp, err := ConnectAndExec(args)
		ch <- execResult{resp, err}
	}()
	return ch
}

func waitResult(t *testing.T, ch <-chan execResult) *Response {
	t.Helper()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("ConnectAndExec: %v", r.err)
		}
		return r.resp
	case <-time.After(10 * time.Second):
		t.Fatal("no IPC response")
		return nil
	}
}

func assertNotReported(t *testing.T, h *fakeHandler) {
	t.Helper()
	select {
	case err := <-h.reported:
		t.Errorf("unexpected ReportError(%v)", err)
	default:
	}
}

func TestServerReturnsCommandError(t *testing.T) {
	h := newFakeHandler(errors.New("boom"))
	startTestServer(t, h, time.Hour)

	res := execAsync([]string{":foo"})
	h.runQueued(t)
	resp := waitResult(t, res)

	if resp.Error != "boom" || resp.Queued {
		t.Errorf("response = %+v, want error \"boom\", not queued", resp)
	}
	if args := <-h.commands; len(args) != 1 || args[0] != ":foo" {
		t.Errorf("Command(%q), want [\":foo\"]", args)
	}
	assertNotReported(t, h)
}

func TestServerReturnsSuccess(t *testing.T) {
	h := newFakeHandler(nil)
	startTestServer(t, h, time.Hour)

	res := execAsync([]string{":foo"})
	h.runQueued(t)
	resp := waitResult(t, res)

	if resp.Error != "" || resp.Queued {
		t.Errorf("response = %+v, want success", resp)
	}
	assertNotReported(t, h)
}

// A client started by aerc must not wait: the main loop may be blocked on it.
func TestServerAsyncClientDoesNotWait(t *testing.T) {
	h := newFakeHandler(errors.New("boom"))
	startTestServer(t, h, time.Hour)
	t.Setenv(NoWaitEnv, "1")

	// the main loop is "blocked": nothing is run until the reply arrives
	resp := waitResult(t, execAsync([]string{":foo"}))
	if !resp.Queued || resp.Error != "" {
		t.Errorf("response = %+v, want queued", resp)
	}

	h.runQueued(t)
	if err := <-h.reported; err == nil || err.Error() != "boom" {
		t.Errorf("ReportError(%v), want boom", err)
	}
}

// A waiting client whose command is not picked up in time gets a Queued
// response; the error is then reported by the handler.
func TestServerPickupTimeout(t *testing.T) {
	h := newFakeHandler(errors.New("boom"))
	startTestServer(t, h, time.Millisecond)

	resp := waitResult(t, execAsync([]string{":foo"}))
	if !resp.Queued || resp.Error != "" {
		t.Errorf("response = %+v, want queued", resp)
	}

	h.runQueued(t)
	if err := <-h.reported; err == nil || err.Error() != "boom" {
		t.Errorf("ReportError(%v), want boom", err)
	}
}

// The liveness probe of a starting aerc instance must be answered even when
// the main loop is blocked, and must not run anything.
func TestServerProbeDoesNotSchedule(t *testing.T) {
	h := newFakeHandler(nil)
	startTestServer(t, h, time.Hour)

	resp := waitResult(t, execAsync(nil))
	if resp.Error != "" || resp.Queued {
		t.Errorf("response = %+v, want success", resp)
	}
	select {
	case <-h.queue:
		t.Error("probe scheduled a command")
	default:
	}
}

func TestPickupFromEnv(t *testing.T) {
	tests := []struct {
		val  string
		want time.Duration
	}{
		{"", pickupTimeout},
		{"500ms", 500 * time.Millisecond},
		{"10s", 10 * time.Second},
		{"5", pickupTimeout},
		{"0s", pickupTimeout},
		{"-1s", pickupTimeout},
		{"soon", pickupTimeout},
	}
	for _, tt := range tests {
		t.Setenv(TimeoutEnv, tt.val)
		if got := pickupFromEnv(); got != tt.want {
			t.Errorf("%s=%q: got %s, want %s", TimeoutEnv, tt.val, got, tt.want)
		}
	}
}

// A connected client that sends nothing must not hold up other clients.
func TestServerIdleClientDoesNotBlockOthers(t *testing.T) {
	h := newFakeHandler(nil)
	startTestServer(t, h, time.Hour)

	idle, err := net.Dial("unix", xdg.RuntimePath("aerc.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idle.Close() }()

	res := execAsync([]string{":foo"})
	h.runQueued(t)
	if resp := waitResult(t, res); resp.Error != "" {
		t.Errorf("response = %+v, want success", resp)
	}
}
