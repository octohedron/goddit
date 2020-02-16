## Golang chat server 🏓

============================

##### Try it out!

- Install mongodb
- Clone repo to `$GOPATH/src/github.com/octohedron/goddit`
- Install dependencies

```bash
$ go get
```

- Build the binary

```bash
$ go build
```

- Edit the [.env](./.env) file in the root of the repository with your values

```bash
# from reddit.com/prefs/apps
APPID=YOUR_APP_ID
APPSECRET=YOUR_APP_SECRET
GODDITADDR=YOUR_SERVER_ADDR # i.e. http://localhost:9000
GODDITDOMAIN=YOUR_DOMAIN # i.e. localhost / goddit.pro
GPORT=9000 # 9000 / 80
GCOOKIE=YOUR_COOKIE # i.e. goddit
MONGO_ADDR=YOUR_MONGO_ADDR # i.e. 127.0.0.1, 172.17.0.1 for docker, etc
```

- Run it manually

```bash
$ nohup ./goddit &
```

- Run it with Docker

```bash
docker rm goddit && docker build -t goddit . && docker run --name goddit -p 9000:9000 -it goddit
```

## Features

- Crawls popular subreddits (first 25) and adds them as chatrooms in a separate `goroutine` (for faster startup) when you start the program
- Loads previous chatroom history (up to 150 messages) when switching rooms
- Authorizes via reddit and pairs a `cookie` with a user in the server for Authorization
- Only broadcasts messages to those present in the room, not to others in different rooms
- Different color avatars, autogenerated from [placeskull.com](http://placeskull.com)
- Enforces a single `WebSocket` for each client, closing the connection when switching rooms
- Each client reads and writes to a different `goroutine` (2 goroutines/client)
- Database writes are performed in a separate `goroutine`

##### CONTRIBUTING: Yes

LICENSE: MIT
