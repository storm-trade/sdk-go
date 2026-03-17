package sequencer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Err:        "bad_request",
		Msg:        "invalid address",
	}

	require.Contains(t, err.Error(), "400")
	require.Contains(t, err.Error(), "invalid address")

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, 400, apiErr.StatusCode)
}

func TestAPIError_ErrorFallback(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Err:        "internal",
	}
	require.Contains(t, err.Error(), "internal")
}

func TestAPIError_EmulationError(t *testing.T) {
	err := &APIError{
		StatusCode:        400,
		Err:               "EMULATE_ERROR",
		Message:           "contract error",
		ContractExitCode:  123,
		VmExitCode:        456,
		ContractInterface: "vamm",
	}

	require.True(t, err.IsEmulationError())
	require.Contains(t, err.Error(), "contract error")
	require.Equal(t, 123, err.ContractExitCode)
	require.Equal(t, 456, err.VmExitCode)
	require.Equal(t, "vamm", err.ContractInterface)
}
