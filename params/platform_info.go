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
}{
	PlatformName:               "Thora Network",
	PlatformShortName:          "Thora", //Ethereum
	PlatformShortNameLowerCase: "thora", //ethereum
	CoinName:                   "THA",
	GETHCmd:                    "thora", //must be same with geth/main.go => clientIdentifier
	IPCPath:                    "thora.ipc",
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
