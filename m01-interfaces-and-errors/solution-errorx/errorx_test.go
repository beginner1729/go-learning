package errorx

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCodeThroughWrapping(t *testing.T) {
	base := errors.New("customer: not found")
	coded := WithCode(base, CodeNotFound)
	wrapped := fmt.Errorf("byID %q: %w", "c1", coded) // wrap again on top

	if got := Code(wrapped); got != CodeNotFound {
		t.Fatalf("Code through wrapping = %q, want %q", got, CodeNotFound)
	}
	// errors.Is must still reach the original cause through our wrapper.
	if !errors.Is(wrapped, base) {
		t.Fatalf("errors.Is lost the cause through codedError")
	}
}

func TestHTTPStatus(t *testing.T) {
	cases := map[string]int{
		CodeNotFound:   http.StatusNotFound,
		CodeValidation: http.StatusBadRequest,
		CodeConflict:   http.StatusConflict,
		CodeInternal:   http.StatusInternalServerError,
	}
	for code, want := range cases {
		err := WithCode(errors.New("x"), code)
		if got := HTTPStatus(err); got != want {
			t.Fatalf("HTTPStatus(%s) = %d, want %d", code, got, want)
		}
	}
	// Unknown/untagged error defaults to 500.
	if got := HTTPStatus(errors.New("bare")); got != http.StatusInternalServerError {
		t.Fatalf("bare error = %d, want 500", got)
	}
}

func TestWithCodeNil(t *testing.T) {
	if WithCode(nil, CodeInternal) != nil {
		t.Fatal("WithCode(nil) should be nil")
	}
}
