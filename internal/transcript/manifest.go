package transcript

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// BuildFileManifest hashes file contents for paths referenced by tool calls.
func BuildFileManifest(projectPath string, calls []ToolCall) map[string]string {
	if projectPath == "" {
		return nil
	}
	seen := make(map[string]bool)
	out := make(map[string]string)
	for _, tc := range calls {
		p := tc.Target
		if p == "" {
			p = PathFromToolInput(tc)
		}
		p = strings.Trim(p, `"'`+"`")
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		abs := p
		if !filepath.IsAbs(p) {
			abs = filepath.Join(projectPath, filepath.Clean(p))
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		out[p] = hex.EncodeToString(sum[:])
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// PathFromToolInput extracts a file path from tool call input fields.
func PathFromToolInput(tc ToolCall) string {
	if tc.Input == nil {
		return ""
	}
	for _, k := range []string{"path", "file_path", "target_file"} {
		if raw, ok := tc.Input[k]; ok {
			var p string
			_ = json.Unmarshal(raw, &p)
			if p != "" {
				return p
			}
		}
	}
	return ""
}
