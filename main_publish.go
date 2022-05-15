// +build publish_log

package main

import log "github.com/sirupsen/logrus"

func InitMain() {
	log.SetLevel(log.InfoLevel)
}

func DeferMain() {

}
