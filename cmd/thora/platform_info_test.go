package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"testing"
)

func TestPlatformInfo(t *testing.T) {
	textSrc := `{{.PlatformName}} test`
	output := params.WaterMarkText(textSrc)
	expected := fmt.Sprintf("%v test", params.PlatformChainInfo.PlatformName)
	log.Info(output)
	if output != expected {
		t.Errorf("TestPlatformInfo failed, expected %v, got %v", expected, output)
	}

	textSrc = "{{.PlatformShortName}} test"
	output = params.WaterMarkText(textSrc)
	expected = fmt.Sprintf("%s test", params.PlatformChainInfo.PlatformShortName)
	log.Info(output)
	if output != expected {
		t.Errorf("TestPlatformInfo failed, expected %v, got %v", expected, output)
	}

	textSrc = `{{.PlatformShortNameLowerCase}} test`
	output = params.WaterMarkText(textSrc)
	expected = fmt.Sprintf("%s test", params.PlatformChainInfo.PlatformShortNameLowerCase)
	log.Info(output)
	if output != expected {
		t.Errorf("TestPlatformInfo failed, expected %v, got %v", expected, output)
	}

	textSrc = `{{.CoinName}} test`
	output = params.WaterMarkText(textSrc)
	expected = fmt.Sprintf("%s test", params.PlatformChainInfo.CoinName)
	log.Info(output)
	if output != expected {
		t.Errorf("TestPlatformInfo failed, expected %v, got %v", expected, output)
	}

	textSrc = `{{.GETHCmd}} test`
	output = params.WaterMarkText(textSrc)
	expected = fmt.Sprintf("%s test", params.PlatformChainInfo.GETHCmd)
	log.Info(output)
	if output != expected {
		t.Errorf("TestPlatformInfo failed, expected %v, got %v", expected, output)
	}
}
