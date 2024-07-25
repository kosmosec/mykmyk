package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kosmosec/mykmyk/cmd/cli/cmd"
)

func main() {
	logFile, err := os.OpenFile("mykmyk.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("main", err)
	}

	defer logFile.Close()

	log.SetOutput(logFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		<-sigs
		fmt.Println("Aborted")
		cancel()
		os.Exit(1)
	}()

	rootCmd := cmd.NewRoot()
	//ctx, cancel := cancelableContext()
	//defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
