package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Printf("worker bootstrap started")
	log.Printf("worker mode: dispatch placeholder active")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case sig := <-sigCh:
			log.Printf("worker shutdown on signal %s", sig.String())
			return
		case <-ticker.C:
			log.Printf("worker heartbeat")
		}
	}
}
