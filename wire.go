package hexcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// decodeFields keeps inconsistent API representations out of public field types.
// Only the named scalar fields and documented aliases are normalized.
func decodeFields(data []byte, out any, integers, texts []string, aliases map[string]string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("expected a resource object")
	}
	for alias, name := range aliases {
		if _, exists := fields[name]; !exists {
			if value, ok := fields[alias]; ok {
				fields[name] = value
			}
		}
		delete(fields, alias)
	}
	for _, name := range integers {
		raw := bytes.TrimSpace(fields[name])
		if len(raw) == 0 || raw[0] != '"' {
			continue
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return fmt.Errorf("invalid %s integer", name)
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid %s integer", name)
		}
		fields[name] = json.RawMessage(strconv.FormatInt(n, 10))
	}
	for _, name := range texts {
		raw := bytes.TrimSpace(fields[name])
		if len(raw) == 0 || raw[0] == '"' || bytes.Equal(raw, []byte("null")) {
			continue
		}
		if raw[0] != '-' && (raw[0] < '0' || raw[0] > '9') {
			return fmt.Errorf("invalid %s string or number", name)
		}
		fields[name], _ = json.Marshal(string(raw))
	}
	normalized, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(normalized, out)
}

func requireID(id string, err error) error {
	if err != nil {
		return err
	}
	if id == "" {
		return errors.New("resource object has no ID")
	}
	return nil
}

func numericID(id string) (json.Number, error) {
	if id == "" {
		return "", errors.New("hexcore: VDS ID must be a positive decimal integer")
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return "", errors.New("hexcore: VDS ID must be a positive decimal integer")
		}
	}
	id = strings.TrimLeft(id, "0")
	if id == "" {
		return "", errors.New("hexcore: VDS ID must be a positive decimal integer")
	}
	return json.Number(id), nil
}
