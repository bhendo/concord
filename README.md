# Concord

[![](https://img.shields.io/github/workflow/status/bhendo/concord/Test?longCache=tru&label=Test&logo=github%20actions&logoColor=fff)](https://github.com/bhendo/concord/actions?query=workflow%3ATest)
[![Go Reference](https://pkg.go.dev/badge/github.com/bhendo/concord.svg)](https://pkg.go.dev/github.com/bhendo/concord)
[![Go Report Card](https://goreportcard.com/badge/github.com/bhendo/concord)](https://goreportcard.com/report/github.com/bhendo/concord)

An HTTP(S) Roundtripper for use with Golang's `http.Client` with an interfaces that provides a mechanism for responding to proxies that require authentication.

## ProxyAuthorizer

A `ProxyAuthorizer` implements the `Handshaker` interface which represents a mechanism for handling responses from proxies that require authentication `response.StatusCode == 407`.

The handshake function receives a `*http.Response`, a `*http.Request`, and a `net.Conn` and returns a `*http.Response`. The response returned should be a response that is a result of successful or failed authentication. In the cases of successful authentication the response returned is often the desired response for the provided request.

The incoming `*http.Response` can be used to determine what kind of authentication is provided by the proxy server (e.g. Basic, Negotiate, or NTLM). **The body of this response (if there is one) must be closed before writing to the `net.Conn`.**

The `Handshaker` interface allows `concord.Transport` to handle simple (e.g. basic authentication) and complicated (e.g. NTLM or Kerberos) proxy authentication.

### BasicAuthProxy Handshaker

A sample handshaker is provided that adds a `Proxy-Authorization` request header for basic authentication

## Sample Usage

```go
package main

import (
    "net/http"
    "net/url"

    "github.com/bhendo/concord"
    "github.com/bhendo/concord/handshakers"
)

func main() {
    proxyURL, _ := url.Parse("http://some-basic-auth-proxy:8080")
    t := concord.Transport{
        Proxy: http.ProxyURL(proxyURL),
        ProxyAuthorizer: &handshakers.BasicAuthProxy{
            UserName: "username",
            Password: "password",
        }
    }
    c := http.Client{
        Transport: &t
    }
    c.Get("http://desired-website")
}
```

## TODO

- [ ] `DialContext` similar to `http.Transport`
- [ ] `DialTLS` similar to `http.Transport`
- [ ] Reuse connections
