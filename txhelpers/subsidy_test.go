package txhelpers

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg"
)

func TestBlockSubsidy(t *testing.T) {
	totalSubsidy := UltimateSubsidy(&chaincfg.MainNetParams)

	if totalSubsidy != 3200307811695360 {
		t.Errorf("Bad total subsidy; want 3200307811695360, got %v", totalSubsidy)
	}
}
