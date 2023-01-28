# rdsdba

## Introduction
This cloud be a CLI toolkit for RDS DBA.

## Functions
- Support MySQL InnoDB buffer pool warmup.
- Stress test specified query/queries.

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
  stress      Run stress test on MySQL
  warmup      Warm up MySQL InnoDB buffer pool

Flags:
  -l, --connection-max-lifetime duration   the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default (default -1m0s)
  -D, --debug                              show debug level log
  -h, --help                               help for rdsdba
  -H, --host string                        RDS host (default "localhost")
  -c, --max-connection int                 max number of open connections to RDS (default 50)
  -i, --max-idle-connection int            max number of idle connections to RDS (default 50)
  -p, --password string                    RDS password
  -P, --port int                           RDS port (default 3306)
  -u, --user string                        RDS user (default "root")
  -v, --version                            version for rdsdba

Use "rdsdba [command] --help" for more information about a command.
```

```shell
 ./rdsdba warmup --help
Warm up RDS MySQL instance InnoDB buffer pool by reading data on disk and load into memory to
speed up upcoming traffic.

Usage:
  rdsdba warmup [flags]

Flags:
  -h, --help           help for warmup
  -o, --only strings   only load specific tables to memory, comma separated format:schema_name.table_name, schema_name2.table_name2, whitespaces between comma is allowed
  -s, --skip strings   skip cold tables to let them stay on disk, comma separated format:schema_name1.table_name1,schema_name2.table_name2, whitespaces between comma is allowed 
  -t, --thread int     number of threads (default 1)

Global Flags:
  -l, --connection-max-lifetime duration   the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default (default -1m0s)
  -D, --debug                              show debug level log
  -H, --host string                        RDS host (default "localhost")
  -c, --max-connection int                 max number of open connections to RDS (default 50)
  -i, --max-idle-connection int            max number of idle connections to RDS (default 50)
  -p, --password string                    RDS password
  -P, --port int                           RDS port (default 3306)
  -u, --user string                        RDS user (default "root")
```
```shell
Run stress test on MySQL,
specified queries instead of random queries.

Usage:
  rdsdba stress [flags]

Flags:
  -f, --file string     the file which contains multiple queries used for stress test, for each query must provide a weighted, separated by ';'
  -h, --help            help for stress
  -q, --query string    single query used for stress test, accepted in command line
  -t, --thread int      number of threads(connections) (default 1)
  -T, --time duration   stress test time, support time duration [s|m|h] (default 30s)

Global Flags:
  -l, --connection-max-lifetime duration   the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default (default -1m0s)
  -D, --debug                              show debug level log
  -H, --host string                        RDS host (default "localhost")
  -c, --max-connection int                 max number of open connections to RDS (default 50)
  -i, --max-idle-connection int            max number of idle connections to RDS (default 50)
  -p, --password string                    RDS password
  -P, --port int                           RDS port (default 3306)
  -u, --user string                        RDS user (default "root")
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

### MySQL Stress Test Read Only
#### Stress test single query
```shell
rdsdba stress --time 60s --thread 20 --host localhost --user root -p xxxx --query "select sleep(1)" 

{"level":"info","time":"2023-01-28T12:39:41+08:00","message":"start"}
{"level":"info","time":"2023-01-28T12:40:41+08:00","message":"end"}

        Latency(ms):
                min:                1017
                avg:                1068
                max:                1094
                95th percentile:    1076

        General statistics:
                total time:         60.003456s
                total queries:      1118

        SQL statistics:
                qps:                18.632260
                ignored errors:     20

```
#### Stress test multiple queries
1. Prepare queries and weight
```
$ cat queries.txt
select sleep(1); 6
select sleep(2); 2
select sleep(3); 2
```
> `select sleep(1)` execution times will be 60%
2. Run stress test
```shell
rdsdba stress --time 60s --thread 20 --host localhost --user root -p xxxx --file queries.txt
```