package ipc

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/seamanticus/aerc/lib/xdg"
)

// NoWaitEnv is set in the environment of every process started by aerc. An
// IPC client that finds it sends async requests: aerc may be blocked waiting
// for that very process and could not run the command until it exits.
const NoWaitEnv = "AERC_IPC_NOWAIT"

func ConnectAndExec(args []string) (*Response, error) {
	sockpath := xdg.RuntimePath("aerc.sock")
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	req, err := (&Request{
		Arguments: args,
		Async:     os.Getenv(NoWaitEnv) != "",
	}).Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	_, err = conn.Write(append(req, '\n'))
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, errors.New("no response from server")
	}
	resp, err := DecodeResponse(scanner.Bytes())
	if err != nil {
		return nil, err
	}

	return resp, nil
}
