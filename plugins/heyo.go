package plugins

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/plugins"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/monkeydioude/heyo/pkg/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getRPCClient() (rpc.BrokerClient, error) {
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true, // Skip verification for testing; remove this in production
	})

	addr := consts.HEYO_SERVER_ADDR
	if os.Getenv("HEYO_SERVER_ADDR") != "" {
		addr = os.Getenv("HEYO_SERVER_ADDR")
	}
	cl, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(creds),
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
		ctx, cancelFn := context.WithTimeout(ctx, 1*time.Second)
		defer cancelFn()
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("error while marshalling in heyo plugin: %s", err)
			return
		}
		grpcClient.Enqueue(ctx, &rpc.Message{
			Event:       string(plugins.OnUserCreation),
			Data:        string(data),
			ClientUuid:  clientUuid,
			MessageUuid: uuid.NewString(),
		})
	})
}
