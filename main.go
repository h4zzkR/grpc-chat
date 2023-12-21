package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/net/context"
)

var (
	isServer bool
	hostAdrr string
	password string
	username string

	err error
)

func onExit(stop context.CancelFunc) {
	// https://gobyexample.com/exit
	log.Printf("Shutdown...")

	stop()

	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func runRoutine(ctx context.Context) chan error {
	eventC := make(chan error)

	if isServer {
		log.SetPrefix("[Server] ")
		log.Print("Running in server mode...")
		err = NewServer(hostAdrr, password).Run(ctx)
	} else {
		log.SetPrefix("[Client] ")
		log.Print("Running in client mode...")
		err = NewClient(hostAdrr, username, password).Run(ctx)
	}

	if err != nil {
		log.Fatal("Failure: ", err)
	}

	return eventC
}

func main() {
	flag.BoolVar(&isServer, "s", false, "if true, run server")
	flag.StringVar(&hostAdrr, "h", "127.0.0.1:7778", "server host adress")
	flag.StringVar(&password, "p", "", "password for server entering")
	flag.StringVar(&username, "u", "", "client username")
	flag.Parse()

	// https://henvic.dev/posts/signal-notify-context/
	// https://gobyexample.com/signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer onExit(stop)

	select {
	case err = <-runRoutine(ctx):
		onExit(stop)
	case <-ctx.Done():
		onExit(stop)
	}

}
