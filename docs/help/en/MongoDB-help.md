## Installation
installation require root privilege.
```shell
wget -qO - https://www.mongodb.org/static/pgp/server-8.0.asc | gpg -o /etc/apt/keyrings/mongodb-server-8.0.gpg --dearmor
```

set to mirror if necessary.
```shell
# # use editor whatever you like is ok. nano here is an example. 
echo "deb [signed-by=/etc/apt/keyrings/mongodb-server-8.0.gpg] https://mirrors.cernet.edu.cn/mongodb/apt/debian bookworm/mongodb-org/8.0 main" \
	| tee /etc/apt/sources.list.d/mongodb-org-8.0.list
apt update && apt install -y mongodb-org
```

## Basic Usage

login into mongo backend in user mode. 
```shell
mongosh # login into mongoServer
> use admin
admin> db.createUser({user:"admin",pwd:"password!!!",roles:[{role:"userAdminAnyDatabase",db:"admin"}]})
admin> use b0gus
b0gus> show tables
b0gus> db.AddrInfo.drop()
b0gus> db.CommandInfo.find()
b0gus> quit
```
