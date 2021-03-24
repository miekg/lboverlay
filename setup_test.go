package lboverlay

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetupLboverlay(t *testing.T) {
	tests := []struct {
		input     string
		shouldErr bool
	}{
		{`lboverlay`, false},
		{`lboverlay a`, false},
		{`lboverlay a b`, true},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		_, err := parse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found none for input %s", i, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
			}
		}
	}
}
