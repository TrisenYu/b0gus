## 安装

```shell
apt install postgresql
nano /etc/postgresql/17/main/pg_hba.conf # 改PostgreSQL的配置
```

## Basic Usage
用psql登录到postgreSQL。
```shell
sudo -u postgres psql
```
|                     命令名                   |   命令功能   |
|:-------------------------------------------:|:-----------:|
|                    `\du`                    |   展示用户   |
|                    `\dt`                    |    展示表    |
| `DROP TABLE IF EXISTS "<tabName>" CASCADE;` |    删除表    |
|         `select * from "AddrInfo";`         |  查看表中内容 |

运维有一次性按照批处理的方式交给PostgreSQL执行的命令。不过这由持有权限的程度所决定。

```shell
# 切换用户从而登录到PostgreSQL
sudo -u postgres psql << EOS
\du;
\dt;
DROP TABLE IF EXISTS "<tabName>" CASCADE;
EOS
```

