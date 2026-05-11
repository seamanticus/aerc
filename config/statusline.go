package config

import (
	"fmt"
	"strings"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type StatuslineConfig struct {
	StatusColumns   []*ColumnDef `ini:"status-columns" parse:"ParseColumns" default:"left<*,center>=,right>*"`
	ColumnSeparator string       `ini:"column-separator" default:" "`
	Separator       string       `ini:"separator" default:" | "`
	DisplayMode     string       `ini:"display-mode" default:"text"`
	pendingKeysCol  int
}

var statuslineConfig atomic.Pointer[StatuslineConfig]

func Statusline() *StatuslineConfig {
	return statuslineConfig.Load()
}

func parseStatusline(file *ini.File) (*StatuslineConfig, error) {
	conf := new(StatuslineConfig)
	statusline := file.Section("statusline")
	if err := MapToStruct(statusline, conf, true); err != nil {
		return nil, err
	}

	log.Debugf("aerc.conf: [statusline] %#v", conf)
	return conf, nil
}

func (s *StatuslineConfig) ParseColumns(sec *ini.Section, key *ini.Key) ([]*ColumnDef, error) {
	if !sec.HasKey("column-left") {
		_, _ = sec.NewKey("column-left", "[{{.Account}}] {{.StatusInfo}}")
	}
	if !sec.HasKey("column-center") {
		_, _ = sec.NewKey("column-center", "{{.PendingKeys}}")
	}
	if !sec.HasKey("column-right") {
		_, _ = sec.NewKey("column-right", "{{.TrayInfo}} | {{cwd}}")
	}
	result, err := ParseColumnDefs(key, sec)
	if err != nil {
		return nil, err
	}
	s.pendingKeysCol = -1
	for i, col := range result {
		raw, err := sec.GetKey(fmt.Sprintf("column-%s", col.Name))
		if err == nil && strings.Contains(raw.String(), "PendingKeys") {
			s.pendingKeysCol = i
			break
		}
	}
	return result, nil
}

func (s *StatuslineConfig) PendingKeysColIndex() int {
	return s.pendingKeysCol
}
