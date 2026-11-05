package ops

import "sort"

// Port of Python difflib.SequenceMatcher.ratio() (no junk heuristic; names are short)
// and difflib.get_close_matches, so "did you mean" suggestions match the oracle.

func longestMatch(a, b []rune, b2j map[rune][]int, alo, ahi, blo, bhi int) (int, int, int) {
	besti, bestj, bestsize := alo, blo, 0
	j2len := map[int]int{}
	for i := alo; i < ahi; i++ {
		newj2len := map[int]int{}
		for _, j := range b2j[a[i]] {
			if j < blo {
				continue
			}
			if j >= bhi {
				break
			}
			k := j2len[j-1] + 1
			newj2len[j] = k
			if k > bestsize {
				besti, bestj, bestsize = i-k+1, j-k+1, k
			}
		}
		j2len = newj2len
	}
	return besti, bestj, bestsize
}

func ratio(a, b []rune) float64 {
	if len(a)+len(b) == 0 {
		return 1.0
	}
	b2j := map[rune][]int{}
	for j, r := range b {
		b2j[r] = append(b2j[r], j)
	}
	matched := 0
	var rec func(alo, ahi, blo, bhi int)
	rec = func(alo, ahi, blo, bhi int) {
		i, j, k := longestMatch(a, b, b2j, alo, ahi, blo, bhi)
		if k == 0 {
			return
		}
		if alo < i && blo < j {
			rec(alo, i, blo, j)
		}
		matched += k
		if i+k < ahi && j+k < bhi {
			rec(i+k, ahi, j+k, bhi)
		}
	}
	rec(0, len(a), 0, len(b))
	return 2.0 * float64(matched) / float64(len(a)+len(b))
}

// closeMatches returns up to n possibilities whose ratio against word is >= cutoff, best first
// (ties: larger string first, like heapq.nlargest on (score, string) tuples).
func closeMatches(word string, possibilities []string, n int, cutoff float64) []string {
	type scored struct {
		score float64
		s     string
	}
	w := []rune(word)
	var res []scored
	for _, x := range possibilities {
		if sc := ratio([]rune(x), w); sc >= cutoff {
			res = append(res, scored{sc, x})
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].score != res[j].score {
			return res[i].score > res[j].score
		}
		return res[i].s > res[j].s
	})
	if len(res) > n {
		res = res[:n]
	}
	out := make([]string, len(res))
	for i, r := range res {
		out[i] = r.s
	}
	return out
}
