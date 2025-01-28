package main

import (
	"flag"
	"log"
	"os"
	"slices"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func show_help() {
	log.Println(`=== HELP ===
Args & flags:
  - flags:
    * method="api"|"rpc"
  - args:
	./client "auth" "login"|"signup"
    ./client "user" "password"|"deactivate"
    ./client "jwt" "status"|"refresh"
    ./client "realm" "add"|"view" <if add:"name of the realm">

For auth login/signup, login and password should be passed as env vars CLIENT_LOGIN & CLIENT_PASSWORD.
For jwt status/refresh, token should be passed as env var CLIENT_JWT.`)
	os.Exit(1)
}

var (
	methodsMap  = map[string]func(string, string) call{"api": newApiCall, "rpc": newRpcCall}
	servicesMap = map[string][]string{
		"auth":  {"login", "signup"},
		"user":  {"password", "login", "deactivate"},
		"jwt":   {"status", "refresh"},
		"realm": {"create", "view"},
	}
)

type call interface {
	trigger() error
}

func setupCall(args []string, method string) call {
	var fn func(string, string) call
	var ok bool

	argService := args[0]
	var actions []string
	if actions, ok = servicesMap[argService]; !ok {
		log.Fatalf("allowed servicesMap: %+v", servicesMap)
	}
	if fn, ok = methodsMap[method]; !ok {
		log.Fatalf("allowed methodsMap: %+v", methodsMap)
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
	if len == 1 && args[0] == "help" {
		show_help()
		return
	}
	call := setupCall(args, *methodPtr)
	if err := call.trigger(); err != nil {
		log.Fatal(err)
	}
}
