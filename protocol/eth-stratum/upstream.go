package eth_stratum

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"gopkg.in/guregu/null.v4"
)

// ResponseGeneral 通用的返回格式
type ResponseGeneral struct {
	Id     int         `json:"id"`
	Result bool        `json:"result"`
	Error  null.String `json:"error"`
}

func (resp *ResponseGeneral) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (resp ResponseGeneral) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseMethodGeneral 通用的方法请求
type ResponseMethodGeneral struct {
	Id     null.Int      `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

func (resp ResponseMethodGeneral) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseHandshakeNotify 握手 mining.notify 数据包
type ResponseHandshakeNotify struct {
	Id int `json:"id"`
	// ["mining.notify","0000","EthereumStratum/1.0.0"]
	// "00"
	Result []interface{} `json:"result"`
	Error  string        `json:"error"`
}

func (resp *ResponseHandshakeNotify) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return errors.New(err.Error() + " | raw: " + string(data))
	}

	return nil
}

func (resp ResponseHandshakeNotify) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseMiningNotify 握手 mining.notify 数据包
type ResponseMiningNotify struct {
	Id     null.Int      `json:"id"`
	Result []interface{} `json:"result"`
	Method string        `json:"method,omitempty"`
	Error  null.String   `json:"error"`
}

func (resp *ResponseMiningNotify) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return errors.New(err.Error() + " | raw: " + string(data))
	}
	if resp.Method != "mining.notify" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.notify] but recived [%s]", resp.Method))
	}
	if len(resp.Result) < 4 {
		return errors.New(fmt.Sprintf("mining.notify param mismatch expect 4 but recived [%d]", len(resp.Result)))
	}

	return nil
}

func (resp ResponseMiningNotify) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseMiningSetDifficulty 握手 mining.set_difficulty 数据包
type ResponseMiningSetDifficulty struct {
	Id     int       `json:"id"`
	Params []float64 `json:"params"`
	Method string    `json:"method"`
}

func (resp *ResponseMiningSetDifficulty) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return errors.New(err.Error() + " | raw: " + string(data))
	}
	if resp.Method != "mining.set_difficulty" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.set_difficulty] but recived [%s]", resp.Method))
	}
	if len(resp.Params) < 1 {
		return errors.New(fmt.Sprintf("mining.set_difficulty param mismatch expect 1 but recived [%d]", len(resp.Params)))
	}

	return nil
}

func (resp ResponseMiningSetDifficulty) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseNotify mining.notify
type ResponseNotify struct {
	Id int `json:"id"`
	// string string string bool
	Params []interface{} `json:"params"`
	Method string        `json:"method"`
	Height int           `json:"height"`
}

func (resp *ResponseNotify) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return errors.New(err.Error() + " | raw: " + string(data))
	}
	if resp.Method != "mining.notify" {
		return errors.New(fmt.Sprintf("Method mismatch expect [mining.notify] but recived [%s]", resp.Method))
	}
	if len(resp.Params) < 4 {
		return errors.New(fmt.Sprintf("mining.set_difficulty param mismatch expect 1 but recived [%d]", len(resp.Params)))
	}

	return nil
}
