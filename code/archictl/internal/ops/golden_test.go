package ops_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/ops"
)

const goldenDir = "../../testdata/golden"

// run mimics one oracle invocation and returns (exit code, trimmed output). Edits go through
// RunEdit exactly like the commands do.
func run(t *testing.T, model string, args []string) (int, string) {
	t.Helper()
	opt := func(name string) *int {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				v := 0
				for _, c := range args[i+1] {
					v = v*10 + int(c-'0')
				}
				return &v
			}
		}
		return nil
	}
	flag := func(name string) string {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	has := func(name string) bool {
		for _, a := range args {
			if a == name {
				return true
			}
		}
		return false
	}
	var pos []string
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			if args[i] != "--json" {
				i++
			}
			continue
		}
		pos = append(pos, args[i])
	}
	fail := func(err error) (int, string) {
		var r *archi.Refusal
		if errors.As(err, &r) {
			return 1, r.Msg
		}
		t.Fatalf("%v: %v", args, err)
		return 2, ""
	}
	cmd := args[0]
	switch cmd {
	case "validate", "find", "show", "orphans", "unused":
		m, err := ops.LoadModel(model)
		if err != nil {
			t.Fatal(err)
		}
		switch cmd {
		case "validate":
			if errs := m.ValidateAll(); len(errs) > 0 {
				return 0, strings.Join(errs, "\n")
			}
			return 0, "OK"
		case "find":
			q := ""
			if len(pos) > 0 {
				q = pos[0]
			}
			items := ops.Find(m, q, flag("--type"))
			if has("--json") {
				return 0, archi.PyJSON(items)
			}
			return 0, ops.FormatFind(items)
		case "show":
			d, err := ops.Show(m, pos[0])
			if err != nil {
				return fail(err)
			}
			if has("--json") {
				return 0, archi.PyJSON(d)
			}
			return 0, ops.FormatShow(m, d)
		case "orphans":
			return 0, ops.FormatOrphans(ops.Orphans(m))
		default:
			return 0, ops.FormatUnused(ops.Unused(m))
		}
	}
	msg, err := ops.RunEdit(model, "", func(m *ops.Model) (string, error) {
		switch cmd {
		case "add":
			e, err := ops.Add(m, pos[0], pos[1], flag("--folder"), flag("--doc"))
			if err != nil {
				return "", err
			}
			return "added " + m.Label(m.ByID[e.Str("id")]), nil
		case "connect":
			r, err := ops.Connect(m, pos[0], pos[1], pos[2], flag("--name"))
			if err != nil {
				return "", err
			}
			return "added " + m.ByID[r.Str("id")].Type() + " #" + r.Str("id"), nil
		case "rename":
			e, old, err := ops.Rename(m, pos[0], pos[1])
			if err != nil {
				return "", err
			}
			return "renamed " + archi.PyRepr(old) + " -> " + archi.PyRepr(pos[1]) + " #" + e.Str("id"), nil
		case "delete":
			n, err := m.Resolve(pos[0])
			if err != nil {
				return "", err
			}
			typ := n.Type()
			e, rels, nodes, err := ops.Delete(m, pos[0])
			if err != nil {
				return "", err
			}
			return "deleted " + typ + " " + e.Str("name") + " #" + e.Str("id") + "; " +
				itoa(rels) + " relationship(s) and " + itoa(nodes) + " view item(s) removed", nil
		case "add-to-view":
			x, y := opt("--x"), opt("--y")
			v, e, n, c, err := ops.AddToView(m, pos[0], pos[1], x, y)
			if err != nil {
				return "", err
			}
			return "added " + archi.PyRepr(e.Str("name")) + " to view " + archi.PyRepr(v.Str("name")) +
				" with " + itoa(c) + " connection(s) #" + n.Str("id"), nil
		}
		t.Fatalf("unknown command %s", cmd)
		return "", nil
	})
	if err != nil {
		return fail(err)
	}
	return 0, msg
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestGoldenOps(t *testing.T) {
	var spec struct {
		Input string
		Steps []struct {
			Args   []string
			Exit   int
			Output string
		}
	}
	b, err := os.ReadFile(filepath.Join(goldenDir, "ops", "commands.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	in, err := os.ReadFile(filepath.Join(goldenDir, spec.Input))
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(t.TempDir(), "work.archimate")
	if err := os.WriteFile(work, in, 0o644); err != nil {
		t.Fatal(err)
	}
	for i, s := range spec.Steps {
		before, _ := os.ReadFile(work)
		code, out := run(t, work, s.Args)
		if code != s.Exit || strings.TrimRight(out, "\n") != s.Output {
			t.Errorf("step %d %v:\n got exit=%d %q\nwant exit=%d %q", i, s.Args, code, out, s.Exit, s.Output)
		}
		if s.Exit == 1 {
			after, _ := os.ReadFile(work)
			if string(before) != string(after) {
				t.Errorf("step %d %v: refused edit modified the file", i, s.Args)
			}
		}
	}
	if _, err := os.Stat(work + ".lock"); err == nil {
		t.Error("lock file left behind")
	}
	final, _ := os.ReadFile(work)
	sig, err := archi.CanonSig(string(final), true)
	if err != nil {
		t.Fatal(err)
	}
	sb, _ := json.Marshal(sig)
	var got, want any
	_ = json.Unmarshal(sb, &got)
	cb, err := os.ReadFile(filepath.Join(goldenDir, "ops", "canon.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(cb, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("final model differs from ops/canon.json")
	}
}

func TestLockIsExclusive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "m.archimate")
	unlock, err := ops.Lock(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ops.Lock(p); err == nil {
		t.Error("second lock must fail")
	}
	unlock()
	if u2, err := ops.Lock(p); err != nil {
		t.Errorf("lock after release: %v", err)
	} else {
		u2()
	}
}
