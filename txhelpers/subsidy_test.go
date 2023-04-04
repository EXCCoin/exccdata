// Copyright (c) 2018-2022, The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txhelpers

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg/v3"
)

func TestUltimateSubsidy(t *testing.T) {
	// Mainnet
	wantMainnetSubsidy := int64(3398978555227904)
	totalSubsidy := UltimateSubsidy(chaincfg.MainNetParams(), -1)

	if totalSubsidy != wantMainnetSubsidy {
		t.Errorf("Bad total subsidy; want %d, got %d",
			wantMainnetSubsidy, totalSubsidy)
	}

	// verify cache
	totalSubsidy2 := UltimateSubsidy(chaincfg.MainNetParams(), -1)
	if totalSubsidy != totalSubsidy2 {
		t.Errorf("Bad total subsidy; want %d, got %d",
			totalSubsidy, totalSubsidy2)
	}

	// Testnet
	wantTestnetSubsidy := int64(2685161595227904)
	totalTNSubsidy := UltimateSubsidy(chaincfg.TestNet3Params(), -1)

	if totalTNSubsidy != wantTestnetSubsidy {
		t.Errorf("Bad total subsidy; want %d, got %d",
			wantTestnetSubsidy, totalTNSubsidy)
	}

	// verify cache
	totalTNSubsidy2 := UltimateSubsidy(chaincfg.TestNet3Params(), -1)
	if totalTNSubsidy != totalTNSubsidy2 {
		t.Errorf("Bad total subsidy; want %d, got %d",
			totalTNSubsidy, totalTNSubsidy2)
	}

	// re-verify mainnet cache
	totalSubsidy3 := UltimateSubsidy(chaincfg.MainNetParams(), -1)
	if totalSubsidy != totalSubsidy3 {
		t.Errorf("Bad total subsidy; want %d, got %d",
			totalSubsidy, totalSubsidy3)
	}
}

func BenchmarkUltimateSubsidy(b *testing.B) {
	// warm up
	totalSubsidy := UltimateSubsidy(chaincfg.MainNetParams(), -1)
	// verify cache
	totalSubsidy2 := UltimateSubsidy(chaincfg.MainNetParams(), -1)
	if totalSubsidy != totalSubsidy2 {
		b.Errorf("Bad total subsidy; want %d, got %d",
			totalSubsidy, totalSubsidy2)
	}

	for i := 0; i < b.N; i++ {
		totalSubsidy = UltimateSubsidy(chaincfg.MainNetParams(), -1)
	}

	if totalSubsidy != totalSubsidy2 {
		b.Errorf("Bad total subsidy; want %d, got %d",
			totalSubsidy, totalSubsidy2)
	}
}
