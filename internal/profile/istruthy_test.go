package profile

import "testing"

func TestIsTruthy(t *testing.T) {
	truthy := []string{"1", "true", "True", "TRUE", "yes", "YES", "Yes", "on", "ON", "On"}
	falsy := []string{"0", "false", "False", "FALSE", "no", "off", "", "maybe", "2"}

	for _, v := range truthy {
		if !isTruthy(v) {
			t.Errorf("isTruthy(%q) = false, want true", v)
		}
	}
	for _, v := range falsy {
		if isTruthy(v) {
			t.Errorf("isTruthy(%q) = true, want false", v)
		}
	}
}

func TestIsTruthy_TrimsWhitespace(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"  true  ", true},
		{"\ttrue\n", true},
		{"  false  ", false},
		{"  1  ", true},
	}
	for _, c := range cases {
		got := isTruthy(c.input)
		if got != c.want {
			t.Errorf("isTruthy(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
