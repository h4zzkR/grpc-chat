package main

import (
	"fmt"
	pb "grpc-messenger/proto"
	"log"
	"strconv"
	"time"

	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	// "github.com/inancgumus/screen"
	"github.com/redis/go-redis/v9"
)

type client struct {
	HostAdrr  string
	password  string
	username  string
	authToken string

	redisClient *redis.Client
	pb.MessengerClient
}

func NewClient(hostAdrr, username, password string) *client {
	return &client{
		HostAdrr: hostAdrr,
		password: password,
		username: username,
		redisClient: redis.NewClient(&redis.Options{
			Addr:     RedisCacheAddr,
			Password: RedisCachePass, // no password set
			DB:       0,              // use default DB
		}),
		authToken: "",
	}
}

func (c *client) Run(ctx context.Context) error {
	_, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Printf("Starting client request on %s", c.HostAdrr)

	conn, err := grpc.DialContext(ctx, c.HostAdrr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal("Client dialing failure: ", err)
		return err
	}

	defer conn.Close()
	c.MessengerClient = pb.NewMessengerClient(conn)

	if c.authToken, err = c.loginClient(ctx); err != nil {
		return err
	}

	log.Printf("Client %s logged in with %s token", c.username, c.authToken)

	c.LoadHistory()

	err = c.Messenging(ctx)

	return err
}

func (c *client) LoadHistory() {
	msgs := UnCacheSortAllMessages(c.redisClient)

	for _, item := range msgs {
		if !CheckExpire(c.redisClient, item.content) {
			c.printMessage(item.timestamp, item.content.Message.Name, item.content.Message.Content)
		}
	}
}

func (c *client) Messenging(ctx context.Context) error {
	md := metadata.Pairs("authtoken", c.authToken, "username", c.username)
	ctx = metadata.NewOutgoingContext(ctx, md)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clientProxy, err := c.MessengerClient.MessageStream(ctx)
	if err != nil {
		return err
	}

	go c.receiveRoutine(clientProxy)
	err = c.sendRoutine(ctx, clientProxy)

	if err != nil {
		log.Printf("Send routine error: %s", err.Error())
		return err
	}

	err = clientProxy.CloseSend()

	if err != nil {
		log.Printf("Close send error: %s", err.Error())
		return err
	}

	return err
}

// Read from terminal and send it via gRPC
func (c *client) sendRoutine(ctx context.Context, proxy pb.Messenger_MessageStreamClient) error {
	input := make(chan interface{}) // input stream
	go readUnderlying(input)        // go and read

	c.printPrompt()

	for {
		select { // read or close
		case <-ctx.Done():
			log.Print("Client logging out...")
			return c.logoutClient()
		case <-proxy.Context().Done():
			return nil
		case lineOrErr := <-input:
			if err, ok := lineOrErr.(error); ok {
				log.Printf("Scanner failure: %v", err)
				return err
			} else {

				msReq := c.parseInput(lineOrErr.(string))

				err := proxy.Send(msReq)
				if err != nil {
					log.Printf("Message sending failure: %v", err)
					return err
				}
				// log.Print("SENTMSG: ", c.username, lineOrErr.(string))
			}
		}
	}
}

// Get new messages from server via gRPC and print it to client
func (c *client) receiveRoutine(proxy pb.Messenger_MessageStreamClient) {
	for {
		in, err := proxy.Recv()
		if err != nil {
			return
		}

		// log.Print("RECVMSG: ", in.Timestamp.AsTime(), in.Message.Name, in.Message.Content)
		// log.Print("EXPIRES: ", in.Message.Expire)
		c.printMessage(in.Timestamp.AsTime(), in.Message.Name, in.Message.Content)
		c.printPrompt()
		// TODO error handling
	}
}

func (c *client) parseInput(input string) *pb.MSRequest {

	expireTime := time.Second * 0

	if strings.HasPrefix(input, "/time") {
		parts := strings.Split(input, " ")
		if len(parts) >= 2 {
			timerStr := parts[1]
			timer, err := strconv.Atoi(timerStr)
			if err == nil {
				expireTime = time.Duration(timer) * time.Second
				input = parts[2]
			} else {
				// TODO
			}
		}
	}

	return &pb.MSRequest{
		Message: input,
		Expire:  uint32(expireTime.Seconds()),
	}
}

func (c *client) printMessage(ts time.Time, uname string, content string) {
	strTs := ts.In(time.Local).Format("02-Jan-2006 15:04")
	fmt.Printf("\r%s [%s says:] %s\n", strTs, uname, content)
}

func (c *client) printPrompt() {
	// fmt.Printf("%s, write a message: ", c.username)
}

// Initial login for client. It sends password and username for space
// and receives client token. Token will be sent with every future message.
func (c *client) loginClient(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ClientConnectionTimeout)
	defer cancel()

	result, err := c.MessengerClient.Login(ctx, &pb.LoginRequest{
		Password: c.password,
		Username: c.username,
	})

	if err != nil {
		log.Fatal("client login failure: ", err)
	}

	return result.Token, err
}

func (c *client) logoutClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), ClientConnectionTimeout)
	defer cancel()

	_, err := c.MessengerClient.Logout(ctx, &pb.LogoutRequest{
		Username: c.username,
	})

	if err != nil {
		log.Fatal("client logout failure: ", err)
	}

	return err
}
