package wallet

import "testing"

// Regression tests for GitHub issue #21: an unknown --network value used to
// fall through to the Ethereum generator, producing a real private key for a
// network the user did not ask for and that no backup path would persist.

func TestValidateRejectsUnsupportedNetwork(t *testing.T) {
	tests := []struct {
		name      string
		network   string
		wantError bool
	}{
		{"empty defaults to ethereum", "", false},
		{"ethereum", "ethereum", false},
		{"bitcoin", "bitcoin", false},
		{"solana", "solana", false},
		{"typo", "etherium", true},
		{"unsupported chain", "polygon", true},
		{"arbitrary text", "not-a-network", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria := GenerationCriteria{Network: tt.network}

			err := criteria.Validate()
			if tt.wantError && err == nil {
				t.Errorf("expected network %q to be rejected", tt.network)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected network %q to be accepted, got %v", tt.network, err)
			}
		})
	}
}

func TestIsSupportedNetwork(t *testing.T) {
	if !IsSupportedNetwork("") {
		t.Error("the empty network means the ethereum default and must be supported")
	}
	for _, network := range SupportedNetworks {
		if !IsSupportedNetwork(network) {
			t.Errorf("%q is listed in SupportedNetworks but reported unsupported", network)
		}
	}
	if IsSupportedNetwork("dogecoin") {
		t.Error("an unlisted network must not be reported as supported")
	}
}
