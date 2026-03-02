#!/usr/bin/env bash
# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
set -ue

# TODO and yet to Test.
# use www-data instead
# and we need port forwarding like this
# so that we could deploy b0gus services on non-privilege ports
# and other services should have similar port forwarding
sudo iptables -t nat -A PREROUTING -p tcp --dport 22 -j REDIRECT --to-ports $(B0GUS_SSH_PORT)
