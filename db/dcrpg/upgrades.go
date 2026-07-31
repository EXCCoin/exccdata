// Copyright (c) 2019-2021, The Decred developers
// See LICENSE for details.

package dcrpg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/EXCCoin/exccd/chaincfg/v3"
	"github.com/EXCCoin/exccdata/db/dcrpg/v8/internal"
	"github.com/EXCCoin/exccdata/v8/stakedb"
)

// The database schema is versioned in the meta table as follows.
const (
	// compatVersion indicates major DB changes for which there are no automated
	// upgrades. A complete DB rebuild is required if this version changes. This
	// should change very rarely, but when it does change all of the upgrades
	// defined here should be removed since they are no longer applicable.
	compatVersion = 2

	// schemaVersion pertains to a sequence of incremental upgrades to the
	// database schema that may be performed for the same compatibility version.
	// This includes changes such as creating tables, adding/deleting columns,
	// adding/deleting indexes or any other operations that create, delete, or
	// modify the definition of any database relation.
	schemaVersion = 1

	// maintVersion indicates when certain maintenance operations should be
	// performed for the same compatVersion and schemaVersion. Such operations
	// include duplicate row check and removal, forced table analysis, patching
	// or recomputation of data values, reindexing, or any other operations that
	// do not create, delete or modify the definition of any database relation.
	maintVersion = 0
)

var (
	targetDatabaseVersion = &DatabaseVersion{
		compat: compatVersion,
		schema: schemaVersion,
		maint:  maintVersion,
	}
)

// DatabaseVersion models a database version.
type DatabaseVersion struct {
	compat, schema, maint uint32
}

// String implements Stringer for DatabaseVersion.
func (v DatabaseVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.compat, v.schema, v.maint)
}

// NewDatabaseVersion returns a new DatabaseVersion with the version major.minor.patch
func NewDatabaseVersion(major, minor, patch uint32) DatabaseVersion {
	return DatabaseVersion{major, minor, patch}
}

// DBVersion retrieves the database version from the meta table. See
// (*DatabaseVersion).NeededToReach for version comparison.
func DBVersion(db *sql.DB) (ver DatabaseVersion, err error) {
	err = db.QueryRow(internal.SelectMetaDBVersions).Scan(&ver.compat, &ver.schema, &ver.maint)
	return
}

// CompatAction defines the action to be taken once the current and the required
// pg table versions are compared.
type CompatAction int8

// These are the recognized CompatActions for upgrading a database from one
// version to another.
const (
	Rebuild CompatAction = iota
	Upgrade
	Maintenance
	OK
	TimeTravel
	Unknown
)

// NeededToReach describes what action is required for the DatabaseVersion to
// reach another version provided in the input argument.
func (v *DatabaseVersion) NeededToReach(other *DatabaseVersion) CompatAction {
	switch {
	case v.compat < other.compat:
		return Upgrade
	case v.compat > other.compat:
		return TimeTravel
	case v.schema < other.schema:
		return Upgrade
	case v.schema > other.schema:
		return TimeTravel
	case v.maint < other.maint:
		return Maintenance
	case v.maint > other.maint:
		return TimeTravel
	default:
		return OK
	}
}

// String implements Stringer for CompatAction.
func (v CompatAction) String() string {
	actions := map[CompatAction]string{
		Rebuild:     "rebuild",
		Upgrade:     "upgrade",
		Maintenance: "maintenance",
		TimeTravel:  "time travel",
		OK:          "ok",
	}
	if actionStr, ok := actions[v]; ok {
		return actionStr
	}
	return "unknown"
}

// DatabaseUpgrade is used to define a required DB upgrade.
type DatabaseUpgrade struct {
	TableName               string
	UpgradeType             CompatAction
	CurrentVer, RequiredVer DatabaseVersion
}

// String implements Stringer for DatabaseUpgrade.
func (s DatabaseUpgrade) String() string {
	return fmt.Sprintf("Table %s requires %s (%s -> %s).", s.TableName,
		s.UpgradeType, s.CurrentVer, s.RequiredVer)
}

type metaData struct {
	netName         string
	currencyNet     uint32
	bestBlockHeight int64
	// bestBlockHash   dbtypes.ChainHash
	dbVer DatabaseVersion
	// ibdComplete bool
}

func initMetaData(db *sql.DB, meta *metaData) error {
	_, err := db.Exec(internal.InitMetaRow, meta.netName, meta.currencyNet,
		meta.bestBlockHeight, // meta.bestBlockHash,
		meta.dbVer.compat, meta.dbVer.schema, meta.dbVer.maint,
		false /* meta.ibdComplete */)
	return err
}

func updateSchemaVersion(db *sql.DB, schema uint32) error {
	_, err := db.Exec(internal.SetDBSchemaVersion, schema)
	return err
}

func updateMaintenanceVersion(db *sql.DB, maint uint32) error {
	_, err := db.Exec(internal.SetDBMaintenanceVersion, maint)
	return err
}

