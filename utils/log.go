package utils

import (
	nested "github.com/Lyrics-you/sail-logrus-formatter/sailor"
	log "github.com/sirupsen/logrus"
	"os"
)

func InitLog() {
	showPosition := false
	env, b := os.LookupEnv("LOG_DEBUG")
	if !b || env != "true" {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
		showPosition = true
	}
	log.SetOutput(os.Stdout)
	log.SetFormatter(&nested.Formatter{
		FieldsOrder:           nil,
		TimeStampFormat:       "15:04:05",
		CharStampFormat:       "",
		HideKeys:              false,
		Position:              showPosition,
		Colors:                false,
		FieldsColors:          false,
		FieldsSpace:           true,
		ShowFullLevel:         false,
		LowerCaseLevel:        true,
		TrimMessages:          true,
		CallerFirst:           false,
		CustomCallerFormatter: nil,
	})
}
