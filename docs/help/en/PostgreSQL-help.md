## Installation

```shell
# operation under root privilege
apt install postgresql
nano /etc/postgresql/17/main/pg_hba.conf # edit configuration of PostgreSQL
```

## Basic Usage
log into PostgreSQL by client application<psql>.
```shell
sudo -u postgres psql
```
|                command name                 | functionality of the command |
|:-------------------------------------------:|:----------------------------:|
|                    `\du`                    |          show users          |
|                    `\dt`                    |         show tables          |
| `DROP TABLE IF EXISTS "<tabName>" CASCADE;` |         delete table         |
|         `select * from "AddrInfo";`         |  view content in the table   |

there is a way that can help O&M once execute commands defined in PostgreSQL in batch mode.

```shell
sudo -u postgres psql << EOS
\du;
\dt;
DROP TABLE IF EXISTS "<tabName>" CASCADE;
EOS
```

