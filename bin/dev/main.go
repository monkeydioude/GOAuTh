// dev gathers a developer's shortcuts against the goauth database.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func show_help() {
	fmt.Fprintln(os.Stderr, `=== HELP ===
Args:
	./dev "access-key" "create" <realm> <login>   (an access key that never expires, see README)

The database is read from the DB_PATH and DB_SCHEMA env vars.`)
	os.Exit(1)
}

func main() {
	godotenv.Load()
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		show_help()
	}
	switch args[0] {
	case "access-key":
		if len(args) != 4 || args[1] != "create" {
			show_help()
		}
		accessKeyCreate(args[2], args[3])
	default:
		show_help()
	}
}
