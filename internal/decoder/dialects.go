package decoder

import (
	"fmt"

	"github.com/bluenviron/gomavlib/v4/pkg/dialect"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/all"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/ardupilotmega"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/common"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/standard"
)

// GetDialect returns the requested gomavlib dialect instance.
func GetDialect(name string) (*dialect.Dialect, error) {
	switch name {
	case "common":
		return common.Dialect, nil
	case "ardupilotmega":
		return ardupilotmega.Dialect, nil
	case "standard":
		return standard.Dialect, nil
	case "all":
		return all.Dialect, nil
	case "raw":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown dialect %q", name)
	}
}
