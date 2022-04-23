package main

import (
	"errors"
	"flag"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"stratumproxy/config"
	"stratumproxy/connection"
	"stratumproxy/injector/eth"
	"stratumproxy/util"
	"stratumproxy/webui"
	"syscall"
	"time"
)

var profileType string

func main() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	flag.StringVar(&config.ConfigFile, "config", "config.yml", "配置文件路径")
	flag.StringVar(&profileType, "profile", "", "性能监控")
	flag.Parse()

	// 初始化
	InitMain()
	defer DeferMain()

	log.Infof("加载配置文件 [%s]", config.ConfigFile)
	_, err := os.Stat(config.ConfigFile)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		example, _ := config.ExampleConfigFile.ReadFile("config.example.yml")
		err = ioutil.WriteFile(config.ConfigFile, example, 0755)
		if err != nil {
			log.Fatalf("无法写入配置文件 [%s]: %s", config.ConfigFile, err.Error())
			return
		}
	}

	err = config.LoadConfig(config.ConfigFile)
	if err != nil {
		log.Fatalf("无法加载配置文件 [%s]: %s", config.ConfigFile, err.Error())
		return
	}
	go func() {
		time.Sleep(1 * time.Minute)
		err = config.SaveConfig(config.ConfigFile)
		if err != nil {
			log.Errorf("无法保存配置文件 [%s]！请及时记录当前配置并关闭软件，否则可能造成当前配置丢失！ [%s]", config.ConfigFile, err.Error())
			return
		}
	}()

	go func() {
		log.Infof("在 [%s] 上启动在线面板", config.GlobalConfig.WebUI.Bind)
		err = webui.StartWebServer()
		if err != nil {
			log.Fatalf("无法启动在线面板 [%s]", err.Error())
		}
	}()

	// 加载协议
	eth.RegisterProtocol()

	// 启动矿池
	for _, pool := range config.GlobalConfig.Pools {
		server, err := connection.NewPoolServer(pool)
		if err != nil {
			log.Errorf("无法启动矿池 [%s]: %s", pool.Name, err)
			continue
		}

		go func() { _ = server.Start() }()
	}

	// 生成密码
	if config.GlobalConfig.WebUI.Auth.Username == "" || config.GlobalConfig.WebUI.Auth.Passwd == "" {
		config.GlobalConfig.WebUI.Auth.Username = util.GetRandomString2(6)
		config.GlobalConfig.WebUI.Auth.Passwd = util.GetRandomString2(12)
		log.Infof("初始管理员凭据已生成！登陆后台后请及时更改！用户名 [%s] 密码 [%s]", config.GlobalConfig.WebUI.Auth.Username, config.GlobalConfig.WebUI.Auth.Passwd)
		_ = config.SaveConfig(config.ConfigFile)
	}

	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
	log.Errorf("接收到 [%s] 信号，程序退出! ", <-chSig)

	log.Infof("保存配置文件 [%s]", config.ConfigFile)
	err = config.SaveConfig(config.ConfigFile)
	if err != nil {
		log.Errorf("无法保存配置文件 [%s]: %s", config.ConfigFile, err.Error())
		return
	}
}
