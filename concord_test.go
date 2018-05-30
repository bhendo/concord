package concord

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
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
	res, err := t.RoundTrip(req)
	if err != nil {
		test.Errorf("failed with error: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		test.Errorf("expected the response body to be readable but got an error: %s", err)
	}
	if string(data) != "hello" {
		test.Errorf("expected the response body to contain 'hello' but received '%s'", string(data))
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
	res, err := t.RoundTrip(req)
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		test.Errorf("expected the response body to be readable but got an error: %s", err)
	}
	if string(data) != "hello" {
		test.Errorf("expected the response body to contain 'hello' but received '%s'", string(data))
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
		response.Write([]byte("hello"))
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
	res, err := t.RoundTrip(req)
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		test.Errorf("expected the response body to be readable but got an error: %s", err)
	}
	if string(data) != "hello" {
		test.Errorf("expected the response body to contain 'hello' but received '%s'", string(data))
	}
}

func TestWithHTTPClient(test *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("hello"))
	}))
	defer server.Close()

	t := Transport{}
	c := http.Client{Transport: &t}
	res, err := c.Get(fmt.Sprintf("http://%s", server.Listener.Addr().String()))
	if err != nil {
		test.Fatalf("failed with error: %s", err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		test.Errorf("expected status code 200 but received: %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		test.Errorf("expected the response body to be readable but got an error: %s", err)
	}
	if string(data) != "hello" {
		test.Errorf("expected the response body to contain 'hello' but received '%s'", string(data))
	}
}

func TestConnBodyWrapper(test *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("hello"))
	}))
	defer server.Close()
	conn, _ := net.Dial("tcp", server.Listener.Addr().String())
	res, _ := http.DefaultClient.Get(server.URL)
	res, _ = wrapConnBody(conn, res)
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		test.Errorf("expected the response body to be readable but got an error: %s", err)
	}
	if string(data) != "hello" {
		test.Errorf("expected the response body to contain 'hello' but received '%s'", string(data))
	}
	if err := res.Body.Close(); err != nil {
		test.Errorf("closing the response body returned an error: %s", err)
	}
	if _, err := conn.Write([]byte("shouldfail")); err == nil {
		test.Errorf("expected writing to conn after closing the response body to fail, it did not.")
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
