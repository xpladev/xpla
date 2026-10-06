package wasm_test

import (
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

const wasmIntegrationSubprocessEnv = "XPLA_WASM_INTEGRATION_SUBPROCESS"

// The pinned EVM dependency stores coin metadata in process-global state. Each
// integration scenario constructs its own app and must therefore run in a fresh
// process, just like a real node start.
func runInIsolatedProcess(t *testing.T) bool {
	t.Helper()
	if os.Getenv(wasmIntegrationSubprocessEnv) == t.Name() {
		return false
	}

	runTestInSubprocess(t, wasmIntegrationSubprocessEnv, t.Name())
	return true
}

func runTestInSubprocess(t *testing.T, envKey, envValue string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$", "-test.v")
	command.Env = append(os.Environ(), envKey+"="+envValue)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	t.Log(string(output))
}
