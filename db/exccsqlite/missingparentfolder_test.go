package exccsqlite

import (
	"path/filepath"
	"testing"

	"github.com/EXCCoin/exccdata/testutil"
)

// TestMissingParentFolder ensures InitDB() is able to create
// a new DB-file parent directory if necessary
// See https://github.com/EXCCoin/exccdata/issues/515
func TestMissingParentFolder(t *testing.T) {
	testutil.BindCurrentTestSetup(t)
	testMissingParentFolder()
}

func testMissingParentFolder() {
	testName := testutil.TestName()
	testutil.ResetTempFolder(&testName)
	// Specify DB file in non-existent path
	target := filepath.Join(testName, "x", "y", "z", testutil.DefaultDBFileName)
	targetDBFile := testutil.FilePathInsideTempDir(target)

	InitTestDB(targetDBFile)
}
