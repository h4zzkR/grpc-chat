package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"hash/fnv"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "grpc-messenger/proto"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

func MakeCacheKey(msg *pb.MSResponse) string {
	return msg.Message.Name + "\t" + msg.Timestamp.AsTime().Format(time.RFC3339)
}

func CacheMessage(client *redis.Client, msg *pb.MSResponse) {
	key := MakeCacheKey(msg)
	data, err := proto.Marshal(msg)

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	content := msg.Message.Content + "\t" + strconv.Itoa(int(msg.Message.Expire))

	_, err = client.HSet(ctx, RedisCacheDBName, key, data).Result()

	if err != nil {
		log.Fatalf("Error adding %s", content)
	}
}

func RemoveFromCacheMessage(client *redis.Client, key string) {
	ctx := context.Background()
	client.HDel(ctx, RedisCacheDBName, key)
}

func GetCachedHistory(client *redis.Client) map[string]string {
	ctx := context.Background()
	return client.HGetAll(ctx, RedisCacheDBName).Val()
}

type SortEntry struct {
	content   *pb.MSResponse
	timestamp time.Time
}

func UnCacheSortAllMessages(client *redis.Client) []SortEntry {
	history := GetCachedHistory(client)

	var sortedMsgs []SortEntry

	for _, value := range history {

		msg := new(pb.MSResponse)
		err := proto.Unmarshal([]byte(value), msg)
		if err != nil {
			panic(err)
		}

		sortedMsgs = append(sortedMsgs, SortEntry{msg, msg.Timestamp.AsTime()})
	}

	sort.Slice(sortedMsgs, func(i, j int) bool {
		return sortedMsgs[i].timestamp.Before(sortedMsgs[j].timestamp)
	})

	return sortedMsgs
}

func CheckExpire(client *redis.Client, msg *pb.MSResponse) bool {
	if msg.Message.Expire == 0 {
		return false
	}

	delay := msg.Timestamp.AsTime().Add(time.Second * time.Duration(msg.Message.Expire))
	if time.Now().Compare(delay) == 1 {
		RemoveFromCacheMessage(client, MakeCacheKey(msg))
		return true
	}

	return false
}

func readUnderlying(lines chan interface{}) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		lines <- s.Text()
	}
	lines <- s.Err()
}

// Sith technique to generate UIDs based on usernames
// Why: https://auth0.com/learn/token-based-authentication-made-easy
func MakeToken(username string) string {
	// Generate a random 32-byte array
	tokenBytes := make([]byte, UidLength)
	rand.Read(tokenBytes)

	h := fnv.New32a()
	h.Write([]byte(username))
	usernameHash := h.Sum32()

	tokenBytes = append(tokenBytes, byte(usernameHash>>24))
	tokenBytes = append(tokenBytes, byte(usernameHash>>16))
	tokenBytes = append(tokenBytes, byte(usernameHash>>8))
	tokenBytes = append(tokenBytes, byte(usernameHash))

	token := base64.RawStdEncoding.EncodeToString(tokenBytes)

	return token
}

func StreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	md, ok := metadata.FromIncomingContext(ss.Context())

	if !ok {
		return status.Error(codes.InvalidArgument, "metadata not found")
	}

	token, ok := md["authtoken"]
	if !ok || len(token) == 0 {
		return status.Error(codes.Unauthenticated, "missing authorization token")
	}

	username, ok := md["username"]
	if !ok || len(username) == 0 {
		return status.Error(codes.Unauthenticated, "missing username")
	}

	ctx := context.WithValue(ss.Context(), "authToken", token[0])
	ctx = context.WithValue(ctx, "username", username[0])

	log.Printf("Interceptor: user %s | token %s", username[0], token[0])

	return handler(srv, &WrappedStream{ss, ctx})
}

// A wrapper for the server stream that includes a modified context
type WrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *WrappedStream) Context() context.Context {
	return w.ctx
}
