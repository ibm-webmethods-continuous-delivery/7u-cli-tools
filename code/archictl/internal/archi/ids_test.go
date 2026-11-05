package archi

import "testing"

func TestB36(t *testing.T) {
	for n, want := range map[int]string{0: "0", 9: "9", 10: "a", 35: "z", 36: "10", 1295: "zz", 1296: "100"} {
		if got := B36(n); got != want {
			t.Errorf("B36(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestSlug(t *testing.T) {
	used := map[string]bool{}
	cases := []struct{ in, want string }{
		{"Business Process View", "Business_Process_View"},
		{"Business Process View", "Business_Process_View_2"},
		{"business process view", "business_process_view_3"},
		{"", "view"},
		{"///", "view_2"},
		{"a.b-c", "a.b-c"},
	}
	for _, c := range cases {
		if got := Slug(c.in, used); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	long := Slug("0123456789012345678901234567890123456789012345678901234567890", map[string]bool{})
	if len(long) != 50 {
		t.Errorf("slug length = %d, want 50", len(long))
	}
}

func TestCollectIDsAndGen(t *testing.T) {
	obj := O("model", O("id", "m1", "kids", []any{O("folder", O("id", "f1")), "ignored", O("x", "y")}))
	taken := map[string]bool{}
	CollectIDs(obj, taken)
	if !taken["m1"] || !taken["f1"] || len(taken) != 2 {
		t.Fatalf("taken = %v", taken)
	}
	taken["f2"] = true
	gen := NewIDGen("f", taken)
	if a, b := gen(), gen(); a != "f3" || b != "f4" {
		t.Errorf("gen produced %q, %q; want f3, f4", a, b)
	}
}
