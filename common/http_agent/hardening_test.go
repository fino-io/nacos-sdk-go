package http_agent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type hardeningTransport func(*http.Request) (*http.Response, error)

func (f hardeningTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type hardeningBody struct {
	io.Reader
	closed bool
}

func (b *hardeningBody) Close() error { b.closed = true; return nil }

func TestHardeningRequestOnlyResult(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, status := range []int{200, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := &hardeningBody{Reader: strings.NewReader("result")}
			http.DefaultTransport = hardeningTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
			})
			got := (&HttpAgent{}).RequestOnlyResult(http.MethodGet, "http://example.test", nil, 1000, nil)
			if status == 200 {
				require.Equal(t, "result", got)
			} else {
				require.Empty(t, got)
			}
			require.True(t, body.closed)
		})
	}

	t.Run("unsupported_method", func(t *testing.T) {
		require.NotPanics(t, func() {
			require.Empty(t, (&HttpAgent{}).RequestOnlyResult("UNSUPPORTED", "http://example.test", nil, 1000, nil))
		})
	})
}

func TestHardeningHTTPEncoding(t *testing.T) {
	params := map[string]string{"special&key": "中文 &+=?#", "empty": "", "normal": "abc"}
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			requests := make(chan *http.Request, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				requests <- r
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			header := http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}, "X-Test": []string{"preserved"}}
			resp, err := (&HttpAgent{}).Request(method, server.URL+"/path?existing=keep", header, 1000, params)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			r := <-requests
			require.Equal(t, "keep", r.URL.Query().Get("existing"))
			require.Equal(t, "preserved", r.Header.Get("X-Test"))
			for key, value := range params {
				require.Contains(t, r.Form, key)
				require.Equal(t, value, r.Form.Get(key))
			}
		})
	}
}

func TestHardeningMalformedURL(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPut} {
		_, err := (&HttpAgent{}).Request(method, "://bad%url", nil, 1000, map[string]string{"k": "v"})
		require.Error(t, err)
	}
}

type hardeningErrorReader struct{}

func (hardeningErrorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestHardeningReadFailureClosesBody(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	body := &hardeningBody{Reader: hardeningErrorReader{}}
	http.DefaultTransport = hardeningTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
	})
	require.Empty(t, (&HttpAgent{}).RequestOnlyResult(http.MethodGet, "http://example.test", nil, 1000, nil))
	require.True(t, body.closed)
}

func TestHardeningHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}))
	defer server.Close()
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			_, err := (&HttpAgent{}).Request(method, server.URL, nil, 20, nil)
			require.Error(t, err)
		})
	}
}
