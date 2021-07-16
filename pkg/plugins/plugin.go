package plugins

import (
	"fmt"
	"strings"
)

type Plugin interface {
	Init() error
	Name() string
	GetParam(string) string
}

// DecomposeCSL splits the comma separated list of plugins to be configured and registered
func DecomposeCSL(in string) (map[string]Plugin, error) {
	out := make(map[string]Plugin)
	if in == "" {
		return nil, fmt.Errorf("no plugins specified")
	}

	pStrList := strings.Split(in, ",")
	if len(pStrList) == 0 {
		return nil, fmt.Errorf("no plugins specified")
	}

	for _, pName := range pStrList {
		p := getPluginFromName(pName)
		if p == nil {
			return nil, fmt.Errorf("plugin %s not found", pName)
		}
		out[pName] = p
	}

	return out, nil
}

func getPluginFromName(name string) Plugin {
	switch name {
	case "keptn":
		return keptn()
	default:
		return nil
	}
}
