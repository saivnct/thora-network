package params

import (
	"bytes"
	"text/template"
)

var PlatformChainInfo = struct {
	PlatformName               string
	PlatformShortName          string
	PlatformShortNameLowerCase string
	CoinName                   string
	GETHCmd                    string
	IPCPath                    string

	ENRPlatformProtocolName    string
	ENRPlatformProtocolVersion string

	TestPlatformName               string
	TestPlatformShortName          string
	TestPlatformShortNameLowerCase string
}{
	PlatformName:               "Thora Network",
	PlatformShortName:          "Thora", //Ethereum
	PlatformShortNameLowerCase: "thora", //ethereum
	CoinName:                   "THA",
	GETHCmd:                    "thora", //must be same with geth/main.go => clientIdentifier
	IPCPath:                    "thora.ipc",

	ENRPlatformProtocolName:    "thora",
	ENRPlatformProtocolVersion: "v1.0.0",

	TestPlatformName:               "Oda Test Network",
	TestPlatformShortName:          "Oda",
	TestPlatformShortNameLowerCase: "oda",
}

func WaterMarkText(inText string) string {
	tpl, err := template.New("").Parse(inText)

	var buf bytes.Buffer
	err = tpl.Execute(&buf, PlatformChainInfo)
	if err != nil {
		return inText
	}
	return buf.String()
}
