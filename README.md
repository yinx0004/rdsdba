# rdsdba

## Introduction
This cloud be a CLI toolkit for RDS DBA.

## Functions
Currently only support MySQL InnoDB buffer pool warmup.

## Examples
### Help
```shell
./rdsdba --help
RDS DBA CLI plan to provide rich features to support general
database operations, troubleshooting, performance diagnose, etc

Usage:
  rdsdba [command]

Available Commands:
  help        Help about any command
  warmup      Warm up data

Flags:
  -h, --help      help for rdsdba
  -v, --version   version for rdsdba

Use "rdsdba [command] --help" for more information about a command.
```

```shell
 ./rdsdba warmup --help
Warm up RDS instance by reading data on disk and load into memory to
speed up upcoming traffic.

Usage:
  rdsdba warmup [flags]

Flags:
  -l, --connection-max-lifetime duration   the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default (default -1m0s)
  -D, --debug                              show debug level log
  -h, --help                               help for warmup
  -H, --host string                        mysql host (default "localhost")
  -c, --max-connection int                 max number of open mysql connections (default 50)
  -i, --max-idle-connection int            max number of idle mysql connections (default 50)
  -o, --only strings                       only load specific tables to memory, comma separated format:schema_name.table_name, schema_name2.table_name2, whitespaces between comma is allowed
  -p, --password string                    mysql password
  -P, --port int                           mysql server port (default 3306)
  -s, --skip strings                       skip cold tables to let them stay on disk, comma separated format:schema_name1.table_name1,schema_name2.table_name2, whitespaces between comma is allowed
  -t, --thread int                         number of threads (default 20)
  -u, --user string                        mysql user (default "root")
```

### MySQL InnoDB Buffer Pool Warmup
#### Warmup all user tables
```shell
rdsdba warmup -H '127.0.0.1' --user root --password 'yourpassword' --thread 4  2>&1 |tee 1.log 
```

#### Warmup partial user tables by skipping some tables
```shell
rdsdba warmup -H '127.0.0.1' --user root --password 'yourpassword' --thread 10 --skip 'testdb1.cold_tab1, testdb2.cold_tab2'  2>&1 |tee 1.log 
```

#### Warmup partial user tables by providing some tables need to be warmed up
```shell
rdsdba warmup -H '127.0.0.1' --user root --password 'yourpassword' --thread 2 --only 'testdb1.hot_tab1, testdb2.hot_tab2'  2>&1 |tee 1.log 
```
#### Notice
**--skip and --only flag are exclusive**