# A TCP Traffic Forwarder

Golang implementation of 「A TCP Traffic Forwarder」, which does forward traffic from local socket to remote socket for client. The code requires Golang version 1.18 or newer.

## How to Install?

```shell
git clone https://github.com/amazingchow/traffic-forwarder.git
cd /path/to/traffic-forwarder
make build
```

## How to Use?

To start the forwarder proc:
```shell
cd /path/to/traffic-forwarder
make start
```

To stop the forwarder proc:
```shell
cd /path/to/traffic-forwarder
make stop
```

Watch logs from the forwarder proc:
```shell
cd /path/to/traffic-forwarder
tail -f -n50 nohup.out
```
