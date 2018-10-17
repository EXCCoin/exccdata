package exccsqlite

import (
	"testing"

	"github.com/EXCCoin/exccd/chaincfg"
	"github.com/EXCCoin/exccdata/testutil"
)

func TestIsZeroHashP2PHKAddress(t *testing.T) {
	t.SkipNow()

	testutil.BindCurrentTestSetup(t)

	mainnetDummy := "DsQxuVRvS4eaJ42dhQEsCXauMWjvopWgrVg"
	testnetDummy := "TsR28UZRprhgQQhzWns2M6cAwchrNVvbYq2"
	simnetDummy := "SsUMGgvWLcixEeHv3GT4TGYyez4kY79RHth"

	positiveTest := true
	negativeTest := !positiveTest

	testIsZeroHashP2PHKAddress(mainnetDummy, &chaincfg.MainNetParams, positiveTest)
	testIsZeroHashP2PHKAddress(testnetDummy, &chaincfg.TestNetParams, positiveTest)
	testIsZeroHashP2PHKAddress(simnetDummy, &chaincfg.SimNetParams, positiveTest)

	// wrong network
	testIsZeroHashP2PHKAddress(mainnetDummy, &chaincfg.SimNetParams, negativeTest)
	testIsZeroHashP2PHKAddress(testnetDummy, &chaincfg.MainNetParams, negativeTest)
	testIsZeroHashP2PHKAddress(simnetDummy, &chaincfg.TestNetParams, negativeTest)

	// wrong address
	testIsZeroHashP2PHKAddress("", &chaincfg.SimNetParams, negativeTest)
	testIsZeroHashP2PHKAddress("", &chaincfg.MainNetParams, negativeTest)
	testIsZeroHashP2PHKAddress("", &chaincfg.TestNetParams, negativeTest)

}
func testIsZeroHashP2PHKAddress(expectedAddress string, params *chaincfg.Params, expectedTestResult bool) {
	result := IsZeroHashP2PHKAddress(expectedAddress, params)
	if expectedTestResult != result {
		testutil.ReportTestFailed(
			"IsZeroHashP2PHKAddress(%v) returned <%v>, expected <%v>",
			expectedAddress,
			result,
			expectedTestResult)
	}
}
