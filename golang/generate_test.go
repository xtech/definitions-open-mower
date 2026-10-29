package golang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xtech/xbot_framework/xbot_service_interface_go/codegen"
)

// TestEverySchemaHasAClient fails when a service.json has no generated client
// (i.e. a new service was added without a //go:generate directive, or one was
// deleted). TestGeneratedClientsAreUpToDate checks the content of the clients
// that do have a directive, so together they keep the module in sync with the
// schemas without relying on git.
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

// clientDirective is one //go:generate line that produces a client.
type clientDirective struct {
	service   string // -service, relative to this package
	serviceID uint16 // -service-id, 0: Dial takes the id as an argument
	pkg       string // -package, "" : derived from the service type
	out       string // -out, relative to this package
}

// clientDirectives parses the //go:generate lines of generate.go. They are the
// single source of truth for which schema becomes which client with which
// service id, so the test below checks exactly what `go generate` writes.
func clientDirectives(path string) ([]clientDirective, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []clientDirective
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "//go:generate") || !strings.Contains(line, "xbot-codegen") {
			continue
		}
		flags := map[string]string{}
		fields := strings.Fields(strings.TrimPrefix(line, "//go:generate"))
		for i := 0; i+1 < len(fields); i++ {
			if strings.HasPrefix(fields[i], "-") {
				flags[fields[i]] = fields[i+1]
			}
		}
		d := clientDirective{service: flags["-service"], pkg: flags["-package"], out: flags["-out"]}
		if d.service == "" || d.out == "" {
			return nil, fmt.Errorf("%s: //go:generate without -service/-out: %s", path, line)
		}
		if raw := flags["-service-id"]; raw != "" {
			id, err := strconv.ParseUint(raw, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("%s: bad -service-id %q in %s", path, raw, line)
			}
			d.serviceID = uint16(id)
		}
		out = append(out, d)
	}
	return out, nil
}

// TestGeneratedClientsAreUpToDate regenerates every client from its schema and
// compares the result with the checked in file. The generator is a library
// (codegen.Generate), so this needs no `go generate`, no git and no network: a
// schema change without a regenerated client fails `go test ./...` right where
// the schema was edited, instead of only in CI.
func TestGeneratedClientsAreUpToDate(t *testing.T) {
	directives, err := clientDirectives("generate.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(directives) == 0 {
		t.Fatal("generate.go has no xbot-codegen //go:generate directive")
	}

	for _, d := range directives {
		t.Run(d.out, func(t *testing.T) {
			schema, err := codegen.Load(d.service)
			if err != nil {
				t.Fatalf("load %s: %v", d.service, err)
			}
			want, err := codegen.Generate(schema, codegen.Options{ServiceID: d.serviceID, Package: d.pkg})
			if err != nil {
				t.Fatalf("generate %s: %v", d.service, err)
			}
			got, err := os.ReadFile(d.out)
			if err != nil {
				t.Fatalf("%s is missing (run `go generate ./...`): %v", d.out, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s does not match %s - run `go generate ./...` and commit the result:\n%s",
					d.out, d.service, firstDiff(got, want))
			}
		})
	}
}

// firstDiff names the first line that differs, so a stale client reads as one
// changed line instead of two dumped files.
func firstDiff(got, want []byte) string {
	gotLines := strings.Split(string(got), "\n")
	wantLines := strings.Split(string(want), "\n")
	for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
		if gotLines[i] != wantLines[i] {
			return fmt.Sprintf("line %d:\n  generated: %s\n  committed: %s", i+1, wantLines[i], gotLines[i])
		}
	}
	// One file is a prefix of the other: same content up to the shorter one.
	n := min(len(gotLines), len(wantLines))
	return fmt.Sprintf("identical up to line %d, then %d committed vs %d generated lines",
		n, len(gotLines), len(wantLines))
}
