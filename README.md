# ssh-chat-relay

Protocol relay for ssh servers.

Built for exposing [ssh-chat](https://github.com/shazow/ssh-chat) over other protocols, like websocket.

Requires ssh-chat v1.9 or newer (with `TERM=bot` support).

## Usage

```
Usage:
  ssh-chat-relay [OPTIONS] [Addr]

Application Options:
      --websocket= Websocket host:port to bind to and supply a relay
      --name=      Username to connect with (default: ssh-chat-relay)
  -v, --verbose    Show verbose logging.
      --version    Print version and exit.
      --pprof=     Bind pprof on http server on this addr. (Example: "localhost:6060")

Help Options:
  -h, --help       Show this help message

Arguments:
  Addr:            SSH host:port to connect with and relay
```

## License

MIT
