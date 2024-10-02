package main

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type rpcCall struct {
	apiCall
}

func setupRPCRequest() (*grpc.ClientConn, error) {
	return grpc.NewClient(
		"[::]:9100",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func (c rpcCall) trigger() error {
	fmt.Printf("Sending request: %+v\n", c)
	var res *v1.Response
	var err error
	var headerMD metadata.MD
	conn, err := setupRPCRequest()
	ctx := context.Background()
	if err != nil {
		return err
	}
	ctx = rpc.WriteOutgoingMetas(ctx, [2]string{consts.X_REQUEST_ID_LABEL, uuid.NewString()})
	switch c.service {
	case "auth":
		switch c.action {
		case "login":
			client := v1.NewAuthClient(conn)
			res, err = client.Login(
				ctx,
				&v1.UserRequest{
					Login:    os.Getenv("CLIENT_LOGIN"),
					Password: os.Getenv("CLIENT_PASSWORD"),
				},
				grpc.Header(&headerMD),
			)
		case "signup":
			client := v1.NewAuthClient(conn)
			res, err = client.Signup(
				ctx,
				&v1.UserRequest{
					Login:    os.Getenv("CLIENT_LOGIN"),
					Password: os.Getenv("CLIENT_PASSWORD"),
				},
				grpc.Header(&headerMD),
			)
		}
	case "jwt":
		switch c.action {
		case "status":
			client := v1.NewJWTClient(conn)
			// ctx = services.AddAuthorizationTokenMetaIn(ctx, os.Getenv("CLIENT_JWT"))
			ctx = rpc.AddOutgoingCookie(ctx, http.Cookie{
				Name:  consts.AuthorizationCookie,
				Value: "Bearer " + os.Getenv("CLIENT_JWT"),
			})
			res, err = client.Status(
				ctx,
				&v1.Empty{},
				grpc.Header(&headerMD),
			)
		case "refresh":
			client := v1.NewJWTClient(conn)
			ctx = rpc.AddOutgoingCookie(ctx, http.Cookie{
				Name:  consts.AuthorizationCookie,
				Value: "Bearer " + os.Getenv("CLIENT_JWT"),
			})
			res, err = client.Refresh(
				ctx,
				&v1.Empty{},
				grpc.Header(&headerMD),
			)
		}
	}
	if err != nil {
		return err
	}
	err = conn.Close()
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("nil response")
	}
	fmt.Printf("Response: %d\n%s\nHeaders: %+v\n", res.Code, res, headerMD)
	return err
}

func newRpcCall(service, action string) call {
	return rpcCall{apiCall{service, action}}
}
