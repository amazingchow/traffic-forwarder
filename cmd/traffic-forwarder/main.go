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
	_ConfigFile = flag.String("f", "./etc/traffic-forwarder.conf", "the setting file")
)

func runTrafficForwarder(fConf string) bool {
	logrus.Infof("Ready to load setting file:%s.", fConf)

	fin, err := os.Open(fConf)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to load setting file:%s.", fConf)
		return false
	}
	defer fin.Close()

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
		setting := strings.Split(line, " ")
		if len(setting) != 3 {
			logrus.Infof("Skip invalid line:%s.", line)
			continue
		}
		logrus.Infof("Use line:'%s' to setup forwarding tunnel.", line)

		localPort, _ := strconv.Atoi(setting[0])
		remoteHost := setting[1]
		remotePort, _ := strconv.Atoi(setting[2])
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
				logrus.Infof("Remote %s connected on [::]:%d.", remote, lp)

				downstream, err := net.Dial("tcp", fmt.Sprintf("%s:%d", rh, rp))
				if err != nil {
					logrus.WithError(err).Errorf("Failed to listen on %s:%d.", rh, rp)
					upstream.Close()
					continue
				}

				logrus.Infof("Forwarding traffic from 0.0.0.0:%d to %s:%d for remote:%s.", lp, rh, rp, upstream.RemoteAddr().String())
				go Transfer(upstream, downstream)
				go Transfer(downstream, upstream)
			}
		}(localPort, remoteHost, remotePort)
	}
	if err = scanner.Err(); err != nil {
		logrus.WithError(err).Errorf("Failed to load setting file:%s.", *_ConfigFile)
		return false
	}

	logrus.Infof("Loaded setting file:%s.", *_ConfigFile)
	return true
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	if runTrafficForwarder(*_ConfigFile) {
		logrus.Info("## Start Service")
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-signalCh
		logrus.Info("## Stop Service")
	} else {
		logrus.Error("## CANNOT Start Service")
	}
}
