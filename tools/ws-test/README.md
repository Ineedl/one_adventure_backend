# WebSocket Test Server

Run locally with `go run -buildvcs=false .` from this directory. It listens on `127.0.0.1:18081`.

`POST /connect` opens a WebSocket using the supplied `Authorization: Bearer` token and connection parameters. `POST /send` sends its `body` as the WebSocket request `data` unchanged.

```bash
curl http://127.0.0.1:18081/connect -H 'Content-Type: application/json' -d '{
  "url":"ws://127.0.0.1:8080/ws_gateway/1/ws",
  "token":"<access-token>", "server_id":1, "channel_id":1, "channel_index":1
}'

curl http://127.0.0.1:18081/send -H 'Content-Type: application/json' -d '{
  "connection_id":"<id from connect>", "type":"LOGIN_REQ", "body":{"username":"demo"}
}'
```
