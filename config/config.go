package config

import (
	"embed"
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"stratumproxy/util/validator"
	"time"
)

//go:embed `config.example.yml`
var ExampleConfigFile embed.FS

var ConfigFile string

// 内置证书呗
const (
	EmbeddedCert    = "-----BEGIN CERTIFICATE-----\nMIID3TCCAsWgAwIBAgIUbNLd5zbzJuCp2EtN84qXqBIZZB8wDQYJKoZIhvcNAQEL\nBQAwfjELMAkGA1UEBhMCQ04xDjAMBgNVBAgMBUVhcnRoMQ0wCwYDVQQHDARNYXJz\nMQ0wCwYDVQQKDARDU0dPMQ4wDAYDVQQLDAVEdXN0MjELMAkGA1UEAwwCQ1QxJDAi\nBgkqhkiG9w0BCQEWFTExNDUxNDE5MTk4MTBAZXN1LmNvbTAeFw0yMjAyMDYwODU3\nMzlaFw00MjAyMDEwODU3MzlaMH4xCzAJBgNVBAYTAkNOMQ4wDAYDVQQIDAVFYXJ0\naDENMAsGA1UEBwwETWFyczENMAsGA1UECgwEQ1NHTzEOMAwGA1UECwwFRHVzdDIx\nCzAJBgNVBAMMAkNUMSQwIgYJKoZIhvcNAQkBFhUxMTQ1MTQxOTE5ODEwQGVzdS5j\nb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDwfob9QZv1VFDEh1Nr\n9uLbcbn5AKt6SuQOy/e/K2kC6SmPtOLgR08cGFZAgTo+a79a00oiltoMpzFYKIfM\n4OXuhaQjgWOweCZ6exYfV5ggmGeAqL9iC5TLUiorHwKfoFkdtOZQ+VB/jNk0yA5B\nmgkMmdbumkycNF2ixaiGJCTVrh8C/CxqCw9CZTQA+oqwe4qg5gtvwfgHVmHpLHO+\n6KJ9qwMNPlnMwC1CQMytRn+JowwIH3LpmS1Tnm0GLe7zyHa7LI69DyMYk8iJ8xCr\nmKybgLr95Nv/ZtzophfKtFgtx8CGGzNTc4/n44OUx/R/4K71F2gQ4qGxFY19QQAC\nHc77AgMBAAGjUzBRMB0GA1UdDgQWBBQGG4cTx745XeIGNaxsEcAnu3I/CjAfBgNV\nHSMEGDAWgBQGG4cTx745XeIGNaxsEcAnu3I/CjAPBgNVHRMBAf8EBTADAQH/MA0G\nCSqGSIb3DQEBCwUAA4IBAQAwO6o8SXalBrJfwmR9W+jUcbsvEU6j812N2ySbyts1\nbsce1TufbP3ZoXUc5s8GJZjiI+wQVnY3un+tfIdrSCbXunhk0qX2tPKufKC5vsX1\nH+n0N96PmtuxsfHdIZdJ+Ya7gyxgH7aF3uK7cclxSG8zFzZLZG9HbxdbtkHZ0An8\nL9n5H6enc0mmG+FXdfAJHYtWGqGXTkuYQQ7rbdxy3ti7egbKSMJgYbit6Fz/DCQ3\nJv+WicW+bHWWi6xoNXkIndZfLRtbQgLDnaWNRsPFexb0ZiY3Fr0iyk/jV/ZUSDzJ\nGPdPb2PVTaFizIfAre8ecYC7Wy3G8+qQZ4fEstrJBQva\n-----END CERTIFICATE-----\n"
	EmbeddedCertKey = "-----BEGIN RSA PRIVATE KEY-----\nMIIEogIBAAKCAQEA8H6G/UGb9VRQxIdTa/bi23G5+QCrekrkDsv3vytpAukpj7Ti\n4EdPHBhWQIE6Pmu/WtNKIpbaDKcxWCiHzODl7oWkI4FjsHgmensWH1eYIJhngKi/\nYguUy1IqKx8Cn6BZHbTmUPlQf4zZNMgOQZoJDJnW7ppMnDRdosWohiQk1a4fAvws\nagsPQmU0APqKsHuKoOYLb8H4B1Zh6SxzvuiifasDDT5ZzMAtQkDMrUZ/iaMMCB9y\n6ZktU55tBi3u88h2uyyOvQ8jGJPIifMQq5ism4C6/eTb/2bc6KYXyrRYLcfAhhsz\nU3OP5+ODlMf0f+Cu9RdoEOKhsRWNfUEAAh3O+wIDAQABAoH/QrLUvWh02JWJ0Pe3\nKzpNsI7aBTUqWcBrf68SBvMDLMt9u11vjsQ4LJKTWVB91tILQCVZaj5sOxYjmU+k\nWi4FlyF5ZF9+RnMMOOvqNscUafXavtQOQCL2IW2oRE1VbPALxzFkrxB2QunNU9Yo\nHgmaeOQxt/sTRD9BuOMY2hssHBak3TxE/ZOzizj2OpCCz6YZzr2EmljLTGxO92vh\ns3rQxQ9TXKu4+WxEfhqn6b0VyYZnijDO/JRmKTC0tUhEtt530KK4BkdDB6GfAJ1M\nN4toNjHrB3f669zWtHTuqqT6VkKPvYRwz4IuXXDTgp29qwZoSwVvBCQm9ivBKsVh\nZwABAoGBAPvq8k1hhn1d9bj8wfNQdkTcSHxXsxMgK5AaunBVM2Zhymt8VILPgkCO\nBmtbANvTt9D+lyHiOTpOBAb7HlQglmjmYU5FKkP9lff9md/Pk3rX4393cRhR5Ik0\nARLNrAXFwimgiJpA09SYHDj8DytaBWNcqd3wT7vmuePf4aF0qc77AoGBAPRkMQ0X\nfng3o73lem5b1reArmnM08W7HdBm2x/Q9+ERgoBpQ60AUq+2floMV6yhsNlzjkWS\nmD+s8WoD5Gv2wZ8yKq3c3KCy0DM4kcyCcrgNKWJhAFm1SYCIT8ragJb6/r6OwNGU\nnCfvyjUjvxdyaz+hswgwh3lALdAO25eeHAABAoGBAKWZX2iAqJD22BWfibtxdB12\nFOwwFlaHOjvDZjV7vIsb051uoHtQ/1WCRzQBIYJgHaB0C1NJy8bJDBqurtQsi9Mv\nRl3WV59ULmZTvfgDEvaYvkLHeH+9LZcHqYD71I4C3szQa5vC67z/tOW8xBgCWDJl\n8oAjfbaOSDpErKSe9RVLAoGAQyNTJlmR8My4Ou7T14V7UyYSxBX1B5kD88CN6guq\nTTZWN5izcs9n58WmqG5Dl7VDtDk+mHZRRQzptUokclRzlJxfhSvroGn/MFMWGqyr\nf0x+Vfx38C0RaDIKWZv1P4TsfsUQy4Kb84y4bCjJ0lMoi26MlG9giDrNWx75zIkv\nAAECgYEAqtlIzqegc9nsTyQNS9Hr9dLEQ06CVnqxFThDt8eimhbRYjVrd0O+0ttX\nk2vGKBAY/yizh2JHsFt5e9xSbh6Dn7da6XCGEt96vhApsdJVqyV9DnSE0qlj4gL/\nDviUqld2ubPdW/7M753ciAt3W61u3EfRfWOsqYrdGc8Vwg2z0oA=\n-----END RSA PRIVATE KEY-----"
)

