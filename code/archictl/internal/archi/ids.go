package archi

import (
	"regexp"
	"strconv"
	"strings"
)

// Python's \w is Unicode-aware; mirror that.
var slugRe = regexp.MustCompile(`[^\p{L}\p{N}_.-]+`)

// B36 encodes n in base-36 (0→"0", 10→"a", 35→"z", 36→"10").
func B36(n int) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n <= 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(digits[n%36]) + s
		n /= 36
	}
	return s
}

// Slug converts name to a stable file-name fragment, avoiding collisions (case-insensitive).
// used is updated with the lower-cased result.
func Slug(name string, used map[string]bool) string {
	s := strings.Trim(slugRe.ReplaceAllString(name, "_"), "_")
	if r := []rune(s); len(r) > 50 {
		s = string(r[:50])
	}
	if s == "" {
		s = "view"
	}
	base := s
	for n := 1; used[strings.ToLower(s)]; {
		n++
		s = base + "_" + strconv.Itoa(n)
	}
	used[strings.ToLower(s)] = true
	return s
}

// CollectIDs walks a nested Obj/[]any structure and adds every string "id" value to taken.
func CollectIDs(x any, taken map[string]bool) {
	switch v := x.(type) {
	case Obj:
		if v == nil {
			return
		}
		for _, k := range v.keys {
			val := v.m[k]
			if k == "id" {
				if s, ok := val.(string); ok {
					taken[s] = true
				}
			}
			CollectIDs(val, taken)
		}
	case []any:
		for _, item := range v {
			CollectIDs(item, taken)
		}
	}
}

// NewIDGen returns a generator yielding prefix+b36(n) for n=1,2,... skipping ids already taken.
// Generated ids are added to taken.
func NewIDGen(prefix string, taken map[string]bool) func() string {
	n := 0
	return func() string {
		for {
			n++
			cand := prefix + B36(n)
			if !taken[cand] {
				taken[cand] = true
				return cand
			}
		}
	}
}
