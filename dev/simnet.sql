-- This resets the exccdata_simnet database.

DROP DATABASE IF EXISTS exccdata_simnet;
DROP USER IF exists exccdata_simnet_stooge;

CREATE USER exccdata_simnet_stooge PASSWORD 'pass';
CREATE DATABASE exccdata_simnet OWNER exccdata_simnet_stooge;
