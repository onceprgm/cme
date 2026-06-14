package java

import "testing"

func TestParseMajor(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"1.8.0_292", 8},
		{"1.8.0_452", 8},
		{"1.7.0_80", 7},
		{"17.0.1", 17},
		{"17", 17},
		{"21.0.2", 21},
		{"21", 21},
		{"11.0.19", 11},
		{"16.0.1", 16},
		{"garbage", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := parseMajor(c.in); got != c.want {
			t.Errorf("parseMajor(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	cases := []struct {
		have, want int
		ok         bool
	}{
		{17, 17, true},
		{21, 17, true},
		{8, 17, false},
		{16, 17, false},
	}
	for _, c := range cases {
		if got := satisfies(c.have, c.want); got != c.ok {
			t.Errorf("satisfies(%d, %d) = %v, want %v", c.have, c.want, got, c.ok)
		}
	}
}
