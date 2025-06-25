package types

import (
	"fmt"
	"strconv"
	"strings"
)

// EvmChainId extracts the middle number from chain id which is
// a string following the "ALPHABET_NUMBER-NUMBER" format
func EvmChainId(chainid string) (uint64, error) {
	if chainid == "" { // standalone cmd case
		return 0, nil
	}
	parts := strings.Split(chainid, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("missing '_' or no content after '_' in %s", chainid)
	}

	subParts := strings.Split(parts[1], "-")
	if len(subParts) < 2 {
		return 0, fmt.Errorf("missing '-' or no content after '-' in the second part of %s", chainid)
	}

	id, err := strconv.ParseUint(subParts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("number parsing failed: %s (cause: %w)", subParts[0], err)
	}

	return id, nil
}