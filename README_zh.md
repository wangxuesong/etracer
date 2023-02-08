# eTracer: 基于 eBPF 的代码跟踪工具

> **Note:**
>
> 支持Linux系统内核x86_64 4.18及以上版本；

# 使用

## 命令参数

> **Note**
>
> 需要ROOT权限执行。

```shell
Usage:
  etracer [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  pg          跟踪 PostgresSQL
```

```shell
Usage:
  etracer pg [flags]

Flags:
  -f, --funcname string   function name to hook (default "exec_simple_query")
  -h, --help              help for pg
  -p, --postgres string   postgres binary file path, use to hook (default "/usr/bin/postgres")
```

例子：
```shell
sudo ./etracer pg -p /home/martin/.pgx/13.9/pgx-install/bin/postgres
```
在 PG 上执行 SQL 后产生类似输出：
```shell
2023/02/07 15:56:51 Listening for events..
2023/02/07 15:56:54 pid: 38312, func: exec_simple_query, sql: select * from test;
```

# 编译方法

## 工具链版本
* golang >1.18
* clang >9.0
* clang backend: llvm >9.0

## 编译

```shell
sudo yum -y update
sudo yum -y install make llvm clang kernel-devel git
git clone https://github.com/wangxuesong/etracer.git
cd etracer
make build
sudo ./etracer pg
```
