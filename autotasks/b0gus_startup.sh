#!/usr/bin/env bash
# SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

# TODO.
# use www-data instead
# and we need port forwarding like this
# then we could deploy b0gus services on non-privilege ports
sudo iptables -t nat -A PREROUTING -p tcp --dport 22 -j REDIRECT --to-ports $(B0GUS_SSH_PORT)