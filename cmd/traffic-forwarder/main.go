package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	_ConfigFile = flag.String("conf", "./etc/traffic-forwarder.conf", "The path of the configuration file.")
)

func RunTrafficForwarder(configFile string) bool {
	logrus.Infof("Loading setting file:%s.", configFile)
	fin, err := os.Open(configFile)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to load setting file:%s.", configFile)
		return false
	}
	defer fin.Close()
	logrus.Infof("Opened setting file:%s.", configFile)

	scanner := bufio.NewScanner(fin)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			logrus.Info("Skip blank line.")
			continue
		}
		if ok := strings.HasPrefix(line, "#"); ok {
			logrus.Infof("Skip comment line:%s.", line)
			continue
		}
		setting := strings.Split(line, "|")
		if len(setting) != 3 {
			logrus.Infof("Skip invalid setting:%s.", line)
			continue
		}
		logrus.Infof("Use line:'%s' to setup forwarding tunnel.", line)

		localPort, _ := strconv.Atoi(strings.TrimSpace(setting[0]))
		remoteHost := strings.TrimSpace(setting[1])
		remotePort, _ := strconv.Atoi(strings.TrimSpace(setting[2]))
		if localPort <= 0 || remotePort <= 0 {
			logrus.Infof("Skip invalid setting:%s.", line)
			continue
		}

		go func(lp int, rh string, rp int) {
			ln, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", lp))
			if err != nil {
				logrus.WithError(err).Errorf("Failed to listen on [::]:%d.", lp)
				return
			}
			logrus.Infof("Listening on [::]:%d.", lp)

			for {
				upstream, err := ln.Accept()
				if err != nil {
					logrus.WithError(err).Errorf("Failed to accept new connection on [::]:%d.", lp)
					continue
				}
				remote := upstream.RemoteAddr().String()
				logrus.Infof("Client<ip:%s> connected on [::]:%d.", remote, lp)

				downstream, err := net.Dial("tcp", fmt.Sprintf("%s:%d", rh, rp))
				if err != nil {
					logrus.WithError(err).Errorf("Failed to connect to %s:%d for client<ip:%s>.", rh, rp, remote)
					upstream.Close()
					continue
				}

				logrus.Infof("Forwarding traffic from 0.0.0.0:%d to %s:%d for client<ip:%s>.", lp, rh, rp, remote)
				// Transfer data from client to remote server.
				go Transfer(downstream, upstream)
				// Transfer data from remote server to client.
				go Transfer(upstream, downstream)
			}
		}(localPort, remoteHost, remotePort)
	}
	if err = scanner.Err(); err != nil {
		logrus.WithError(err).Errorf("Failed to continue to load setting file:%s.", *_ConfigFile)
		return false
	}

	return true
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	if *_ConfigFile == "" {
		logrus.Error("Invalid configuration file.")
		return
	}

	if RunTrafficForwarder(*_ConfigFile) {
		logrus.Info("Service started.")
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-signalCh
		logrus.Warning("Service stopped.")
	} else {
		logrus.Error("Failed to start service.")
	}
}