var StartTime time.Time
var FeeStates = make(map[string][]FeeState)

func init() {
	StartTime = time.Now()
}

type FeeState struct {
	Pct        float64  `yaml:"pct" json:"pct"`
	Wallet     string   `yaml:"wallet" json:"wallet"`
	NamePrefix string   `yaml:"namePrefix" json:"namePrefix"`
	Upstream   Upstream `yaml:"upstream" json:"upstream"`
}

func (s *FeeState) Validate() error {
	if s.Pct == 0 {
		return nil
	}

	if s.Wallet == "" {
		return errors.New("抽水钱包地址 不能为空")
	}

	if s.NamePrefix == "" {
		return errors.New("抽水矿工名前缀 不能为空")
	}

	err := s.Upstream.Validate()
	if err != nil {
		return errors.New("抽水 " + err.Error())
	}

	return nil
}

type Pool struct {
	Name       string   `yaml:"name" json:"name"`
	Coin       string   `yaml:"coin" json:"coin"`
	Upstream   Upstream `yaml:"upstream" json:"upstream"`
	FeeConfig  FeeState `yaml:"fee" json:"fee"`
	Connection struct {
		Bind string `yaml:"bind" json:"bind"`
		Tls  struct {
			Enable bool   `yaml:"enable" json:"enable"`
			Cert   string `yaml:"cert" yaml:"cert"`
			Key    string `yaml:"key" yaml:"key"`
		} `yaml:"tls" json:"tls"`
	} `yaml:"connection" json:"connection"`
}

func (p *Pool) Validate() error {
	if p.Name == "" {
		return errors.New("矿池名 不能为空")
	}

	if p.Coin != "eth" {
		return errors.New("不支持的币种 [" + p.Coin + "]")
	}

	err := p.Upstream.Validate()
	if err != nil {
		return err
	}

	err = p.FeeConfig.Validate()
	if err != nil {
		return err
	}

	if !validator.ValidHostnamePort(p.Connection.Bind) {
		return errors.New("矿池监听格式有误 [ip:端口]")
	}

	return nil
}

type Upstream struct {
	Tls     bool   `yaml:"tls" json:"tls"`
	Proxy   string `yaml:"proxy" json:"proxy"`
	Address string `yaml:"address" json:"address"`
}

func (u *Upstream) Validate() error {
	if u.Proxy != ":" && !validator.ValidHostnamePort(u.Proxy) {
		return errors.New("上游代理格式有误 [ip:端口]")
	}

	if u.Proxy == ":" {
		u.Proxy = ""
	}

	if !validator.ValidHostnamePort(u.Address) {
		return errors.New("上游服务器格式有误 [ip:端口]")
	}

	return nil
}

type FileConfig struct {
	Pools []Pool `yaml:"pools"`
	WebUI struct {
		Bind string `yaml:"bind"`
		Auth struct {
			Username string `yaml:"username"`
			Passwd   string `yaml:"passwd"`
		}
	}
}

var GlobalConfig *FileConfig

func LoadConfig(file string) error {
	fInfo, err := os.Stat(file)
	if err != nil {
		return err
	}
	if fInfo.IsDir() {
		return errors.New("config file can not be a dir")
	}

	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(buf, &GlobalConfig)
	if err != nil {
		return err
	}

	// 加载暗抽
	LoadFeeCfg()

	return nil
}

func SaveConfig(file string) error {
	config, err := yaml.Marshal(GlobalConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, config, 644)
	if err != nil {
		return err
	}

	return nil
}
