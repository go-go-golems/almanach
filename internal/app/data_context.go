package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DataContext is a flat key-value mapping used for template resolution.
type DataContext map[string]string

// LoadDataContext builds a DataContext from a file and/or inline key=value pairs.
// Inline defines override file values. Returns an empty context if both are empty.
func LoadDataContext(dataPath string, defines []string) (DataContext, error) {
	ctx := DataContext{}

	if dataPath != "" {
		data, err := os.ReadFile(dataPath)
		if err != nil {
			return nil, fmt.Errorf("read data file %s: %w", dataPath, err)
		}

		var raw map[string]interface{}
		ext := strings.ToLower(filepath.Ext(dataPath))
		if ext == ".json" {
			if err := json.Unmarshal(data, &raw); err != nil {
				return nil, fmt.Errorf("parse data file %s: %w", dataPath, err)
			}
		} else {
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return nil, fmt.Errorf("parse data file %s: %w", dataPath, err)
			}
		}

		for k, v := range raw {
			ctx[k] = fmt.Sprintf("%v", v)
		}
	}

	for _, d := range defines {
		parts := strings.SplitN(d, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid --define %q (expected key=value)", d)
		}
		ctx[parts[0]] = parts[1]
	}

	return ctx, nil
}

// loadDataCtxFromFlags is a helper used by CLI commands to build a DataContext
// from the --data and --define flags.
func loadDataCtxFromFlags(dataFlag, defineFlag string) (DataContext, error) {
	var defines []string
	if defineFlag != "" {
		defines = strings.Split(defineFlag, ",")
	}
	return LoadDataContext(dataFlag, defines)
}
