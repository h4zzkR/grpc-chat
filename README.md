# Install go and other

$ wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
$ sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
$ export PATH=$PATH:/usr/local/go/bin

$ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
$ sudo apt install -y protobuf-compiler

$ go get golang.org/x/net/context
$ go get -u github.com/inancgumus/screen

$ protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/messenger.proto

$ sudo snap install redis
$ sudo apt install redis-server
$ go get github.com/redis/go-redis/v9
$

# Run

run redis server: `redis-server --port 7777`
server: `go build && ./grpc-messenger -s`
client with h4zzkR username: `go build && ./grpc-messenger -u h4zzkR`

# Demo

People talk in spaces (or telegram groups), there are no direct messages.
Messages are server-side cached, when new client connects he loads this message history and can view older messages.

/time 5 hello - auto deleted messages (update CLI to see they were removed)

![two-people-session](https://lh3.googleusercontent.com/pw/AMWts8DOruQBrr2YttzHXzIlaqLvfxugsAOaq9XnmpayfGB1tlnPZeM5vixDWExeIRqlS09faq-C-eHzFm_iP1ZpDCglWykz22EN1GlcVELrKP-vCHK76gqn0gCNfpBZ52F7kbkw7ssnvF2A_4VCosSTzzGO=w1280-h313-s-no?authuser=0)

# Todo
- Cancellation (shutdown server, logout, client reactions to this events)
