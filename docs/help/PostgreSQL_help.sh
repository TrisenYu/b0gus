#!/usr/bin/env sh

set -ue
cat << EOF
###### log into PostgreSQL by cli<psql>.
sudo -u postgre psql
###### show user
\du
###### show tables
\dt
###### delete table.
DROP TABLE IF EXISTS "tabName" CASCADE;
###### view content in the table like this.
select * from "AddrInfo";
EOF