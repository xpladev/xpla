package dynamicdeflation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoCLIUsesStatusTerminology(t *testing.T) {
	options := (AppModule{}).AutoCLIOptions()
	require.NotNil(t, options)
	require.NotNil(t, options.Query)

	var found bool
	for _, command := range options.Query.RpcCommandOptions {
		if command.RpcMethod == "Status" {
			found = true
			require.Equal(t, "status", command.Use)
			require.Equal(t, "Query the Dynamic Deflation Pool account status", command.Short)
		}
	}
	require.True(t, found, "Status AutoCLI command is not registered")
}
