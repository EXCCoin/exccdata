#!/usr/bin/env bash

# Run this script from the "dev" folder to:
#  1. Start parallel simnet nodes and wallets
#  2. Initialize a postgresql database for this simnet session.
#  3. Start exccdata in simnet mode connected to the alpha node.
#
# When done testing, stop exccdata with CTRL+C or SIGING, then use stop-simnet.sh
# to stop all simnet nodes and wallets.

set -e

HARNESS_ROOT=~/exccdsimnet

echo "Starting simnet nodes and wallets..."
rm -rf ~/exccdsimnet
./exccdata-harness.tmux

echo "Use stop-simnet.sh to stop nodes and wallets."

sleep 5

echo "Preparing PostgreSQL for simnet exccdata..."
PSQL="sudo -u postgres -H psql"
$PSQL < ./simnet.sql

rm -rf ~/.exccdata/data/simnet
rm -rf datadir
pushd .. > /dev/null
exccdata -C ./dev/exccdata-simnet.conf -g --exccdserv=127.0.0.1:19201 \
--exccdcert=${HARNESS_ROOT}/beta/rpc.cert
popd > /dev/null

echo " ***
Don't forget to run ./stop-simnet.sh!
 ***"
