#!/usr/bin/env sh
# shellcheck disable=SC3040
set -euo pipefail

cat << EOS
#### basic usage
### installation require root privilege
# wget -qO - https://www.mongodb.org/static/pgp/server-8.0.asc | gpg -o /etc/apt/keyrings/mongodb-server-8.0.gpg --dearmor
## set to mirror if necessary
# deb [signed-by=/etc/apt/keyrings/mongodb-server-8.0.gpg] https://mirrors.cernet.edu.cn/mongodb/apt/debian bookworm/mongodb-org/8.0 main
# apt update && apt install -y mongodb-org
## done

$ mongosh # login into mongoServer
> use admin
> db.createUser({user:"admin",pwd:"password!!!",roles:[{role:"userAdminAnyDatabase",db:"admin"}]})
> use b0gus
b0gus> show tables
b0gus> db.AddrInfo.drop()
b0gus> db.CommandInfo.find()
> quit
EOS