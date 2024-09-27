package main

import (
	"flag"
	"fmt"
	"log"
	"slices"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func show_help() {
	fmt.Println(`=== HELP ===
Args & flags:
  - flags:
    * method="api"|"rpc"
  - args:
    ./client "jwt" "status"|"refresh"
    ./client "auth" "login"|"signup"
	
For auth login/signup, login and password should be passed as env vars CLIENT_LOGIN & CLIENT_PASSWORD.
For jwt status/refresh, token should be passed as env var CLIENT_JWT.`)
}

var (
	methodsMap  = map[string]func(string, string) call{"api": newApiCall, "rpc": newRpcCall}
	servicesMap = map[string][]string{
		"auth": {"login", "signup"},
		"jwt":  {"status", "refresh"},
	}
)

type call interface {
	trigger() error
}

func setupCall(args []string, method string) call {
	var fn func(string, string) call
	var ok bool
	if fn, ok = methodsMap[method]; !ok {
		log.Fatalf("allowed methodsMap: %+v", methodsMap)
	}
	argService := args[0]
	var actions []string
	if actions, ok = servicesMap[argService]; !ok {
		log.Fatalf("allowed servicesMap: %+v", servicesMap)
	}

	argAction := args[1]
	if !slices.Contains(actions, argAction) {
		log.Fatalf("allowed actions: %+v", actions)
	}
	return fn(argService, argAction)
}

func main() {
	methodPtr := flag.String("method", "api", "api or rpc")
	flag.Parse()
	args := flag.Args()
	len := len(args)
	if len != 2 {
		if len == 1 && args[0] == "help" {
			show_help()
			return
		}
		log.Fatal("client requires 2 parameters: ")
		show_help()
	}
	call := setupCall(args, *methodPtr)
	if err := call.trigger(); err != nil {
		log.Fatal(err)
	}
}
