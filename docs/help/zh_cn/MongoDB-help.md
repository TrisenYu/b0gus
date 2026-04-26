## 安装
安装需要root权限。
```shell
# wget -qO - https://www.mongodb.org/static/pgp/server-8.0.asc | gpg -o /etc/apt/keyrings/mongodb-server-8.0.gpg --dearmor
```
国内最好设置镜像源。
```shell
# # nano编辑vim编辑或者gedit编辑都可以
# nano /etc/apt/sources.list.d/mongodb-org-8.0.list
deb [signed-by=/etc/apt/keyrings/mongodb-server-8.0.gpg] https://mirrors.cernet.edu.cn/mongodb/apt/debian bookworm/mongodb-org/8.0 main
# apt update && apt install -y mongodb-org
## 应该是下好了
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
