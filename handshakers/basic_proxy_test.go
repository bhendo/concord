package handshakers

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicProxyAuthorizerHandshake(test *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if pa := request.Header.Get("Proxy-Authorization"); pa != "Basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk" {
			test.Errorf("incorrect Proxy-Authorization header. Got: %s", pa)
		}
		response.Write([]byte("authorized"))
	}))
	defer proxy.Close()
	h := &BasicProxyAuthorizer{
		Username: "testuser",
		Password: "testpassword",
	}
	req, _ := http.NewRequest("GET", "http://someserver", nil)
	conn, _ := net.Dial("tcp", proxy.Listener.Addr().String())
	res := &http.Response{
		StatusCode: http.StatusProxyAuthRequired,
		Header:     make(http.Header),
	}
	res.Header.Set("Proxy-Authenticate", `Basic realm="Access to some server"`)
	res, err := h.Handshake(res, req, conn)
	if err != nil {
		test.Errorf("failed with error: %s", err.Error())
	}
	if res.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", res.StatusCode)
	}
}
