//go:build migration

package dcrpg

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/EXCCoin/exccdata/v8/db/dbtypes"
)

func TestCompatVersion1Upgrade(t *testing.T) {
	dsn := os.Getenv("EXCCDATA_TEST_PG")
	if dsn == "" {
		t.Skip("EXCCDATA_TEST_PG is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	for _, stmt := range []string{
		`CREATE TEMP TABLE blocks (
			id SERIAL8 PRIMARY KEY, hash TEXT, height INT4, size INT4,
			is_valid BOOLEAN, is_mainchain BOOLEAN, version INT4,
			numtx INT4, num_rtx INT4, tx INT, txdbids INT8[],
			num_stx INT4, stx INT, stxdbids INT8[], time TIMESTAMPTZ,
			nonce INT8, vote_bits INT2, voters INT2, fresh_stake INT2,
			revocations INT2, pool_size INT4, bits INT4, sbits INT8,
			difficulty FLOAT8, stake_version INT4, previous_hash TEXT,
			chainwork TEXT, winners TEXT[]
		)`,
		`CREATE TEMP TABLE transactions (
			id SERIAL8 PRIMARY KEY, block_hash TEXT, block_height INT8,
			block_time TIMESTAMPTZ, time TIMESTAMPTZ, tx_type INT4,
			version INT4, tree INT2, tx_hash TEXT, block_index INT4,
			lock_time INT4, expiry INT4, size INT4, spent INT8, sent INT8,
			fees INT8, mix_count INT4, mix_denom INT8, num_vin INT4,
			vin_db_ids INT8[], num_vout INT4, vout_db_ids INT8[],
			is_valid BOOLEAN, is_mainchain BOOLEAN)`,
		`CREATE TEMP TABLE vins (tx_hash TEXT, prev_tx_hash TEXT)`,
		`CREATE TEMP TABLE vouts (tx_hash TEXT, pkscript TEXT, script_req_sigs INT, script_addresses TEXT[])`,
		`CREATE TEMP TABLE addresses (tx_hash TEXT, matching_tx_hash TEXT)`,
		`CREATE TEMP TABLE tickets (tx_hash TEXT, block_hash TEXT)`,
		`CREATE TEMP TABLE votes (tx_hash TEXT, block_hash TEXT, candidate_block_hash TEXT, ticket_hash TEXT)`,
		`CREATE TEMP TABLE misses (block_hash TEXT, candidate_block_hash TEXT, ticket_hash TEXT)`,
		`CREATE TEMP TABLE block_chain (this_hash TEXT, prev_hash TEXT, next_hash TEXT)`,
		`CREATE TEMP TABLE swaps (contract_tx TEXT, spend_tx TEXT)`,
		`CREATE TEMP TABLE treasury (tx_hash TEXT, block_hash TEXT)`,
		`CREATE TEMP TABLE stats (blocks_id INT8, pool_size INT8, pool_val INT8)`,
		`CREATE TEMP TABLE meta (best_block_hash TEXT, compatibility_version INT4, schema_version INT4, maintenance_version INT4)`,
		`INSERT INTO blocks (hash, previous_hash, winners, tx, stx) VALUES (
			'000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f',
			'202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f',
			ARRAY[
				'000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f',
				'202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f'
			], 1, 2)`,
		`INSERT INTO addresses VALUES ('00112233', '')`,
		`INSERT INTO vouts VALUES ('00112233', 'script', 1, ARRAY['addr1', 'addr2'])`,
		`INSERT INTO swaps VALUES ('00112233', 'aabbccdd')`,
		`INSERT INTO treasury VALUES ('00112233', 'aabbccdd')`,
		`INSERT INTO meta VALUES (NULL, 1, 11, 0)`,
	} {
		if _, err = db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	done, err := (&Upgrader{db: db, ctx: context.Background()}).compatVersion1Upgrades(
		DatabaseVersion{compat: 1, schema: 11}, DatabaseVersion{compat: 2, schema: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("upgrade did not reach the target version")
	}

	var hash, previousHash string
	var winners string
	err = db.QueryRow(`SELECT encode(hash, 'hex'), encode(previous_hash, 'hex'),
		array_to_string(ARRAY(SELECT encode(x, 'hex') FROM unnest(winners) x), ',') FROM blocks`).
		Scan(&hash, &previousHash, &winners)
	if err != nil {
		t.Fatal(err)
	}
	const hashText = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	const previousHashText = "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"
	if hash != hashText || previousHash != previousHashText {
		t.Fatalf("incorrect byte order: hash=%s previous_hash=%s", hash, previousHash)
	}
	if winners != hashText+","+previousHashText {
		t.Fatalf("incorrect winners: %v", winners)
	}
	var scannedHash dbtypes.ChainHash
	if err = db.QueryRow(`SELECT hash FROM blocks`).Scan(&scannedHash); err != nil {
		t.Fatal(err)
	}
	if scannedHash.String() != hashText {
		t.Fatalf("migrated hash scanned as %s, want %s", scannedHash.String(), hashText)
	}
	blockID, err := InsertBlock(db, &dbtypes.Block{
		Hash:         scannedHash,
		PreviousHash: scannedHash,
		Winners:      []dbtypes.ChainHash{scannedHash},
	}, true, true, false)
	if err != nil {
		t.Fatalf("insert block with BYTEA winners: %v", err)
	}
	if _, err = db.Exec(`INSERT INTO stats VALUES ($1, 1, 1)`, blockID); err != nil {
		t.Fatal(err)
	}
	if _, err = RetrieveBlockSummaryByHash(context.Background(), db, scannedHash); err != nil {
		t.Fatalf("retrieve block summary by hash: %v", err)
	}

	var matchingHash interface{}
	if err = db.QueryRow(`SELECT matching_tx_hash FROM addresses`).Scan(&matchingHash); err != nil {
		t.Fatal(err)
	}
	if matchingHash != nil {
		t.Fatalf("empty nullable hash became %v instead of NULL", matchingHash)
	}

	var addresses string
	if err = db.QueryRow(`SELECT script_addresses FROM vouts`).Scan(&addresses); err != nil {
		t.Fatal(err)
	}
	if addresses != "{addr1,addr2}" {
		t.Fatalf("incorrect script_addresses: %s", addresses)
	}

	var compat, schema, maint uint32
	if err = db.QueryRow(`SELECT compatibility_version, schema_version, maintenance_version FROM meta`).
		Scan(&compat, &schema, &maint); err != nil {
		t.Fatal(err)
	}
	if compat != 2 || schema != 1 || maint != 0 {
		t.Fatalf("incorrect database version: %d.%d.%d", compat, schema, maint)
	}

	var blockHash, txHash dbtypes.ChainHash
	for i := range blockHash {
		blockHash[i] = byte(i)
		txHash[i] = byte(i + 1)
	}
	wantTx := &dbtypes.Tx{
		BlockHash:        blockHash,
		BlockHeight:      123,
		BlockTime:        dbtypes.NewTimeDefFromUNIX(123456789),
		TxID:             txHash,
		VinDbIds:         []uint64{1, 2},
		VoutDbIds:        []uint64{3},
		IsValid:          true,
		IsMainchainBlock: true,
	}
	if _, err = InsertTx(db, wantTx, false, false); err != nil {
		t.Fatal(err)
	}
	_, gotTx, err := RetrieveDbTxByHash(context.Background(), db, txHash)
	if err != nil {
		t.Fatal(err)
	}
	if gotTx.TxID != wantTx.TxID || gotTx.BlockHash != wantTx.BlockHash ||
		gotTx.BlockHeight != wantTx.BlockHeight {
		t.Fatalf("transaction did not survive round trip: %#v", gotTx)
	}
}
