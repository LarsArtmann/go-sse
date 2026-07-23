# go-sse example server

Minimal SSE server demonstrating the [go-sse](https://github.com/larsartmann/go-sse) library.

## Run

```bash
go run example/server.go
```

## Try it

### Subscribe (terminal 1)

```bash
curl -N http://localhost:8080/events
```

You'll see:

```
event: connected
data: connected
id: 0

: heartbeat
```

### Broadcast (terminal 2)

```bash
curl -X POST http://localhost:8080/broadcast?msg=hello
```

Terminal 1 immediately shows:

```
event: message
data: hello
id: 1770000000000000000
```

## JavaScript client

```html
<script>
  const es = new EventSource("http://localhost:8080/events");

  es.addEventListener("connected", (e) => {
    console.log("connected, last event:", e.lastEventId);
  });

  es.addEventListener("message", (e) => {
    console.log("message:", e.data);
  });
</script>
```

`EventSource` handles reconnection automatically and sends the `Last-Event-ID`
header on reconnect, enabling replay of missed events.
