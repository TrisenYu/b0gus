#!/usr/bin/env sh
# shellcheck disable=SC3040
set -euo pipefail

cat << EOF
###### log into PostgreSQL by client application<psql>.
sudo -u postgres psql
###### show user
\du
###### show tables
\dt
###### delete table.
DROP TABLE IF EXISTS "<tabName>" CASCADE;
###### view content in the table like this.
select * from "AddrInfo";
###### -------------------- end of the basic interaction with postgres-cli ----------------------

####### if you do not want to type one command by one command, then
####### there is a way that can be done only once.
sudo -u postgres psql << EOS
\du;
\dt;
DROP TABLE IF EXISTS "<tabName>" CASCADE;
EOS
####### just like the method written above
EOF