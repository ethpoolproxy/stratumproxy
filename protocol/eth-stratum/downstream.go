package eth_stratum

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
)

// RequestSubscribe 握手 mining.subscribe 数据包
type RequestSubscribe struct {
	Id int `json:"id"`
	// ["innominer/a10-1.1.0","EthereumStratum/1.0.0"]
	Params []string `json:"params"`
	Method string   `json:"method"`
}

func (resp *RequestSubscribe) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return errors.New(err.Error() + " | raw: " + string(data))
	}
	if resp.Method != "mining.subscribe" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.subscribe] but recived [%s]", resp.Method))
	}

	return nil
}

type RequestAuthorize struct {
	Id     int      `json:"id"`
	Params []string `json:"params"`
	Method string   `json:"method"`
	Worker string   `json:"worker"`
}

func (resp *RequestAuthorize) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if resp.Method != "mining.authorize" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.authorize] but recived [%s]", resp.Method))
	}
	if len(resp.Params) < 2 {
		return errors.New(fmt.Sprintf("Params mismatch expect [2] but recived [%d]", len(resp.Params)))
	}

	return nil
}

type RequestSubmit struct {
	Id     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func (resp *RequestSubmit) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}
	if resp.Method != "mining.submit" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.submit] but recived [%s]", resp.Method))
	}
	if len(resp.Params) < 3 {
		return errors.New(fmt.Sprintf("Params mismatch expect [3] but recived [%d]", len(resp.Params)))
	}

	return nil
}

func (resp RequestSubmit) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

type RequestGeneral struct {
	Id     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func (resp *RequestGeneral) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}
