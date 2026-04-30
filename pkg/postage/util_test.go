package postage_test

import (
	"math/big"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/postage"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
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

func TestGetStampEffectiveBytes_Breakpoints(t *testing.T) {
	if got := postage.GetStampEffectiveBytes(16); got != 0 {
		t.Errorf("depth<17 should be 0, got %d", got)
	}
	// depth 22 -> 7.07 GB
	if got := postage.GetStampEffectiveBytes(22); got != int64(7.07*1_000_000_000) {
		t.Errorf("depth=22 = %d", got)
	}
	// depth 30 (in table) -> 3810 GB
	if got := postage.GetStampEffectiveBytes(30); got != int64(3810*1_000_000_000) {
		t.Errorf("depth=30 = %d", got)
	}
}

func TestGetStampDuration_AndAmountForDuration(t *testing.T) {
	// amount=2000, blockTime=5s, pricePerBlock=1 PLUR -> 10000s
	amount := big.NewInt(2000)
	d := postage.GetStampDuration(amount, 1, 5)
	if d.ToSeconds() != 10000 {
		t.Errorf("duration = %d, want 10000", d.ToSeconds())
	}

	// inverse: duration=10000s -> amount = (10000/5)*1 + 1 = 2001
	got := postage.GetAmountForDuration(swarm.DurationFromSeconds(10000), 1, 5)
	if got.Int64() != 2001 {
		t.Errorf("amount = %d, want 2001 (rounded up)", got.Int64())
	}
}

func TestGetDepthForSize(t *testing.T) {
	// 1MB fits in depth 18 (6.09 MB).
	mb, _ := swarm.SizeFromMegabytes(1)
	if got := postage.GetDepthForSize(mb); got != 18 {
		t.Errorf("1MB -> depth %d, want 18", got)
	}
	// 100 GB > depth 25 (96.5 GB), so first fit is depth 26 (208.52 GB).
	gb100, _ := swarm.SizeFromGigabytes(100)
	if got := postage.GetDepthForSize(gb100); got != 26 {
		t.Errorf("100GB -> depth %d, want 26", got)
	}
	// Tiny size (1 byte) still picks depth 17.
	tiny, _ := swarm.SizeFromBytes(1)
	if got := postage.GetDepthForSize(tiny); got != 17 {
		t.Errorf("1 byte -> depth %d, want 17", got)
	}
}
