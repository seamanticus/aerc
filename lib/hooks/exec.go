package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
)

func RunHook(h HookType) error {
	cmd := h.Cmd()
	if cmd == "" {
		return nil
	}
	env := h.Env()
	log.Debugf("hooks: running %T command %q (env %v)", h, cmd, env)

	proc := exec.Command("sh", "-c", cmd)

	var outb, errb bytes.Buffer
	proc.Stdout = &outb
	proc.Stderr = &errb

	// add hook environment variables
	proc.Env = os.Environ()
	proc.Env = append(proc.Env, env...)

	// append to PATH to find hooks
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	paths = append(config.ResourceDirs("hooks"), paths...)
	path := strings.Join(paths, string(os.PathListSeparator))
	proc.Env = append(proc.Env, fmt.Sprintf("PATH=%s", path))

	err := proc.Run()
	log.Tracef("hooks: %q stdout: %s", cmd, outb.String())
	if err != nil {
		log.Errorf("hooks:%q stderr: %s", cmd, errb.String())
	}
	return err
}
