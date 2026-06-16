package greeter

import "testing"

// A test lives next to the code it tests, in a _test.go file, in the same
// package. `go test` finds and runs every func named TestXxx(*testing.T).
func TestGreet(t *testing.T) {
	cases := map[string]string{
		"Ada": "hello, Ada",
		"":    "hello, world",
	}
	for in, want := range cases {
		if got := Greet(in); got != want {
			t.Errorf("Greet(%q) = %q, want %q", in, got, want)
		}
	}
}
