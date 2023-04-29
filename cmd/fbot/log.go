package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	envVarLogLevel = "FBOT_LOGLVL"
)

func init() {
	logLvl := logrus.InfoLevel
	if lvl := os.Getenv(envVarLogLevel); lvl != "" {
		var err error
		logLvl, err = logrus.ParseLevel(lvl)
		if err != nil {
			panic(fmt.Sprintf("%s: %s", envVarLogLevel, err))
		}
	}
	logrus.SetLevel(logLvl)
}
