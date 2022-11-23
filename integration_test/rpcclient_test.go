package integrationtest

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	CodeOK = 0

	ErrStatusUnauthorized = "rest: unauthorized"
	ErrUnknown            = "rest: unknown error"

	HeaderContentType     = "Content-Type"
	HeaderContentTypeJson = "application/json"

	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"

	REST_TIMEOUT = 2 // in sec
)

type Method string

func RESTCallWithJson(nodeDomain string, endpoint string, method Method, param []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(time.Second * REST_TIMEOUT),
	}

	for i := 0; i < 5; i++ {
		url := strings.Join([]string{nodeDomain, endpoint}, "/")

		req, err := http.NewRequest(string(method), url, bytes.NewReader(param))
		if err != nil {
			return []byte{}, err
		}

		req.Header.Set(HeaderContentType, HeaderContentTypeJson)

		res, err := restCall(client, req)
		if os.IsTimeout(err) {
			continue
		} else if err != nil {
			return []byte{}, err
		}

		return res, nil
	}

	return nil, errors.New("exceeded trial")
}

func restCall(client *http.Client, req *http.Request) ([]byte, error) {
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	err = restErrorHandler(res)
	if err != nil {
		return []byte{}, err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			logger.Printf("res.body.close, %s", err)
		}
	}()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	return resBody, nil
}

func restErrorHandler(res *http.Response) error {
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return errors.New(ErrStatusUnauthorized)
	default:
		return errors.New(ErrUnknown)
	}
}
