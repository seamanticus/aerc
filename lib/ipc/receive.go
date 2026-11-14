package ipc

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/seamanticus/aerc/lib/log"
	"github.com/seamanticus/aerc/lib/xdg"
)

type AercServer struct {
	listener net.Listener
	handler  Handler
	startup  context.Context
	// pickup bounds how long a waiting client is kept waiting for its
	// command to start on the main loop, which may be blocked on a
	// subprocess that is itself the client.
	pickup time.Duration
}

// Default for AercServer.pickup; commands normally start within a frame.
const pickupTimeout = 2 * time.Second

// TimeoutEnv overrides pickupTimeout in the running instance. Its value is a
// positive Go duration such as "500ms" or "5s".
const TimeoutEnv = "AERC_IPC_TIMEOUT"

// pickupFromEnv returns the pickup timeout configured with TimeoutEnv, or
// the default when it is unset or invalid.
func pickupFromEnv() time.Duration {
	val := os.Getenv(TimeoutEnv)
	if val == "" {
		return pickupTimeout
	}
	d, err := time.ParseDuration(val)
	if err != nil || d <= 0 {
		log.Warnf("ipc: invalid %s=%q (want a positive duration such as 5s), using %s",
			TimeoutEnv, val, pickupTimeout)
		return pickupTimeout
	}
	return d
}

func StartServer(handler Handler, startup context.Context) (*AercServer, error) {
	sockpath := xdg.RuntimePath("aerc.sock")
	// remove the socket if it is not connected to a session
	if _, err := ConnectAndExec(nil); err != nil {
		// stale socket, if any; Listen reports the real failure
		_ = os.Remove(sockpath)
	}
	log.Debugf("Starting Unix server: %s", sockpath)
	l, err := net.Listen("unix", sockpath)
	if err != nil {
		return nil, err
	}
	as := &AercServer{
		listener: l, handler: handler, startup: startup, pickup: pickupFromEnv(),
	}
	go as.Serve()

	return as, nil
}

func (as *AercServer) Close() {
	if err := as.listener.Close(); err != nil {
		log.Errorf("ipc: failed to close listener: %v", err)
	}
}

var lastId int64 = 0 // access via atomic

func (as *AercServer) Serve() {
	defer log.PanicHandler()

	<-as.startup.Done()

	for {
		conn, err := as.listener.Accept()
		switch {
		case errors.Is(err, net.ErrClosed):
			log.Infof("shutting down UNIX listener")
			return
		case err != nil:
			log.Errorf("ipc: accepting connection failed: %v", err)
			continue
		}

		clientId := atomic.AddInt64(&lastId, 1)
		log.Debugf("unix:%d accepted connection", clientId)
		// One goroutine per connection so that an idle or slow client does
		// not hold up the others until its deadline expires.
		go as.serveConn(conn, clientId)
	}
}

func (as *AercServer) serveConn(conn net.Conn, clientId int64) {
	defer log.PanicHandler()

	scanner := bufio.NewScanner(conn)
	err := conn.SetDeadline(time.Now().Add(1 * time.Minute))
	if err != nil {
		log.Errorf("unix:%d failed to set deadline: %v", clientId, err)
	}
	for scanner.Scan() {
		// allow up to 1 minute between commands
		err = conn.SetDeadline(time.Now().Add(1 * time.Minute))
		if err != nil {
			log.Errorf("unix:%d failed to update deadline: %v", clientId, err)
		}
		msg, err := DecodeRequest(scanner.Bytes())
		log.Tracef("unix:%d got message %s", clientId, scanner.Text())
		if err != nil {
			log.Errorf("unix:%d failed to parse request: %v", clientId, err)
			continue
		}

		response := as.handleMessage(msg)
		result, err := response.Encode()
		if err != nil {
			log.Errorf("unix:%d failed to encode result: %v", clientId, err)
			continue
		}
		_, err = conn.Write(append(result, '\n'))
		if err != nil {
			log.Errorf("unix:%d failed to send response: %v", clientId, err)
			break
		}
	}
	if err := conn.Close(); err != nil {
		log.Errorf("unix:%d failed to close connection: %v", clientId, err)
	}
	log.Tracef("unix:%d closed connection", clientId)
}

// States of a queued command, shared by the connection goroutine and the
// function running on the main loop.
const (
	cmdPending   int32 = iota // not started yet
	cmdStarted                // started; the client waits for the result
	cmdAbandoned              // the client got a Queued response
)

// handleMessage runs the command on the main loop through the handler.
// The result is returned to the client unless the request is async or the
// command did not start within the pickup timeout; it is then reported by
// the handler instead.
func (as *AercServer) handleMessage(req *Request) *Response {
	if len(req.Arguments) == 0 {
		// liveness probe from StartServer: touches no state
		return &Response{}
	}
	var state atomic.Int32
	result := make(chan error, 1)
	run := func() {
		waiting := !req.Async && state.CompareAndSwap(cmdPending, cmdStarted)
		err := as.handler.Command(req.Arguments)
		if err != nil {
			log.Errorf("ipc: command %q failed: %v", req.Arguments, err)
		}
		switch {
		case waiting:
			result <- err
		case err != nil:
			as.handler.ReportError(err)
		}
	}
	// Schedule may block while the main loop queue is full; do not let
	// that delay the pickup timeout below.
	go func() {
		defer log.PanicHandler()
		as.handler.Schedule(run)
	}()

	if req.Async {
		return &Response{Queued: true}
	}
	timer := time.NewTimer(as.pickup)
	defer timer.Stop()
	var err error
	select {
	case err = <-result:
	case <-timer.C:
		if state.CompareAndSwap(cmdPending, cmdAbandoned) {
			log.Warnf("ipc: command %q not started after %s, not waiting",
				req.Arguments, as.pickup)
			return &Response{Queued: true}
		}
		// started just before the timeout: it runs to completion
		err = <-result
	}
	if err != nil {
		return &Response{Error: err.Error()}
	}
	return &Response{}
}
