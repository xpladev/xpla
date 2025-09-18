package params

import (
	"strings"

	evmconfig "github.com/cosmos/evm/server/config"
)

var (
	// BypassMinFeeMsgTypesKey defines the configuration key for the
	// BypassMinFeeMsgTypes value.
	// nolint: gosec
	BypassMinFeeMsgTypesKey = "bypass-min-fee-msg-types"

	// customConfigTemplate defines XPLA's custom application configuration TOML template.
	customConfigTemplate = `
###############################################################################
###                        Custom XPLA Configuration                        ###
###############################################################################
# bypass-min-fee-msg-types defines custom message types the operator may set that
# will bypass minimum fee checks during CheckTx.
# NOTE:
# bypass-min-fee-msg-types = [] will deactivate the bypass - no messages will be allowed to bypass the minimum fee check
# bypass-min-fee-msg-types = [<MsgType>...] will allow messages of specified type to bypass the minimum fee check
# removing bypass-min-fee-msg-types from the config file will apply the default values:
# ["/ibc.core.channel.v1.MsgRecvPacket", "/ibc.core.channel.v1.MsgAcknowledgement", "/ibc.core.client.v1.MsgUpdateClient"]
#
# Example:
# bypass-min-fee-msg-types = ["/ibc.core.channel.v1.MsgRecvPacket", "/ibc.core.channel.v1.MsgAcknowledgement", "/ibc.core.client.v1.MsgUpdateClient"]
bypass-min-fee-msg-types = [{{ range .BypassMinFeeMsgTypes }}{{ printf "%q, " . }}{{end}}]
`
)

// CustomConfigTemplate defines XPLA's custom application configuration TOML
// template. It extends the core SDK template.
func CustomConfigTemplate(customAppTemplate string) string {
	lines := strings.Split(customAppTemplate, "\n")

	// remove minimue-gas-prices
	lines = append(lines[0:7], lines[12:]...)

	// add the XPLA config at the second line of the file
	lines[2] += customConfigTemplate
	return strings.Join(lines, "\n")
}

// CustomAppConfig defines Xpla's custom application configuration.
type CustomAppConfig struct {
	evmconfig.Config

	// BypassMinFeeMsgTypes defines custom message types the operator may set that
	// will bypass minimum fee checks during CheckTx.
	BypassMinFeeMsgTypes []string `mapstructure:"bypass-min-fee-msg-types"`
}