// Upgrader contains a number of elements necessary to perform a database
// upgrade.
type Upgrader struct {
	db      *sql.DB
	params  *chaincfg.Params
	bg      BlockGetter
	stakeDB *stakedb.StakeDatabase
	ctx     context.Context
}

// NewUpgrader is a contructor for an Upgrader.
func NewUpgrader(ctx context.Context, params *chaincfg.Params, db *sql.DB, bg BlockGetter, stakeDB *stakedb.StakeDatabase) *Upgrader {
	return &Upgrader{
		db:      db,
		params:  params,
		bg:      bg,
		stakeDB: stakeDB,
		ctx:     ctx,
	}
}

// UpgradeDatabase attempts to upgrade the given sql.DB with help from the
// BlockGetter. The DB version will be compared against the target version to
// decide what upgrade type to initiate.
func (u *Upgrader) UpgradeDatabase() (bool, error) {
	initVer, upgradeType, err := versionCheck(u.db)
	if err != nil {
		return false, err
	}

	switch upgradeType {
	case OK:
		return true, nil
	case Upgrade, Maintenance:
		// Automatic upgrade is supported. Attempt to upgrade from initVer ->
		// targetDatabaseVersion.
		return u.upgradeDatabase(*initVer, *targetDatabaseVersion)
	case TimeTravel:
		return false, fmt.Errorf("the current table version is newer than supported: "+
			"%v > %v", initVer, targetDatabaseVersion)
	case Unknown, Rebuild:
		fallthrough
	default:
		return false, fmt.Errorf("rebuild of entire database required")
	}
}

func (u *Upgrader) upgradeDatabase(current, target DatabaseVersion) (bool, error) {
	switch current.compat {
	case 1:
		return u.compatVersion1Upgrades(current, target)
	case 2:
		return u.compatVersion2Upgrades(current, target)
	default:
		return false, fmt.Errorf("unsupported DB compatibility version %d", current.compat)
	}
}

func (u *Upgrader) compatVersion2Upgrades(current, target DatabaseVersion) (bool, error) {
	upgradeCheck := func() (done bool, err error) {
		switch current.NeededToReach(&target) {
		case OK:
			// No upgrade needed.
			return true, nil
		case Upgrade, Maintenance:
			// Automatic upgrade is supported.
			return false, nil
		case TimeTravel:
			return false, fmt.Errorf("the current table version is newer than supported: "+
				"%v > %v", current, target)
		case Unknown, Rebuild:
			fallthrough
		default:
			return false, fmt.Errorf("rebuild of entire database required")
		}
	}

	// Initial upgrade status check.
	done, err := upgradeCheck()
	if done || err != nil {
		return done, err
	}

	// Process schema upgrades and table maintenance.
	// initSchema := current.schema
	switch current.schema {
	case 0:
		err = u.upgradeSchema0to1()
		if err != nil {
			return false, fmt.Errorf("failed to upgrade 2.0.0 to 2.1.0: %v", err)
		}
		current.schema++
		current.maint = 0
		if err = storeVers(u.db, &current); err != nil {
			return false, err
		}
		fallthrough
	case 1:
		return upgradeCheck()

	default:
		return false, fmt.Errorf("unsupported schema version %d", current.schema)
	}
}

func (u *Upgrader) compatVersion1Upgrades(current, target DatabaseVersion) (bool, error) {
	upgradeCheck := func() (done bool, err error) {
		switch current.NeededToReach(&target) {
		case OK:
			return true, nil
		case Upgrade, Maintenance:
			return false, nil
		case TimeTravel:
			return false, fmt.Errorf("the current table version is newer than supported: "+
				"%v > %v", current, target)
		default:
			return false, fmt.Errorf("rebuild of entire database required")
		}
	}

	done, err := upgradeCheck()
	if done || err != nil {
		return done, err
	}
	if current.schema != 11 || current.maint != 0 {
		return false, fmt.Errorf("automatic compatibility upgrade requires database version 1.11.0, got %v; run exccdata 6.2.1 first", current)
	}

	log.Infof("Performing TEXT->BYTEA migration (compat 1 -> 2). This will take a while on large databases...")

	tx, err := u.db.BeginTx(u.ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin TEXT->BYTEA migration: %v", err)
	}
	defer tx.Rollback()

	err = u.migrateTextToBytea(tx)
	if err != nil {
		return false, fmt.Errorf("failed TEXT->BYTEA migration: %v", err)
	}

	current.compat = 2
	current.schema = 0
	current.maint = 0
	target.compat = 2

	_, err = tx.Exec(`UPDATE meta SET compatibility_version = $1,
		schema_version = $2, maintenance_version = $3`, current.compat,
		current.schema, current.maint)
	if err != nil {
		return false, fmt.Errorf("failed to store version after compat upgrade: %v", err)
	}
	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit compat upgrade: %v", err)
	}

	log.Infof("TEXT->BYTEA migration complete. DB now at compat=2, schema=0.")

	return u.compatVersion2Upgrades(current, target)
}

