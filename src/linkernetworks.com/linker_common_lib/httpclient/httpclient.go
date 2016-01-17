package httpclient

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Header struct {
	Key   string
	Value string
}

func Http_get(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client := &http.Client{}

	var Body io.Reader

	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("GET", url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Http_post(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client := &http.Client{}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("POST", url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	// req.Header.Set("Content-Type", contenttype)
	resp, err = client.Do(req)
	return
}

func Http_put(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client := &http.Client{}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("PUT", url, Body)
	// req.Header.Set("Content-Type", contenttype)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Http_delete(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client := &http.Client{}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("DELETE", url, Body)
	// req.Header.Set("Content-Type", contenttype)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}
