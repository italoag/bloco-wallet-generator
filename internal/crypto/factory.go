package crypto

import (
	"fmt"
)

// GetAdapter returns the appropriate ChainAdapter for the given chain name
func GetAdapter(chain string) (ChainAdapter, error) {
	switch chain {
	case "ethereum", "eth":
		return NewEthereumAdapter(), nil
	case "bitcoin", "btc":
		// Will be implemented in Task 4
		return nil, fmt.Errorf("bitcoin adapter not yet implemented")
	case "solana", "sol":
		// Will be implemented in Task 5
		return nil, fmt.Errorf("solana adapter not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
}
