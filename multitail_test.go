package multitail

import "testing"

func TestGlobNoMatch(t *testing.T) {
	_, err := OpenGlob("/no/such/dir/*", Config{})
	if err == nil {
		t.Error("OpenGlob should error if a provided glob does not match any files")
	}
}