func (u *Upgrader) migrateTextToBytea(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE FUNCTION pg_temp.exccdata_decode_hash(hash TEXT) RETURNS BYTEA
		LANGUAGE SQL IMMUTABLE STRICT PARALLEL SAFE AS $function$
			SELECT decode(hash, 'hex')
		$function$;

		CREATE FUNCTION pg_temp.exccdata_decode_hash_array(hashes TEXT[]) RETURNS BYTEA[]
		LANGUAGE SQL IMMUTABLE STRICT PARALLEL SAFE AS $function$
			SELECT coalesce(array_agg(pg_temp.exccdata_decode_hash(hash) ORDER BY ord), ARRAY[]::BYTEA[])
			FROM unnest(hashes) WITH ORDINALITY AS x(hash, ord)
		$function$;
	`)
	if err != nil {
		return fmt.Errorf("failed to create hash conversion functions: %v", err)
	}

	type migration struct {
		table string
		col   string
	}
	hashCols := []migration{
		{"blocks", "hash"},
		{"blocks", "previous_hash"},
		{"transactions", "block_hash"},
		{"transactions", "tx_hash"},
		{"vins", "tx_hash"},
		{"vins", "prev_tx_hash"},
		{"vouts", "tx_hash"},
		{"addresses", "tx_hash"},
		{"tickets", "tx_hash"},
		{"tickets", "block_hash"},
		{"votes", "tx_hash"},
		{"votes", "block_hash"},
		{"votes", "candidate_block_hash"},
		{"votes", "ticket_hash"},
		{"misses", "block_hash"},
		{"misses", "candidate_block_hash"},
		{"misses", "ticket_hash"},
		{"block_chain", "this_hash"},
		{"block_chain", "prev_hash"},
		{"swaps", "contract_tx"},
		{"swaps", "spend_tx"},
		{"treasury", "tx_hash"},
		{"treasury", "block_hash"},
	}

	for _, m := range hashCols {
		q := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s TYPE BYTEA USING pg_temp.exccdata_decode_hash(%s)`,
			m.table, m.col, m.col)
		log.Infof("Migrating %s.%s TEXT -> BYTEA...", m.table, m.col)
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed to migrate %s.%s: %v", m.table, m.col, err)
		}
	}

	nullableHashCols := []migration{
		{"addresses", "matching_tx_hash"},
		{"meta", "best_block_hash"},
		{"block_chain", "next_hash"},
	}
	for _, m := range nullableHashCols {
		q := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s TYPE BYTEA USING CASE WHEN %s = '' THEN NULL ELSE pg_temp.exccdata_decode_hash(%s) END`,
			m.table, m.col, m.col, m.col)
		log.Infof("Migrating %s.%s TEXT -> BYTEA (nullable)...", m.table, m.col)
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed to migrate %s.%s: %v", m.table, m.col, err)
		}
	}

	log.Infof("Migrating blocks.winners TEXT[] -> BYTEA[]...")
	_, err = tx.Exec(`ALTER TABLE blocks ALTER COLUMN winners TYPE BYTEA[]
		USING pg_temp.exccdata_decode_hash_array(winners)`)
	if err != nil {
		return fmt.Errorf("failed to migrate blocks.winners: %v", err)
	}
	_, err = tx.Exec(`ALTER TABLE vouts ALTER COLUMN script_addresses TYPE TEXT
		USING script_addresses::TEXT`)
	if err != nil {
		return fmt.Errorf("failed to migrate vouts.script_addresses: %v", err)
	}

	log.Infof("Dropping removed columns...")
	drops := []struct{ table, col string }{
		{"blocks", "tx"},
		{"blocks", "stx"},
		{"transactions", "time"},
		{"vouts", "pkscript"},
		{"vouts", "script_req_sigs"},
	}
	for _, d := range drops {
		q := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, d.table, d.col)
		log.Infof("Dropping %s.%s...", d.table, d.col)
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("failed to drop %s.%s: %v", d.table, d.col, err)
		}
	}

	_, err = tx.Exec(`DROP FUNCTION pg_temp.exccdata_decode_hash_array(TEXT[]),
		pg_temp.exccdata_decode_hash(TEXT)`)
	if err != nil {
		return fmt.Errorf("failed to drop hash conversion functions: %v", err)
	}

	return nil
}

func (u *Upgrader) upgradeSchema0to1() error {
	return nil
}

func storeVers(db *sql.DB, dbVer *DatabaseVersion) error {
	err := updateSchemaVersion(db, dbVer.schema)
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}
	err = updateMaintenanceVersion(db, dbVer.maint)
	if err != nil {
		return fmt.Errorf("failed to update maintenance version: %w", err)
	}
	return nil
}
