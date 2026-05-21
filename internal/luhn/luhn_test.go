package luhn

import "testing"

func TestValid(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"0", true},
		{"12345678903", true},
		{"9278923470", true},
		{"79927398713", true},
		{"79927398710", false},
		{"abc", false},
		{"12345", false},
		{"4561261212345467", true},
		{"4561261212345464", false},
	}
	for _, c := range cases {
		if got := Valid(c.in); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
