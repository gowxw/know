package know

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const version = "3.0.935"

func DoRequest(urlString, urlMethod, body, host string,
	timeout time.Duration, w io.Writer,
	debug bool, extraheader map[string]string) (int64, int, error) {

	fmt.Printf("dorequest url: %s %v\n", urlString, reflect.TypeOf(w))

	if w == nil {
		return 0, 0, errors.New("nil writer")
	}

	if timeout == 0 {
		return 0, 0, errors.New("timeout can not be 0")
	}

	url, err := url.Parse(urlString)
	if err != nil {
		fmt.Printf("could not parse url")
		return 0, 0, err
	}
	if url.Scheme == "" {
		url.Scheme = "http"
	}

	if urlMethod == "" {
		fmt.Printf("no http method")
		return 0, 0, errors.New("no http method")
	}

	if urlMethod == "POST" || urlMethod == "post" {
		urlMethod = "POST"
	}

	if urlMethod == "GET" || urlMethod == "get" {
		urlMethod = "GET"
	}

	req, err := http.NewRequest(urlMethod, url.String(), strings.NewReader(body))
	if err != nil {
		return 0, 0, err
	}

	for k, v := range extraheader {
		req.Header.Add(k, v)
	}

	if host != "" {
		req.Host = host
	}

	if debug == true {
		trace := &httptrace.ClientTrace{
			DNSStart: func(dnsstart httptrace.DNSStartInfo) { fmt.Printf("%v,dns start\n", time.Now()) },
			DNSDone: func(dnsdone httptrace.DNSDoneInfo) {
				fmt.Printf("%v, dns done\n", time.Now())
				if dnsdone.Err != nil {
					fmt.Printf("dns done with err %s\n", dnsdone.Err.Error())
				} else {
					fmt.Printf("dns done with %v\n", dnsdone.Addrs)
				}
			},
			ConnectStart: func(network, addr string) { fmt.Printf("conn start %s %s %v\n", network, addr, time.Now()) },
			ConnectDone: func(net, addr string, err error) {
				if err != nil {
					fmt.Printf("unable to connect to host %v: %v\n", addr, err)
				}
				fmt.Printf("%s conn done \n", time.Now())

			},
			GotConn: func(info httptrace.GotConnInfo) {
				fmt.Printf("%s gotconn \n", time.Now())
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				if info.Err == nil {
					fmt.Printf("write a request ok\n")
				} else {
					fmt.Printf("write a request err\n")
				}
			},
			GotFirstResponseByte: func() { fmt.Printf("%s get response \n", time.Now()) },
		}

		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
	}

	if url.Scheme == "https" {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := http.Client{Transport: tr, Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("client do error : %s\n", err.Error())
		return 0, 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("bad response")
		return 0, resp.StatusCode, errors.New("bad response")
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return n, http.StatusOK, err
	}

	return n, http.StatusOK, nil

}

