# spiffe-pinger

Small utility for testing SPIFFE functionality.

The service:

- Connects to a SPIFFE Workload API to retrieve an X509 SVID
- Spins up a gRPC server that listens on a TCP address, and is protected by TLS
  using the X509 SVID
- Spins up a loop that pings a gRPC server using the X509 SVID as a client
  certificate

Spin up two of these and point them at one another e.g

```shell
SPIFFE_ENDPOINT_SOCKET=unix:///tmp/workload-socket-a.sock LISTEN=127.0.0.1:1338 TARGET=127.0.0.1:1337 go run ./main.go
```

```shell
SPIFFE_ENDPOINT_SOCKET=unix:///tmp/workload-socket-b.sock LISTEN=127.0.0.1:1337 TARGET=127.0.0.1:1338 go run ./main.go
```

The logs will indicate the identity of the service itself, and the identity of
any client which connects to it:

```shell
2024/08/30 13:12:36 INFO Sent message me=spiffe://leaf.tele.ottr.sh/example component=client
2024/08/30 13:12:37 INFO Received request me=spiffe://leaf.tele.ottr.sh/example component=server from=spiffe://spire.tele.ottr.sh/macbook/noah
2024/08/30 13:12:41 INFO Sent message me=spiffe://leaf.tele.ottr.sh/example component=client
2024/08/30 13:12:42 INFO Received request me=spiffe://leaf.tele.ottr.sh/example component=server from=spiffe://spire.tele.ottr.sh/macbook/noah
2024/08/30 13:12:46 INFO Sent message me=spiffe://leaf.tele.ottr.sh/example component=client
2024/08/30 13:12:47 INFO Received request me=spiffe://leaf.tele.ottr.sh/example component=server from=spiffe://spire.tele.ottr.sh/macbook/noah
```