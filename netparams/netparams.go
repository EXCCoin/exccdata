// Copyright (c) 2018 The ExchangeCoin team
// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2016-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netparams

import "github.com/EXCCoin/exccd/chaincfg"

// Params is used to group parameters for various networks such as the main
// network and test networks.
type Params struct {
	*chaincfg.Params
	JSONRPCClientPort string
	JSONRPCServerPort string
	GRPCServerPort    string
}

// MainNetParams contains parameters specific running exccwallet and
// exccd on the main network (wire.MainNet).
var MainNetParams = Params{
	Params:            &chaincfg.MainNetParams,
	JSONRPCClientPort: "9109",
	JSONRPCServerPort: "9110",
	GRPCServerPort:    "9111",
}

// TestNetParams contains parameters specific running exccwallet and
// exccd on the test network.
var TestNetParams = Params{
	Params:            &chaincfg.TestNetParams,
	JSONRPCClientPort: "19109",
	JSONRPCServerPort: "19110",
	GRPCServerPort:    "19111",
}

// SimNetParams contains parameters specific to the simulation test network
// (wire.SimNet).
var SimNetParams = Params{
	Params:            &chaincfg.SimNetParams,
	JSONRPCClientPort: "19556",
	JSONRPCServerPort: "19557",
	GRPCServerPort:    "19558",
}
