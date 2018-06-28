package txhelpers

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg"
)

func TestBlockSubsidy(t *testing.T) {
	totalSubsidy := UltimateSubsidy(&chaincfg.MainNetParams)

	expectedSubsidy := int64(2043784000590848) + chaincfg.MainNetParams.BlockOneSubsidy()
	if totalSubsidy != expectedSubsidy {
		t.Errorf("Bad total subsidy; want %v, got %v", expectedSubsidy, totalSubsidy)
	}
}
