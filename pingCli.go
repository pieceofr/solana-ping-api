package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

func solanaPing(c Cluster, count int, interval int, timeout int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var configpath string
	switch c {
	case MainnetBeta:
		configpath = config.SolanaConfigInfo.Dir + config.SolanaConfigInfo.MainnetPath
	case Testnet:
		configpath = config.SolanaConfigInfo.Dir + config.SolanaConfigInfo.TestnetPath
	case Devnet:
		configpath = config.SolanaConfigInfo.Dir + config.SolanaConfigInfo.DevnetPath
	default:
		configpath = config.SolanaConfigInfo.Dir + config.SolanaConfigInfo.DevnetPath
	}
	cmd := exec.CommandContext(ctx, "solana", "ping",
		"-c", fmt.Sprintf("%d", count),
		"-i", fmt.Sprintf("%d", interval),
		"-C", configpath)
	cmd.Env = append(os.Environ(), ":"+config.SolanaPing.PingExePath)
	stdin, err := cmd.StdinPipe()

	if err != nil {
		log.Println(c, ":Ping StdinPipe Error:", err)
		return "", err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, "values written to stdin are passed to cmd's standard input")
	}()

	out, err := cmd.Output()
	if err != nil {
		log.Println(c, ":Ping Output Error:", err)
		return "", err
	}
	return string(out), nil
}
