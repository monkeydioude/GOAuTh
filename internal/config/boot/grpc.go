package boot

import "os"

type Grpc struct {
	Port string
}

const GrpcDefaultPort = "9100"

// apiBoot returns API start related config
func grpcBoot() *Grpc {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = GrpcDefaultPort
	}
	return &Grpc{
		Port: ":" + port,
	}
}
