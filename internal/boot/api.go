package boot

import "os"

type Api struct {
	Port string
}

const DefaultPort = "8100"

func apiBoot() Api {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = DefaultPort
	}
	return Api{
		Port: port,
	}
}
