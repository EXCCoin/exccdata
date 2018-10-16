package explorer

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg"
)

func TestTestNetName(t *testing.T) {
	netName := netName(&chaincfg.TestNetParams)
	if netName != "Testnet" {
		t.Errorf(`Net name not "Testnet": %s`, netName)
	}
}

func TestMainNetName(t *testing.T) {
	netName := netName(&chaincfg.MainNetParams)
	if netName != "Mainnet" {
		t.Errorf(`Net name not "Mainnet": %s`, netName)
	}
}

func TestSimNetName(t *testing.T) {
	netName := netName(&chaincfg.SimNetParams)
	if netName != "Simnet" {
		t.Errorf(`Net name not "Simnet": %s`, netName)
	}
}
