## 安装
安装需要root权限。
```shell
wget -qO - https://www.mongodb.org/static/pgp/server-8.0.asc | gpg -o /etc/apt/keyrings/mongodb-server-8.0.gpg --dearmor
```
国内最好设置镜像源。
```shell
# 在root权限下用nano编辑vim编辑或者gedit编辑也可以
echo "deb [signed-by=/etc/apt/keyrings/mongodb-server-8.0.gpg] https://mirrors.cernet.edu.cn/mongodb/apt/debian bookworm/mongodb-org/8.0 main" | \
tee /etc/apt/sources.list.d/mongodb-org-8.0.list
apt update && apt install -y mongodb-org
```

## 基本用法
用户模式登录进mongoDB后端的方法。
```shell
$ mongosh
> use admin
admin> db.createUser({user:"admin",pwd:"password!!!",roles:[{role:"userAdminAnyDatabase",db:"admin"}]})
admin> use b0gus
b0gus> show tables
b0gus> db.AddrInfo.drop()
b0gus> db.CommandInfo.find()
b0gus> quit
```
