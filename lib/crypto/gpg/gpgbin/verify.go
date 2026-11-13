package gpgbin

import (
	"bytes"
	"io"
	"os"

	"github.com/seamanticus/aerc/config"
	"github.com/seamanticus/aerc/models"
)

// Verify runs gpg --verify. If s is not nil, then gpg interprets the
// arguments as a detached signature
func Verify(m io.Reader, s io.Reader) (*models.MessageDetails, error) {
	args := []string{"--verify"}
	if s != nil {
		// Detached sig, save the sig to a tmp file and send msg over stdin
		sig, err := os.CreateTemp(config.General().TempDir, "sig")
		if err != nil {
			return nil, err
		}
		defer func() { _ = os.Remove(sig.Name()) }()
		_, err = io.Copy(sig, s)
		if cerr := sig.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
		args = append(args, sig.Name(), "-")
	}
	orig, err := io.ReadAll(m)
	if err != nil {
		return nil, err
	}
	g := newGpg(bytes.NewReader(orig), args)
	_ = g.cmd.Run()

	md := new(models.MessageDetails)
	_ = parseStatusFd(bytes.NewReader(g.stderr.Bytes()), md)

	md.Body = bytes.NewReader(orig)

	return md, nil
}
