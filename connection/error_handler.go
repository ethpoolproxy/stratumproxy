package connection

import (
	log "github.com/sirupsen/logrus"
)

func PanicHandler() {
	err := recover()

	if err != nil {
		log.Errorf("====================== Panic Error ======================")
		log.Errorf("程序遇到严重错误，不会崩溃和影响已有矿机，建议重启和报告给开发者!")
		log.Errorf("TG 群: https://t.me/StratumProxy")
		log.Errorf("Github: https://github.com/ethpoolproxy/stratumproxy")
		log.Errorf("错误详细信息: ")
		log.Errorf("%+v", err)
		log.Errorf("=========================================================")
	}
}
