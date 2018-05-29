package concord

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bhendo/concord/handshakers"
)

func TestRoundTrip(test *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("hello"))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s", server.Listener.Addr().String()), nil)
	t := Transport{}
	resp, err := t.RoundTrip(req)
	if err != nil {
		test.Errorf("failed with error: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", resp.StatusCode)
	}
}

func TestHTTPRoundTripWithDummyProxy(test *testing.T) {
	serverURL, _ := url.Parse("http://someserver/")

	proxy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.String() != serverURL.String() {
			test.Errorf("expected %s to be %s", request.URL.String(), serverURL.String())
		}
		response.Write([]byte("hello"))
	}))
	defer proxy.Close()

	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s", proxy.Listener.Addr()))

	req, _ := http.NewRequest("GET", serverURL.String(), nil)
	t := Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	resp, err := t.RoundTrip(req)
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", resp.StatusCode)
	}
}

func TestHTTPRoundTripWithHandshaker(test *testing.T) {
	serverURL, _ := url.Parse("http://someserver/")

	proxy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.String() != serverURL.String() {
			test.Errorf("expected %s to be %s", request.URL.String(), serverURL.String())
		}
		pa := request.Header.Get("Proxy-Authorization")
		switch pa {
		case "":
			response.Header().Set("Proxy-Authenticate", `Basic realm="Access to some server"`)
			response.WriteHeader(http.StatusProxyAuthRequired)
			response.Write([]byte("Authorization required"))
			return
		case "Basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk":
			break
		default:
			test.Errorf("incorrect Proxy-Authorization header. Got: %s", pa)
		}
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			test.Errorf("failed with error: %s", err.Error())
		}
		if string(data) != "somedata" {
			test.Errorf("expected the body to contain 'somedata' but got: %s", string(data))
		}
		response.Write([]byte("authentication successful"))
	}))
	defer proxy.Close()

	proxyURL, _ := url.Parse(fmt.Sprintf("http://%s", proxy.Listener.Addr()))

	req, _ := http.NewRequest("POST", serverURL.String(), bytes.NewBufferString("somedata"))
	t := Transport{
		Proxy: http.ProxyURL(proxyURL),
		ProxyAuthorizer: &handshakers.BasicProxyAuthorizer{
			Username: "testuser",
			Password: "testpassword",
		},
	}
	resp, err := t.RoundTrip(req)
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", resp.StatusCode)
	}
}

func TestWithHTTPClient(test *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("hello"))
	}))
	defer server.Close()

	t := Transport{}
	c := http.Client{Transport: &t}
	resp, err := c.Get(fmt.Sprintf("http://%s", server.Listener.Addr().String()))
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	if resp.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", resp.StatusCode)
	}
}

func TestCanonicalAddress(test *testing.T) {
	url1, _ := url.Parse("http://127.0.0.1")
	url2, _ := url.Parse("http://127.0.0.1:8080")
	url3, _ := url.Parse("https://127.0.0.1")
	url4, _ := url.Parse("https://127.0.0.1:8443")
	testCases := []struct {
		URL      *url.URL
		Expected string
	}{
		{
			url1,
			"127.0.0.1:80",
		},
		{
			url2,
			"127.0.0.1:8080",
		},
		{
			url3,
			"127.0.0.1:443",
		},
		{
			url4,
			"127.0.0.1:8443",
		},
	}
	for _, testCase := range testCases {
		if addr := canonicalAddress(testCase.URL); addr != testCase.Expected {
			test.Errorf("expected '%s' but got '%s'", testCase.Expected, addr)
		}
	}
}
