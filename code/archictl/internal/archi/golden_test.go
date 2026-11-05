package archi_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
)

const goldenDir = "../../testdata/golden"

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// plain converts Obj trees into map[string]any so that maps compare unordered and lists ordered.
func plain(v any) any {
	switch x := v.(type) {
	case archi.Obj:
		m := map[string]any{}
		for _, k := range x.Keys() {
			m[k] = plain(x.Get(k))
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = plain(x[i])
		}
		return out
	}
	return v
}

func yamlDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = readFile(t, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func goldenModels(t *testing.T) []string {
	t.Helper()
	var m struct{ Models []string }
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(goldenDir, "manifest.json"))), &m); err != nil {
		t.Fatal(err)
	}
	return m.Models
}

func jsonNorm(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGoldenSplitData(t *testing.T) {
	for _, name := range goldenModels(t) {
		for _, mode := range []struct {
			dir   string
			names bool
		}{{"yaml", true}, {"yaml_ids", false}} {
			t.Run(name+"/"+mode.dir, func(t *testing.T) {
				base := filepath.Join(goldenDir, "models", name)
				got, err := archi.SplitToFiles(readFile(t, filepath.Join(base, "in.archimate")), mode.names)
				if err != nil {
					t.Fatal(err)
				}
				want := yamlDir(t, filepath.Join(base, mode.dir))
				if len(got) != len(want) {
					t.Fatalf("file set differs: got %d files, want %d", len(got), len(want))
				}
				for fn, wtxt := range want {
					gtxt, ok := got[fn]
					if !ok {
						t.Fatalf("missing file %s", fn)
					}
					gv, err := archi.LoadYAML(gtxt)
					if err != nil {
						t.Fatalf("%s: %v", fn, err)
					}
					wv, err := archi.LoadYAML(wtxt)
					if err != nil {
						t.Fatalf("golden %s: %v", fn, err)
					}
					if !reflect.DeepEqual(plain(gv), plain(wv)) {
						t.Errorf("%s: parsed YAML differs from golden", fn)
					}
				}
			})
		}
	}
}

func TestGoldenJoinCanon(t *testing.T) {
	for _, name := range goldenModels(t) {
		for _, dir := range []string{"yaml", "yaml_ids"} {
			t.Run(name+"/"+dir, func(t *testing.T) {
				base := filepath.Join(goldenDir, "models", name)
				text, err := archi.JoinFromFiles(yamlDir(t, filepath.Join(base, dir)), false, false)
				if err != nil {
					t.Fatal(err)
				}
				sig, err := archi.CanonSig(text, true)
				if err != nil {
					t.Fatal(err)
				}
				var want any
				if err := json.Unmarshal([]byte(readFile(t, filepath.Join(base, "canon.json"))), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(jsonNorm(t, sig), want) {
					t.Error("canonical form differs from canon.json")
				}
			})
		}
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	for _, name := range goldenModels(t) {
		t.Run(name, func(t *testing.T) {
			text := readFile(t, filepath.Join(goldenDir, "models", name, "in.archimate"))
			for _, names := range []bool{true, false} {
				ok, err := archi.RoundTrip(text, names)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Errorf("round trip (names=%v) changed the canonical form", names)
				}
			}
		})
	}
}

func TestSplitDeterministic(t *testing.T) {
	text := readFile(t, filepath.Join(goldenDir, "models", "archisurance_annotated", "in.archimate"))
	a, err := archi.SplitToFiles(text, true)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := archi.SplitToFiles(text, true)
	if !reflect.DeepEqual(a, b) {
		t.Error("split is not deterministic")
	}
}

func TestGoldenJoinErrors(t *testing.T) {
	var spec struct {
		Base  string
		Cases []struct {
			Name        string
			File        string
			Find        string
			Replace     string
			ExpectError string `json:"expect_error"`
		}
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(goldenDir, "join_errors", "cases.json"))), &spec); err != nil {
		t.Fatal(err)
	}
	for _, c := range spec.Cases {
		t.Run(c.Name, func(t *testing.T) {
			files := yamlDir(t, filepath.Join(goldenDir, spec.Base, "yaml"))
			if !strings.Contains(files[c.File], c.Find) {
				t.Fatalf("anchor not found in %s", c.File)
			}
			files[c.File] = strings.Replace(files[c.File], c.Find, c.Replace, 1)
			_, err := archi.JoinFromFiles(files, true, true)
			if err == nil {
				t.Fatal("expected join to fail")
			}
			if !strings.Contains(err.Error(), c.ExpectError) {
				t.Errorf("error %q does not contain %q", err.Error(), c.ExpectError)
			}
		})
	}
}
