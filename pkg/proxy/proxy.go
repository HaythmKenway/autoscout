package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/elazarl/goproxy"
)

// InterceptedRequest represents the data we want to send to the AI fleet
type InterceptedRequest struct {
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	Headers         map[string][]string `json:"headers"`
	Body            []byte              `json:"body"`
	ResponseCode    int                 `json:"response_code"`
	ResponseBody    []byte              `json:"response_body"`
	ResponseHeaders map[string][]string `json:"response_headers"`
}

var (
	// RequestQueue is the channel where we push intercepted traffic
	RequestQueue = make(chan InterceptedRequest, 1000)
	mu           sync.Mutex
	running      bool
)

func StartProxy(addr string, upstreamProxyURL string) error {
	mu.Lock()
	if running {
		mu.Unlock()
		return nil
	}
	running = true
	mu.Unlock()

	// 1. Load or Create CA
	caCert, caKey, err := LoadOrCreateCA()
	if err != nil {
		return err
	}

	// 2. Setup GoProxy with MITM
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// Handle Upstream Proxy (Burp Suite)
	if upstreamProxyURL != "" {
		u, err := url.Parse(upstreamProxyURL)
		if err == nil {
			proxy.Tr.Proxy = http.ProxyURL(u)
			localUtils.Logger("Using Upstream Proxy: "+upstreamProxyURL, 1)
		} else {
			localUtils.Logger("Invalid Upstream Proxy URL: "+upstreamProxyURL, 2)
		}
	}

	// Handle HTTPS CONNECT requests
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Custom CA for MITM
	goproxy.GoproxyCa = tls.Certificate{
		Certificate: [][]byte{caCert.Raw},
		PrivateKey:  caKey,
		Leaf:        caCert,
	}

	// 3. Intercept and Capture
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			return resp
		}

		// Read Request Body (if available)
		var reqBody []byte
		if ctx.Req.Body != nil {
			reqBody, _ = io.ReadAll(ctx.Req.Body)
			ctx.Req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// Read Response Body
		resBody, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(resBody))

		// Push to AI Queue
		intercepted := InterceptedRequest{
			Method:          ctx.Req.Method,
			URL:             ctx.Req.URL.String(),
			Headers:         ctx.Req.Header,
			Body:            reqBody,
			ResponseCode:    resp.StatusCode,
			ResponseBody:    resBody,
			ResponseHeaders: resp.Header,
		}

		select {
		case RequestQueue <- intercepted:
		default:
			// Queue full
		}

		return resp
	})

	localUtils.Logger("Starting MITM Proxy Server on "+addr, 1)
	crtPath, _ := GetCAPaths()
	localUtils.Logger("CA Certificate saved to: "+crtPath, 1)
	localUtils.Logger("IMPORT THIS CRT INTO YOUR BROWSER TO SEE HTTPS TRAFFIC", 1)

	return http.ListenAndServe(addr, proxy)
}
