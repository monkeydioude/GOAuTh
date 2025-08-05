package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/monkeydioude/goauth/pkg/plugins"

	"github.com/google/uuid"
	"github.com/monkeydioude/heyo/pkg/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const HEYO_SERVER_ADDR = "[::]:8022"

func getRPCClient() (rpc.BrokerClient, error) {
	addr := HEYO_SERVER_ADDR
	if os.Getenv("HEYO_SERVER_ADDR") != "" {
		addr = os.Getenv("HEYO_SERVER_ADDR")
	}
	cl, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	if cl == nil {
		return nil, errors.New("*grpc.ClientConn was nil")
	}
	log.Printf("[INFO] connecting to %s\n", addr)
	return rpc.NewBrokerClient(cl), nil
}

func init() {
	grpcClient, err := getRPCClient()
	if err != nil {
		log.Printf("could not init heyo plugin: %s", err)
		return
	}
	ctx := context.TODO()
	clientUuid := uuid.NewString()
	AddPlugin("heyo-new-user", plugins.OnUserCreation, nil, func(event plugins.Event, payload any) {
		log.Printf("received '%s': %+v\n", event, payload)
		ctx, cancelFn := context.WithTimeout(ctx, 1*time.Second)
		defer cancelFn()
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("error while marshalling in heyo plugin: %s", err)
			return
		}
		ack, err := grpcClient.Enqueue(ctx, &rpc.Message{
			Event:      string(plugins.OnUserCreation),
			Data:       string(data),
			ClientId:   clientUuid,
			MessageId:  uuid.NewString(),
			ClientName: "identity",
		})
		if err != nil {
			log.Printf("error while sending message to the queue: %s", err)
			return
		}
		log.Println(ack)
	})
}
