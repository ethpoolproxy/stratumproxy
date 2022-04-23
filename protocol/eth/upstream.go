package eth

import (
	"errors"
	"github.com/goccy/go-json"
)

// ResponseGeneral 通用的返回格式
type ResponseGeneral struct {
	Id      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  bool   `json:"result"`
	Error   string `json:"error"`
}

func (resp *ResponseGeneral) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (resp ResponseGeneral) Build() ([]byte, error) {
	resp.Jsonrpc = "2.0"

	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

// ResponseSubmitLogin 登陆结果
type ResponseSubmitLogin struct {
	Id     int    `json:"id"`
	Result bool   `json:"result"`
	Error  string `json:"error"`
}

func (resp *ResponseSubmitLogin) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (resp *ResponseSubmitLogin) Valid() error {
	if !resp.Result && resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (resp ResponseSubmitLogin) Build() ([]byte, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

type ResponseWorkerJob struct {
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Height  int      `json:"height"`
	Result  []string `json:"result"`
}

func (resp *ResponseWorkerJob) Parse(data []byte) error {
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (resp *ResponseWorkerJob) Valid() error {
	if (len(resp.Result) != 3 && len(resp.Result) != 4) && resp.Id != 0 && !(resp.Height > 0) {
		return errors.New("invalid job")
	}
	return nil
}

func (resp ResponseWorkerJob) Build() ([]byte, error) {
	resp.Jsonrpc = "2.0"

	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}

type ResponseSubmitWork struct {
	Id      int    `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  bool   `json:"result"`
}

func (resp ResponseSubmitWork) Build() ([]byte, error) {
	resp.Jsonrpc = "2.0"

	b, err := json.Marshal(resp)
	if err != nil {
		return []byte{}, err
	}
	b = append(b, '\n')

	return b, nil
}
