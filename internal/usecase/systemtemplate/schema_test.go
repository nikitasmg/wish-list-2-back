package systemtemplate

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEmbeddedTemplatesUseOnlyCoordinateLayout(t *testing.T) {
	var templates []struct {
		Blocks []map[string]json.RawMessage `json:"blocks"`
	}
	require.NoError(t, json.Unmarshal(templatesJSON, &templates))
	require.Len(t, templates, 8)
	for _, template := range templates {
		for _, block := range template.Blocks {
			for _, field := range []string{"row", "col", "colSpan"} {
				require.Contains(t, block, field)
			}
			for _, field := range []string{"position", "mobilePosition", "rowSpan"} {
				require.NotContains(t, block, field)
			}
		}
	}
}
