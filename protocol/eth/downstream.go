package eth

import (
	"errors"
	"github.com/goccy/go-json"
	"strconv"
)

var MethodNotMatchErr = errors.New("method not match")

// RequestSubmitLogin 登陆请求
type RequestSubmitLogin struct {
	Compact bool   `json:"compact"`
	Id      int    `json:"id"`
	Method  string `json:"method"`
	// 0: 钱包地址 | 1: 密码
	Params []string `json:"params"`
	// 矿工名字
	Worker string `json:"worker"`
}

func (resp *RequestSubmitLogin) Valid() error {
	if resp.Method != "eth_submitLogin" {
		return MethodNotMatchErr
	}

	if len(resp.Params) < 2 {
		return errors.New("not enough arg")
	}

	return nil
}

func (resp *RequestSubmitLogin) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

// RequestHashratePack 提交本地算力
type RequestHashratePack struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
	// 0: hashrate
	Params []string `json:"params"`
	// 矿工名字
	Worker   string `json:"worker"`
	Hashrate int64  `json:"-"`
}

func (resp *RequestHashratePack) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	if len(resp.Params) < 1 {
		return errors.New("not enough arg")
	}

	hashrate, err := strconv.ParseUint(resp.Params[0][2:], 16, 64)
	if err != nil {
		return err
	}
	resp.Hashrate = int64(hashrate)

	return nil
}

func (resp *RequestHashratePack) Valid() error {
	if resp.Method != "eth_submitHashrate" {
		return MethodNotMatchErr
	}

	if len(resp.Params) < 1 || resp.Hashrate < 0 {
		return errors.New("not enough arg")
	}

	return nil
}

func (resp RequestHashratePack) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

type RequestSubmitWork struct {
	Id     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
	Worker string   `json:"worker"`
}

func (resp *RequestSubmitWork) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	if len(resp.Params) < 2 {
		return errors.New("not enough arg")
	}

	return nil
}

func (resp *RequestSubmitWork) Valid() error {
	if resp.Method != "eth_submitWork" {
		return MethodNotMatchErr
	}

	return nil
}

func (resp RequestSubmitWork) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

type RequestGetWork struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
}

func (resp *RequestGetWork) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (resp *RequestGetWork) Valid() error {
	if resp.Method != "eth_getWork" {
		return MethodNotMatchErr
	}

	return nil
}
