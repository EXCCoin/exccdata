#!/bin/sh
#
# Tmux script that sets up a exccdata simnet mining harness.
#
# The script makes a few assumptions about the system it is running on:
# - tmux is installed
# - exccd, exccwallet, exccctl, exccdata and available on $PATH
#
#             alpha  <------>  beta
#    listen   19100           19200
# rpclisten   19101 <.     .> 19201
#                aw |      |  bw
# rpclisten   19102          19202

set -e
exccwallet -C wallet.conf --create <<EOF
y
n
y
${ALPHA_WALLET_SEED}
EOF" C-m
sleep 1
tmux send-keys "exccwallet -C wallet.conf" C-m

################################################################################
# Setup the alpha wallet's exccctl (awctl).
################################################################################
sleep 3
tmux new-window -t $TMUX_SESSION -n 'awctl'
tmux send-keys "cd ${HARNESS_ROOT}/${ALPHA_WALLET}" C-m
tmux send-keys "./ctl getnewaddress default" C-m # alpha send-to address.
tmux send-keys "./ctl getbalance" C-m

################################################################################
# Setup the beta node.
################################################################################
cat > "${HARNESS_ROOT}/${BETA}/ctl" <<EOF
#!/bin/sh
exccctl -C dcrbctl.conf \$*
EOF
chmod +x "${HARNESS_ROOT}/${BETA}/ctl"

tmux new-window -t $TMUX_SESSION -n 'beta'
tmux send-keys "cd ${HARNESS_ROOT}/${BETA}" C-m

echo "Starting beta node"
tmux send-keys "exccd -C ../exccd.conf --appdata=${HARNESS_ROOT}/${BETA} \
--whitelist=127.0.0.1 --connect=127.0.0.1:19100 \
--listen=127.0.0.1:19200 --rpclisten=127.0.0.1:19201 \
--miningaddr=${BETA_WALLET_MININGADDR}" C-m

################################################################################
# Setup the beta node's exccctl (bctl).
################################################################################
cat > "${HARNESS_ROOT}/${BETA}/mine" <<EOF
#!/bin/sh
  NUM=1
  case \$1 in
      ''|*[!0-9]*)  ;;
      *) NUM=\$1 ;;
  esac
  for i in \$(seq \$NUM) ; do
    exccctl -C dcrbctl.conf generate 1
    sleep 0.3
  done
EOF
chmod +x "${HARNESS_ROOT}/${BETA}/mine"

# send some coins to the alpha wallet and mine on beta.
cat > "${HARNESS_ROOT}/${BETA}/advance" <<EOF
#!/bin/sh
  NUM=1
  case \$1 in
      ''|*[!0-9]*)  ;;
      *) NUM=\$1 ;;
  esac
  for i in \$(seq \$NUM) ; do
    cd ${HARNESS_ROOT}/${BETA_WALLET} && ./ctl sendfrom default ${ALPHA_WALLET_SEND_TO_ADDR} 5
    cd ${HARNESS_ROOT}/${BETA} && ./mine 1
  done
EOF
chmod +x "${HARNESS_ROOT}/${BETA}/advance"

# force a reorganization.
cat > "${HARNESS_ROOT}/${BETA}/reorg" <<EOF
#!/usr/bin/env bash
./ctl node remove 127.0.0.1:19100
./mine 1
cd "${HARNESS_ROOT}/${ALPHA}"
./mine 2
cd "${HARNESS_ROOT}/${BETA}"
./ctl node connect 127.0.0.1:19100 perm
EOF
chmod +x "${HARNESS_ROOT}/${BETA}/reorg"

tmux new-window -t $TMUX_SESSION -n 'bctl'
tmux send-keys "cd ${HARNESS_ROOT}/${BETA}" C-m

# Mine 60 blocks on beta.
sleep 1
echo "Mining 60 blocks on beta"
tmux send-keys "./mine 60; tmux wait -S mine-beta" C-m
tmux wait mine-beta

################################################################################
# Setup the beta wallet.
################################################################################
cat > "${HARNESS_ROOT}/${BETA_WALLET}/ctl" <<EOF
#!/bin/sh
exccctl -C bwctl.conf --wallet \$*
EOF
chmod +x "${HARNESS_ROOT}/${BETA_WALLET}/ctl"

tmux new-window -t $TMUX_SESSION -n 'bw'
tmux send-keys "cd ${HARNESS_ROOT}/${BETA_WALLET}" C-m
echo "Creating beta wallet"
tmux send-keys "exccwallet -C wallet.conf --create <<EOF
y
n
y
${BETA_WALLET_SEED}
EOF" C-m
tmux send-keys "exccwallet -C wallet.conf" C-m

################################################################################
# Setup the beta wallet's exccctl (bwctl).
################################################################################
sleep 3
tmux new-window -t $TMUX_SESSION -n 'bwctl'
tmux send-keys "cd ${HARNESS_ROOT}/${BETA_WALLET}" C-m
tmux send-keys "./ctl getnewaddress default" C-m # beta send-to address.
tmux send-keys "./ctl getbalance"

echo "Progressing the chain by 12 blocks via alpha"
tmux select-window -t $TMUX_SESSION:1
tmux send-keys "./advance 12; tmux wait -S advance-alpha" C-m
tmux wait advance-alpha

echo "Progressing the chain by 12 blocks via beta"
tmux select-window -t $TMUX_SESSION:5
tmux send-keys "./advance 12; tmux wait -S advance-beta" C-m
tmux wait advance-beta

echo "Progressed the chain to Stake Validation Height (SVH)"
echo Attach to simnet nodes/wallets with \"tmux a -t $TMUX_SESSION\".
# tmux attach-session -t $TMUX_SESSION


# TODO: the harness currently creates coinbases, tickets, votes and regular
# transactions. It'll need to generate revocations and swap transactions as well.
