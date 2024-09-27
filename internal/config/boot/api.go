package boot

import "os"

type Api struct {
	Port string
}

const ApiDefaultPort = "8100"

// apiBoot returns API start related config
func apiBoot() *Api {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = ApiDefaultPort
	}
	return &Api{
		Port: ":" + port,
	}
}
