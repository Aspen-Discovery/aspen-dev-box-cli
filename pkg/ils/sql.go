package ils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (c *Config) WriteSQL(path, extrasRoot string) error {
	sql, err := c.RenderSQL(extrasRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sql), 0o644)
}

func (c *Config) RenderSQL(extrasRoot string) (string, error) {
	var b strings.Builder
	b.WriteString("SET FOREIGN_KEY_CHECKS=0;\n")
	if err := writeInsert(&b, "account_profiles", c.AccountProfile); err != nil {
		return "", err
	}
	if err := writeInsert(&b, "indexing_profiles", c.IndexingProfile); err != nil {
		return "", err
	}
	if c.Driver != "" {
		fmt.Fprintf(&b, "UPDATE modules SET enabled = 1 WHERE name = %s;\n", quote(c.Driver))
	}
	if c.ExtrasSQL != "" {
		extras, err := os.ReadFile(filepath.Join(extrasRoot, c.ExtrasSQL))
		if err != nil {
			return "", fmt.Errorf("read extras sql: %w", err)
		}
		b.Write(extras)
		if !strings.HasSuffix(string(extras), "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString("SET FOREIGN_KEY_CHECKS=1;\n")
	return b.String(), nil
}

func writeInsert(b *strings.Builder, table string, row map[string]any) error {
	if len(row) == 0 {
		return nil
	}
	cols := make([]string, 0, len(row))
	for k := range row {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	vals := make([]string, len(cols))
	for i, c := range cols {
		v, err := sqlValue(row[c])
		if err != nil {
			return fmt.Errorf("%s.%s: %w", table, c, err)
		}
		vals[i] = v
	}
	fmt.Fprintf(b, "INSERT INTO %s (%s) VALUES (%s);\n",
		table, strings.Join(cols, ", "), strings.Join(vals, ", "))
	return nil
}

func sqlValue(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "NULL", nil
	case bool:
		if x {
			return "1", nil
		}
		return "0", nil
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", x), nil
	case string:
		return quote(x), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", v)
	}
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
