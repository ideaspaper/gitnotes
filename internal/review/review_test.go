package review

import "testing"

func TestRangeInDiff(t *testing.T) {
	changed := map[int]int{10: 9, 11: 9, 12: 9}
	cases := []struct {
		name       string
		start, end int
		set        map[int]int
		want       bool
	}{
		{"single in", 11, 11, changed, true},
		{"single out", 5, 5, changed, false},
		{"range fully in", 10, 12, changed, true},
		{"range partially out", 11, 13, changed, false},
		{"empty changed set", 11, 11, map[int]int{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rangeInDiff(c.set, c.start, c.end); got != c.want {
				t.Errorf("rangeInDiff(%d,%d) = %v, want %v", c.start, c.end, got, c.want)
			}
		})
	}
}

func TestHunkHeaderParsing(t *testing.T) {

	cases := []struct {
		line                       string
		match                      bool
		oldStart, oldCount         int
		newStart, newCount, oldPos int
	}{
		{"@@ -1,3 +4,2 @@ func foo() {", true, 1, 3, 4, 2, 4},
		{"@@ -0,0 +1,234 @@", true, 0, 0, 1, 234, 0},
		{"@@ -10,2 +10,3 @@", true, 10, 2, 10, 3, 12},
		{"@@ -5 +5 @@", true, 5, 1, 5, 1, 6},
		{" not a hunk header", false, 0, 0, 0, 0, 0},
	}
	for _, c := range cases {
		m := hunkHeader.FindStringSubmatch(c.line)
		if (m != nil) != c.match {
			t.Errorf("match(%q) = %v, want %v", c.line, m != nil, c.match)
			continue
		}
		if m == nil {
			continue
		}
		oldStart := atoi(m[1])
		oldCount := countOr1(m[2])
		newStart := atoi(m[3])
		newCount := countOr1(m[4])
		if oldStart != c.oldStart || oldCount != c.oldCount || newStart != c.newStart || newCount != c.newCount {
			t.Errorf("%q parsed old=%d,%d new=%d,%d; want old=%d,%d new=%d,%d",
				c.line, oldStart, oldCount, newStart, newCount, c.oldStart, c.oldCount, c.newStart, c.newCount)
		}
		if got := oldStart + oldCount; got != c.oldPos {
			t.Errorf("oldPos for %q = %d, want %d", c.line, got, c.oldPos)
		}
	}
}

func TestGitlabLineCode(t *testing.T) {

	got := gitlabLineCode("detail.html", 0, 3)
	want := "bec3a43b20c5774df06ec073999a42d38ab7231a_0_3"
	if got != want {
		t.Errorf("gitlabLineCode = %q, want %q", got, want)
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
