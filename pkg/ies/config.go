package ies

type IETokens map[string]string

type IEConfigs struct {
	Tokens IETokens
}

var Cfg IEConfigs
