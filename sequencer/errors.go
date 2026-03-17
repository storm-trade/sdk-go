package sequencer

import "fmt"

type APIError struct {
	StatusCode        int    `json:"-"`
	Err               string `json:"error"`
	Msg               string `json:"msg,omitempty"`
	Message           string `json:"message,omitempty"`
	Code              string `json:"code,omitempty"`
	Description       string `json:"description,omitempty"`
	ContractExitCode  int    `json:"contract_exit_code,omitempty"`
	VmExitCode        int    `json:"vm_exit_code,omitempty"`
	ContractInterface string `json:"contract_interface,omitempty"`
}

func (e *APIError) Error() string {
	msg := e.Msg
	if msg == "" {
		msg = e.Message
	}
	if msg == "" {
		msg = e.Err
	}
	return fmt.Sprintf("sequencer API error %d: %s", e.StatusCode, msg)
}

func (e *APIError) IsEmulationError() bool {
	return e.Err == "EMULATE_ERROR"
}
