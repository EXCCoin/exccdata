package txhelpers

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg"
)

func TestBlockSubsidy(t *testing.T) {
	totalSubsidy := UltimateSubsidy(&chaincfg.MainNetParams)

	if totalSubsidy != 3115773615157966 {
		t.Errorf("Bad total subsidy; want 3115773615157966, got %v", totalSubsidy)
	}
}
