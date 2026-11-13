package config

import (
	"strings"
	"sync/atomic"

	"github.com/go-ini/ini"
	"github.com/seamanticus/aerc/lib/log"
)

type Opener struct {
	Mime string
	Args string
}

var openersConfig atomic.Pointer[[]Opener]

func Openers() []Opener {
	return *openersConfig.Load()
}

func parseOpeners(file *ini.File) ([]Opener, error) {
	var conf []Opener
	openers, err := file.GetSection("openers")
	if err != nil {
		goto out
	}

	for _, key := range openers.Keys() {
		mime := strings.ToLower(key.Name())
		conf = append(conf, Opener{Mime: mime, Args: key.Value()})
	}

out:
	log.Debugf("aerc.conf: [openers] %#v", conf)
	return conf, nil
}
