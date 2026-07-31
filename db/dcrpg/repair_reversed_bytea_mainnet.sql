\set ON_ERROR_STOP on

-- Repairs databases migrated by a build that reversed hash bytes before
-- storing them even though ChainHash.Scan already handles the database byte
-- order. Run only with exccdata stopped.

BEGIN;
SET LOCAL statement_timeout = 0;
SET LOCAL lock_timeout = '30s';

CREATE FUNCTION pg_temp.exccdata_reverse_hash(hash BYTEA) RETURNS BYTEA
LANGUAGE SQL IMMUTABLE STRICT PARALLEL SAFE AS $function$
	SELECT decode(coalesce(string_agg(
		lpad(to_hex(get_byte(hash, i)), 2, '0'), '' ORDER BY i DESC), ''), 'hex')
	FROM generate_series(0, length(hash) - 1) AS i
$function$;

CREATE FUNCTION pg_temp.exccdata_reverse_hash_array(hashes BYTEA[]) RETURNS BYTEA[]
LANGUAGE SQL IMMUTABLE STRICT PARALLEL SAFE AS $function$
	SELECT coalesce(array_agg(pg_temp.exccdata_reverse_hash(hash) ORDER BY ord), ARRAY[]::BYTEA[])
	FROM unnest(hashes) WITH ORDINALITY AS x(hash, ord)
$function$;

DO $guard$
DECLARE
	compat INTEGER;
	schema INTEGER;
	maint INTEGER;
	genesis TEXT;
BEGIN
	SELECT compatibility_version, schema_version, maintenance_version
	INTO STRICT compat, schema, maint FROM meta;
	IF compat <> 2 OR schema <> 1 OR maint <> 0 THEN
		RAISE EXCEPTION 'expected database version 2.1.0, got %.%.%', compat, schema, maint;
	END IF;

	SELECT encode(hash, 'hex') INTO STRICT genesis
	FROM blocks WHERE height = 0 AND is_mainchain;
	IF genesis = '5f91ddfa5e9ffc4b837b81035dd0b0ddf1a1c59116aa5f5ef5f4121a64a69478' THEN
		RAISE EXCEPTION 'hash byte order is already correct; repair not applied';
	END IF;
	IF genesis <> '7894a6641a12f4f55e5faa1691c5a1f1ddb0d05d03817b834bfc9f5efadd915f' THEN
		RAISE EXCEPTION 'unexpected mainnet genesis bytes: %', genesis;
	END IF;
END
$guard$;

\echo Repairing blocks...
ALTER TABLE blocks
	ALTER COLUMN hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(hash),
	ALTER COLUMN previous_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(previous_hash),
	ALTER COLUMN winners TYPE BYTEA[] USING pg_temp.exccdata_reverse_hash_array(winners);

\echo Repairing transactions...
ALTER TABLE transactions
	ALTER COLUMN block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(block_hash),
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash);

\echo Repairing vins...
ALTER TABLE vins
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash),
	ALTER COLUMN prev_tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(prev_tx_hash);

\echo Repairing vouts...
ALTER TABLE vouts
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash);

\echo Repairing addresses...
ALTER TABLE addresses
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash),
	ALTER COLUMN matching_tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(matching_tx_hash);

\echo Repairing tickets...
ALTER TABLE tickets
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash),
	ALTER COLUMN block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(block_hash);

\echo Repairing votes...
ALTER TABLE votes
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash),
	ALTER COLUMN block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(block_hash),
	ALTER COLUMN candidate_block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(candidate_block_hash),
	ALTER COLUMN ticket_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(ticket_hash);

\echo Repairing misses...
ALTER TABLE misses
	ALTER COLUMN block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(block_hash),
	ALTER COLUMN candidate_block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(candidate_block_hash),
	ALTER COLUMN ticket_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(ticket_hash);

\echo Repairing block_chain...
ALTER TABLE block_chain
	ALTER COLUMN this_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(this_hash),
	ALTER COLUMN prev_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(prev_hash),
	ALTER COLUMN next_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(next_hash);

\echo Repairing swaps...
ALTER TABLE swaps
	ALTER COLUMN contract_tx TYPE BYTEA USING pg_temp.exccdata_reverse_hash(contract_tx),
	ALTER COLUMN spend_tx TYPE BYTEA USING pg_temp.exccdata_reverse_hash(spend_tx);

\echo Repairing treasury...
ALTER TABLE treasury
	ALTER COLUMN tx_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(tx_hash),
	ALTER COLUMN block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(block_hash);

\echo Repairing meta...
ALTER TABLE meta
	ALTER COLUMN best_block_hash TYPE BYTEA USING pg_temp.exccdata_reverse_hash(best_block_hash);

DO $verify$
DECLARE
	genesis TEXT;
	meta_hash TEXT;
	block_hash TEXT;
BEGIN
	SELECT encode(hash, 'hex') INTO STRICT genesis
	FROM blocks WHERE height = 0 AND is_mainchain;
	IF genesis <> '5f91ddfa5e9ffc4b837b81035dd0b0ddf1a1c59116aa5f5ef5f4121a64a69478' THEN
		RAISE EXCEPTION 'repaired mainnet genesis bytes are incorrect: %', genesis;
	END IF;

	SELECT encode(m.best_block_hash, 'hex'), encode(b.hash, 'hex')
	INTO meta_hash, block_hash
	FROM meta AS m
	LEFT JOIN blocks AS b ON b.height = m.best_block_height AND b.is_mainchain;
	IF meta_hash IS NOT NULL AND meta_hash IS DISTINCT FROM block_hash THEN
		RAISE EXCEPTION 'meta best block hash does not match blocks table: % <> %',
			meta_hash, block_hash;
	END IF;
END
$verify$;

COMMIT;
\echo Hash byte-order repair committed successfully.
