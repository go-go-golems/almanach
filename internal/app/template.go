package app

import (
	"fmt"
	"os"
	"strings"
)

// ResolveTemplate walks a layout object and resolves all {{expr}} expressions
// using the provided data context. When ctx is empty, this is a no-op —
// existing layout files without template expressions are completely unaffected.
func ResolveTemplate(obj map[string]interface{}, ctx map[string]string) error {
	if len(ctx) == 0 {
		return nil
	}
	return walkMap(obj, ctx)
}

// walkMap resolves template expressions in all string values within a map.
func walkMap(m map[string]interface{}, ctx map[string]string) error {
	for k, v := range m {
		resolved, err := walkValue(v, ctx)
		if err != nil {
			return fmt.Errorf("key %q: %w", k, err)
		}
		m[k] = resolved
	}
	return nil
}

// walkValue resolves template expressions in a value, recursing into maps and slices.
func walkValue(v interface{}, ctx map[string]string) (interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		if err := walkMap(val, ctx); err != nil {
			return nil, err
		}
		return val, nil
	case []interface{}:
		for i, child := range val {
			resolved, err := walkValue(child, ctx)
			if err != nil {
				return nil, fmt.Errorf("index %d: %w", i, err)
			}
			val[i] = resolved
		}
		return val, nil
	case string:
		if strings.Contains(val, "{{") && strings.Contains(val, "}}") {
			return resolveValue(val, ctx)
		}
		return val, nil
	default:
		return v, nil
	}
}

// resolveValue resolves all {{expr}} markers in a single string value.
func resolveValue(s string, ctx map[string]string) (string, error) {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			end := strings.Index(s[i:], "}}")
			if end == -1 {
				return "", fmt.Errorf("unclosed {{ in template string")
			}
			expr := strings.TrimSpace(s[i+2 : i+end])
			if expr == "" {
				return "", fmt.Errorf("empty template expression {{}}")
			}
			val, err := resolveExpr(expr, ctx)
			if err != nil {
				return "", fmt.Errorf("{{%s}}: %w", expr, err)
			}
			result.WriteString(val)
			i = i + end + 2
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String(), nil
}

// resolveExpr resolves a single expression: key, key:fallback, $ENV, $ENV:fallback.
func resolveExpr(expr string, ctx map[string]string) (string, error) {
	// Split on first colon for fallback value.
	parts := strings.SplitN(expr, ":", 2)
	key := parts[0]
	hasFallback := len(parts) == 2
	fallback := ""
	if hasFallback {
		fallback = parts[1]
	}

	// Environment variable syntax: $NAME
	if strings.HasPrefix(key, "$") {
		envKey := key[1:]
		if val, ok := os.LookupEnv(envKey); ok {
			return val, nil
		}
		if hasFallback {
			return fallback, nil
		}
		return "", fmt.Errorf("environment variable %s not set", envKey)
	}

	// Regular key lookup in data context.
	if val, ok := ctx[key]; ok {
		return val, nil
	}
	if hasFallback {
		return fallback, nil
	}
	return "", fmt.Errorf("template variable %q not provided (use --data or -D %s=<value>)", key, key)
}
