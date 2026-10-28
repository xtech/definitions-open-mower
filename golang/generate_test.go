package golang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEverySchemaHasAClient fails when a service.json has no generated client
// (i.e. a new service was added without a //go:generate directive, or one was
// deleted). Together with `go generate ./... && git diff --exit-code` in CI this
// keeps the clients in sync with the schemas.
func TestEverySchemaHasAClient(t *testing.T) {
	schemas, err := filepath.Glob("../*_service.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) == 0 {
		t.Skip("no schemas next to the module (running outside the repository?)")
	}

	for _, path := range schemas {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		pkg := strings.ToLower(schema.Type)
		client := filepath.Join(pkg, "client_generated.go")
		if _, err := os.Stat(client); err != nil {
			t.Errorf("%s (%s): missing %s - add a //go:generate directive and run go generate ./...",
				path, schema.Type, client)
		}
	}
}
