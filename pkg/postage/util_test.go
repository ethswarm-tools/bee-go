package postage_test

import (
	"math/big"
	"testing"

	"github.com/ethersphere/bee-go/pkg/postage"
)

func TestStampMath(t *testing.T) {
	// Stamp Math Tests
	cost := postage.GetStampCost(20, big.NewInt(100))
	// 2^20 * 100 = 1048576 * 100 = 104857600
	if cost.Int64() != 104857600 {
		t.Errorf("GetStampCost = %d, want 104857600", cost.Int64())
	}

	theoretical := postage.GetStampTheoreticalBytes(20)
	if theoretical != 4096*1048576 {
		t.Errorf("GetStampTheoreticalBytes = %d", theoretical)
	}
}
