package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

const (
	online  = "online"
	offline = "offline"
)

var (
	mode            string
	avalancheBin    string
	avalancheConfig string
	rosettaBin      string
	rosettaConfig   string
)

func init() {
	flag.StringVar(&mode, "mode", online, "Operation mode (online/offline)")
	flag.StringVar(&avalancheBin, "avalanche-bin", "", "Path to avalanche binary")
	flag.StringVar(&avalancheConfig, "avalanche-config", "", "Path to avalanche config")
	flag.StringVar(&rosettaBin, "rosetta-bin", "", "Path to rosetta binary")
	flag.StringVar(&rosettaConfig, "rosetta-config", "", "Path to rosetta config")
	flag.Parse()

	if !(mode == online || mode == offline) {
		log.Fatal("invalid mode: " + mode)
	}

	if mode == online {
		if avalancheConfig == "" {
			log.Fatal("avalanche config path is not provided")
		}
		if avalancheBin == "" {
			log.Fatal("avalanche binary path is not provided")
		}
	}

	if rosettaConfig == "" {
		log.Fatal("rosetta config path is not provided")
	}
	if rosettaBin == "" {
		log.Fatal("rosetta binary path is not provided")
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	handleSignals([]context.CancelFunc{cancel})

	g, gctx := errgroup.WithContext(ctx)

	if mode == online {
		g.Go(func() error {
			defer cancel()
			return startCommand(gctx, avalancheBin, "--config-file", avalancheConfig)
		})
	}

	g.Go(func() error {
		defer cancel()
		return startCommand(gctx, rosettaBin, "-config", rosettaConfig)
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}

func startCommand(ctx context.Context, path string, opts ...string) (err error) {
	log.Println("starting command:", path, opts)
	defer log.Println("command", path, "finished, error:", err)

	cmd := exec.Command(path, opts...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				panic(err)
			}
		}
	}()

	err = cmd.Run()
	return err
}

func handleSignals(listeners []context.CancelFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("received signal:", sig)
		for _, listener := range listeners {
			listener()
		}
	}()
}
