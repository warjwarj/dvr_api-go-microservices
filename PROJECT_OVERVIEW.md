# DVR API — Go Microservices: Project Overview

## What It Is

A **DVR (Digital Video Recorder) backend system** built as Go microservices. Real DVR hardware devices connect to it over TCP, send telemetry/status messages, and can push video footage. Frontend clients (browser) connect via WebSocket and subscribe to device streams in real-time.

---

## The Four Services

### 1. `device_svr` — Device Gateway
Accepts raw TCP connections from DVR hardware. Each device sends messages terminated by `\r`. The server:
- Reads the device's ID from its first message
- Registers the connection in a `Dictionary` (thread-safe map)
- Publishes incoming messages to RabbitMQ (`MSGS_FROM_DEVICE_SVR` exchange)
- Subscribes to `MSGS_FROM_API_SVR` and writes those back to the relevant device connection
- Fires `DeviceConnectionStateChange` events to RabbitMQ on connect/disconnect

### 2. `ws_api_svr` — WebSocket API + Pub/Sub
Accepts WebSocket connections from browser clients. Each client sends a JSON frame with:
- `Subscriptions` — list of device IDs they want to follow
- `Messages` — commands to forward to devices (including `$VIDEO` requests)

A `PubSubHandler` runs alongside it:
- Reads device messages from RabbitMQ and fans them out to all subscribed WebSocket clients
- Reads video descriptions from RabbitMQ and routes them to whichever client requested that video
- Maintains a `msgSubscriptions` map (`device_id -> {client_id -> client_id}`) so lookups are O(1) without loops

### 3. `cam_svr` — Camera / Video Ingestion
Accepts TCP connections from DVRs sending raw video data in a custom binary framing protocol (`$FILE` packets). For each connection it:
- Reads framed packets (fixed-size header -> variable payload header -> variable payload body)
- Writes each chunk to disk in `/var/tmp/`
- Runs `convert.sh` (ffmpeg) then `concat.sh` to produce MP4 files
- Uploads them to **AWS S3**
- Publishes a `VideoDescription` (with S3 link, timing metadata) to RabbitMQ (`VIDEO_DESCRIPTION_EXCHANGE`)

### 4. `dbproxy_svr` — Database Proxy
Subscribes to RabbitMQ message streams and persists everything to **MongoDB**. Acts as the persistence layer so no other service needs to know about the database.

---

## The Message Broker (RabbitMQ) — Why It's the Core

Without a broker, every service would need a direct connection to every other service. Instead, each service only knows about **exchanges**, not each other:

```
[DVR Device] --TCP--> device_svr --> MSGS_FROM_DEVICE_SVR --> ws_api_svr (fans out to WS clients)
                                                           └-> dbproxy_svr (persists to MongoDB)

[Browser]    --WS---> ws_api_svr --> MSGS_FROM_API_SVR   --> device_svr (forwards to device)

[DVR Cam]    --TCP--> cam_svr    --> VIDEO_DESCRIPTION_EXCHANGE --> ws_api_svr (delivers to requesting client)
```

This means:
- `dbproxy_svr` can be added or removed without touching any other service
- `cam_svr` doesn't know `ws_api_svr` exists — it just publishes and walks away
- Services can be scaled independently — run two `device_svr` instances and the broker handles it
- A service crash doesn't cascade — messages queue up and are delivered when it recovers

---

## Go's Linear Connection Lifecycle

Every `main.go` reads top-to-bottom like a recipe, with `defer` automatically handling teardown in reverse order:

```go
// 1. Logger
logger, _ := zap.NewDevelopment()
defer logger.Sync()                        // cleaned up last

// 2. AMQP connection
conn, _ := amqp.Dial(config.RABBITMQ_AMQP_ENDPOINT)
defer conn.Close()                         // cleaned up second

// 3. AMQP channel (logical multiplexer over the connection)
ch, _ := conn.Channel()
defer ch.Close()                           // cleaned up first

// 4. Build service, inject dependencies
svr, _ := NewDeviceSvr(logger, endpoint, ..., ch)

// 5. Run
go svr.Run()
<-forever
```

There is no setup/teardown class, no lifecycle hooks, no framework magic. The same linearity appears inside each connection handler:

```go
func (s *DeviceSvr) deviceConnLoop(conn net.Conn) error {
    buf, _ := s.sockOpBufStack.Pop()       // acquire buffer

    // On first message: register, then defer the cleanup
    if id == nil {
        id, _ = utils.GetIdFromMessage(&msg)
        s.connIndex.Add(*id, conn)
        s.svrRegisterChangeChan <- ConnectionStateChange{id, true}

        defer func() {                     // fires when function returns for any reason
            s.connIndex.Delete(id)
            s.sockOpBufStack.Push(buf)     // return buffer to pool
            s.svrRegisterChangeChan <- ConnectionStateChange{id, false}
        }()
    }

    // Normal read loop continues...
}
```

The `defer` fires whether the loop exits cleanly, on EOF, or on any error. **Connection registration and deregistration are co-located** — you can't accidentally forget to deregister, because the cleanup is declared at the same place as the registration. No try/finally, no RAII wrappers, no cleanup callbacks — just sequential code that reads in the order it executes.
