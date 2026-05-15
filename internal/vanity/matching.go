package vanity

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/sha3"

	"bloco-vgen/pkg/wallet"
)

func MatchesGenerationCriteria(address string, criteria wallet.GenerationCriteria) bool {
	return MatchesCriteria(address, criteria.Prefix, criteria.Suffix, criteria.IsChecksum, criteria.Network, criteria.CaseSensitive)
}

func MatchesCriteria(address, prefix, suffix string, isChecksum bool, network string, caseSensitivePattern ...bool) bool {
	addrWithoutPrefix := address
	if strings.HasPrefix(address, "0x") {
		addrWithoutPrefix = address[2:]
	}

	if isChecksum && (network == "ethereum" || network == "") {
		checksumAddress := ToChecksumAddress(address)
		if strings.HasPrefix(checksumAddress, "0x") {
			addrWithoutPrefix = checksumAddress[2:]
		} else {
			addrWithoutPrefix = checksumAddress
		}
	}

	caseSensitive := network == "bitcoin" || network == "solana"
	if len(caseSensitivePattern) > 0 && caseSensitivePattern[0] {
		caseSensitive = true
	}

	if prefix != "" {
		if len(addrWithoutPrefix) < len(prefix) {
			return false
		}
		prefixPart := addrWithoutPrefix[:len(prefix)]

		match := false
		if caseSensitive {
			match = prefixPart == prefix
		} else {
			match = strings.EqualFold(prefixPart, prefix)
		}

		if !match {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG: Prefix check failed: %q does not start with %q (case-sensitive: %v)\n",
					prefixPart, prefix, caseSensitive)
			}
			return false
		}
	}

	if suffix != "" {
		if len(addrWithoutPrefix) < len(suffix) {
			return false
		}
		suffixPart := addrWithoutPrefix[len(addrWithoutPrefix)-len(suffix):]

		match := false
		if caseSensitive {
			match = suffixPart == suffix
		} else {
			match = strings.EqualFold(suffixPart, suffix)
		}

		if !match {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG: Suffix check failed: %q does not end with %q (case-sensitive: %v)\n",
					suffixPart, suffix, caseSensitive)
			}
			return false
		}
	}

	if isChecksum && (prefix != "" || suffix != "") {
		if network == "ethereum" || network == "" {
			result := IsEIP55Checksum(address, prefix, suffix)
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG: EIP55 validation result: %v\n", result)
			}
			return result
		}
		return true
	}

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG: Address validation passed\n")
	}
	return true
}

func ToChecksumAddress(address string) string {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	addrWithoutPrefix := strings.ToLower(address[2:])
	addrBytes := []byte(addrWithoutPrefix)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(addrBytes)
	hash := hasher.Sum(nil)

	var result strings.Builder
	result.WriteString("0x")

	for i, char := range addrWithoutPrefix {
		if char >= '0' && char <= '9' {
			result.WriteByte(byte(char))
		} else if char >= 'a' && char <= 'f' {
			hashByte := hash[i/2]
			var hashBit uint8
			if i%2 == 0 {
				hashBit = hashByte >> 4
			} else {
				hashBit = hashByte & 0x0f
			}

			if hashBit >= 8 {
				result.WriteByte(byte(char - 32))
			} else {
				result.WriteByte(byte(char))
			}
		}
	}

	return result.String()
}

func IsEIP55Checksum(address, prefix, suffix string) bool {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	checksumAddr := ToChecksumAddress(address)

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG EIP55: Original=%s Checksum=%s Prefix=%q Suffix=%q\n",
			address, checksumAddr, prefix, suffix)
	}

	if prefix != "" {
		prefixPart := checksumAddr[2 : 2+len(prefix)]
		if !strings.EqualFold(prefixPart, prefix) {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Prefix failed - got %q expected %q\n", prefixPart, prefix)
			}
			return false
		}
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG EIP55: Prefix matched - got %q expected %q\n", prefixPart, prefix)
		}
	}

	if suffix != "" {
		suffixStart := len(checksumAddr) - len(suffix)
		if suffixStart < 2 {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Suffix too long for address\n")
			}
			return false
		}
		suffixPart := checksumAddr[suffixStart:]
		if !strings.EqualFold(suffixPart, suffix) {
			if os.Getenv("BLOCO_DEBUG") != "" {
				fmt.Printf("DEBUG EIP55: Suffix failed - got %q expected %q\n", suffixPart, suffix)
			}
			return false
		}
		if os.Getenv("BLOCO_DEBUG") != "" {
			fmt.Printf("DEBUG EIP55: Suffix matched - got %q expected %q\n", suffixPart, suffix)
		}
	}

	if os.Getenv("BLOCO_DEBUG") != "" {
		fmt.Printf("DEBUG EIP55: Validation passed\n")
	}
	return true
}
