package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/offchainlabs/nitro/daprovider/celestiamock"
)

func main() {
	addr := flag.String("addr", "0.0.0.0", "listen address")
	port := flag.Uint64("port", 9880, "listen port")
	maxMsgSize := flag.Int("max-message-size", 32*1024*1024, "max message size in bytes")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store := celestiamock.NewStore()
	srv := celestiamock.NewServer(store)
	srv.SetMaxMessageSize(*maxMsgSize)

	_, ln, err := celestiamock.StartRPC(ctx, *addr, *port, srv)
	if err != nil {
		log.Fatalf("failed to start mock celestia DA server: %v", err)
	}
	log.Printf("mock celestia DA server listening on %s", ln.Addr().String())

	<-ctx.Done()
	log.Printf("mock celestia DA server shutting down")
}
