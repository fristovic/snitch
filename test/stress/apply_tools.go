package stress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

func applyToolCalls(projectDir string, calls []transcript.ToolCall) error {
	for _, tc := range calls {
		path := tc.Target
		if path == "" && tc.Input != nil {
			if raw, ok := tc.Input["path"]; ok {
				_ = json.Unmarshal(raw, &path)
			}
		}
		if path == "" {
			continue
		}
		abs := filepath.Join(projectDir, filepath.Clean(path))
		switch tc.Name {
		case "Write":
			body := toolString(tc.Input, "contents")
			if err := writeFile(abs, body); err != nil {
				return err
			}
		case "StrReplace":
			old := toolString(tc.Input, "old_string")
			newStr := toolString(tc.Input, "new_string")
			data, err := os.ReadFile(abs)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				data = nil
			}
			content := string(data)
			if old == "" {
				content = newStr
			} else if strings.Contains(content, old) {
				content = strings.Replace(content, old, newStr, 1)
			} else if len(data) == 0 {
				content = newStr
			}
			if strings.TrimSpace(content) == "" {
				_ = os.Remove(abs)
				continue
			}
			if err := writeFile(abs, content); err != nil {
				return err
			}
		case "Delete":
			_ = os.Remove(abs)
		}
	}
	return nil
}

func toolString(input map[string]json.RawMessage, key string) string {
	if input == nil {
		return ""
	}
	raw, ok := input[key]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}
