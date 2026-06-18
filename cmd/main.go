package main

import (
	"context"
	"os"

	scout "github.com/tomnagengast/scout/internal"
)

func main() {
	if err := scout.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
