package engine

import (
	"fmt"
	"strings"

	"bloco-vgen/pkg/wallet"
)

func generateSyntheticAddresses(count int, criteria wallet.GenerationCriteria) []byte {
	if count <= 0 {
		count = 1
	}

	addresses := make([]byte, count*20)
	for i := 0; i < count; i++ {
		offset := i * 20
		for j := 0; j < 20; j++ {
			addresses[offset+j] = byte((i*37 + j*17 + 11) & 0xff)
		}
	}

	if count > 0 {
		applyPatternToSyntheticAddress(addresses[:20], criteria)
	}
	if count > 1 {
		applyPatternToSyntheticAddress(addresses[20:40], wallet.GenerationCriteria{Prefix: criteria.Prefix})
	}
	if count > 2 {
		applyPatternToSyntheticAddress(addresses[40:60], wallet.GenerationCriteria{Suffix: criteria.Suffix})
	}
	return addresses
}

func applyPatternToSyntheticAddress(address []byte, criteria wallet.GenerationCriteria) {
	if prefix, err := patternNibbles(criteria.Prefix); err == nil {
		for i, nibble := range prefix {
			setAddressNibble(address, i, nibble)
		}
	}
	if suffix, err := patternNibbles(criteria.Suffix); err == nil {
		start := 40 - len(suffix)
		for i, nibble := range suffix {
			setAddressNibble(address, start+i, nibble)
		}
	}
}

func setAddressNibble(address []byte, nibbleIndex int, nibble byte) {
	if nibbleIndex < 0 || nibbleIndex >= 40 {
		return
	}
	byteIndex := nibbleIndex / 2
	if nibbleIndex%2 == 0 {
		address[byteIndex] = (address[byteIndex] & 0x0f) | (nibble << 4)
		return
	}
	address[byteIndex] = (address[byteIndex] & 0xf0) | nibble
}

func patternNibbles(pattern string) ([]byte, error) {
	pattern = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(pattern)), "0x")
	if pattern == "" {
		return nil, nil
	}
	nibbles := make([]byte, len(pattern))
	for i := range pattern {
		switch {
		case pattern[i] >= '0' && pattern[i] <= '9':
			nibbles[i] = pattern[i] - '0'
		case pattern[i] >= 'a' && pattern[i] <= 'f':
			nibbles[i] = pattern[i] - 'a' + 10
		default:
			return nil, fmt.Errorf("invalid hex pattern %q", pattern)
		}
	}
	return nibbles, nil
}

func countAddressMatches(addresses []byte, prefix []byte, suffix []byte) uint32 {
	count := len(addresses) / 20
	matches := uint32(0)
	for i := 0; i < count; i++ {
		address := addresses[i*20 : (i+1)*20]
		if addressMatches(address, prefix, suffix) {
			matches++
		}
	}
	return matches
}

func addressMatches(address []byte, prefix []byte, suffix []byte) bool {
	for i, nibble := range prefix {
		if addressNibble(address, i) != nibble {
			return false
		}
	}
	start := 40 - len(suffix)
	for i, nibble := range suffix {
		if addressNibble(address, start+i) != nibble {
			return false
		}
	}
	return true
}

func addressNibble(address []byte, nibbleIndex int) byte {
	value := address[nibbleIndex/2]
	if nibbleIndex%2 == 0 {
		return (value >> 4) & 0x0f
	}
	return value & 0x0f
}
