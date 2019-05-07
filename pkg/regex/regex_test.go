package regex

import (
	"reflect"
	"testing"
)

func Test_SearchIP(t *testing.T) {
	cases := []struct {
		s      string
		search []string
	}{
		{
			"dsa dskd 2 | 32.43 210.125.56.230 dsad",
			[]string{"210.125.56.230"},
		},
		{
			"8.125.56.",
			nil,
		},
		{
			"8.125.56.2",
			[]string{"8.125.56.2"},
		},
		{
			"8.125.56.2 8.125.56.a8.125.56.2",
			[]string{"8.125.56.2", "8.125.56.2"},
		},
	}
	for _, c := range cases {
		out := SearchIP(c.s)
		if !reflect.DeepEqual(out, c.search) {
			t.Errorf("SearchIP(%s) == %v want %v", c.s, out, c.search)
		}
	}
}

func Test_MatchEmail(t *testing.T) {
	cases := []struct {
		s     string
		valid bool
	}{
		{
			"a@a.com",
			true,
		},
		{
			"a@a",
			false,
		},
		{
			"a.a@a.a.com",
			true,
		},
	}
	for _, c := range cases {
		out := MatchEmail(c.s)
		if out != c.valid {
			t.Errorf("MatchEmail(%s) == %t want %t", c.s, out, c.valid)
		}
	}
}

func Test_MatchDomain(t *testing.T) {
	cases := []struct {
		s     string
		valid bool
	}{
		{
			"example.com",
			true,
		},
		{
			"example._a",
			false,
		},
		{
			"press.example.com",
			true,
		},
		{
			"press.example.com/test",
			false,
		},
	}
	for _, c := range cases {
		out := MatchDomain(c.s)
		if out != c.valid {
			t.Errorf("MatchDomain(%s) == %t want %t", c.s, out, c.valid)
		}
	}
}
