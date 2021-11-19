package main

import (
	"strings"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/valyala/fastjson"
)

const tickMilliseconds uint32 = 5000

var allowedDomains []string

func main() {
	proxywasm.SetVMContext(&vmContext{})
}

type vmContext struct {
	// Embed the default VM context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultVMContext
}

// Override types.DefaultVMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{contextID: contextID}
}

type pluginContext struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext
	contextID uint32
	callBack  func(numHeaders, bodySize, numTrailers int)
	apiHost   string
}

// Override types.DefaultPluginContext.
func (*pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpAuthRandom{contextID: contextID}
}

type httpAuthRandom struct {
	// Embed the default http context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultHttpContext
	contextID uint32
}

// Override types.DefaultPluginContext.
func (ctx *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	configdata, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	s := fastjson.GetString(configdata, "api_host")
	if s == "" {
		proxywasm.LogCriticalf("api_host is not set")
		return types.OnPluginStartStatusFailed
	}
	ctx.apiHost = s

	if err := proxywasm.SetTickPeriodMilliSeconds(tickMilliseconds); err != nil {
		proxywasm.LogCriticalf("failed to set tick period: %v", err)
		return types.OnPluginStartStatusFailed
	}
	proxywasm.LogInfof("set tick period milliseconds: %d", tickMilliseconds)
	ctx.callBack = func(numHeaders, bodySize, numTrailers int) {
		respBody, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
		if err != nil {
			proxywasm.LogErrorf("failed to get http response body: %v", err)
			return
		}
		var p fastjson.Parser
		v, err := p.Parse(string(respBody))
		if err != nil {
			proxywasm.LogErrorf("failed to parse response body: %v", err)
			return
		}
		allowedDomains = []string{}
		for _, domain := range v.GetArray() {
			allowedDomains = append(allowedDomains, string(domain.GetStringBytes("name")))
		}
		proxywasm.LogInfof("allowed domains: %v", allowedDomains)

	}
	return types.OnPluginStartStatusOK
}

func (ctx *httpAuthRandom) OnHttpRequestHeaders(int, bool) types.Action {
	hs, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		proxywasm.LogCriticalf("failed to get request headers: %v", err)
		return types.ActionContinue
	}
	var authority string
	for _, h := range hs {
		if h[0] == ":authority" {
			authority = h[1]
			break
		}
	}
	host := strings.Split(authority, ":")[0]
	if !contains(allowedDomains, host) {
		proxywasm.SendHttpResponse(403, [][2]string{{"wasm-reason", "domain not allowed"}}, nil, -1)
	}

	return types.ActionContinue
}

// Override types.DefaultPluginContext.
func (ctx *pluginContext) OnTick() {
	proxywasm.LogInfof("tick")
	hs := [][2]string{
		{":method", "GET"}, {":authority", ctx.apiHost}, {":path", "/domains"}, {"accept", "*/*"},
	}
	if _, err := proxywasm.DispatchHttpCall("controlplane", hs, nil, nil, 5000, ctx.callBack); err != nil {
		proxywasm.LogCriticalf("dispatch httpcall failed: %v", err)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
