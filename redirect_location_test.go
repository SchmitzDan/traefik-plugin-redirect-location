// Package traefik_plugin_redirect_location is a traefik plugin fixing the location header in a redirect response
package traefik_plugin_redirect_location //nolint
import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRewrites(t *testing.T) {
	tests := []struct {
		desc           string
		rewrites       []Rewrite
		locationBefore string
		expLocation    string
	}{
		{
			desc: "should replace foo by bar in location header",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			locationBefore: "foo",
			expLocation:    "bar",
		},
		{
			desc: "should replace foo by bar in location header",
			rewrites: []Rewrite{
				{
					Regex:       "foo",
					Replacement: "bar",
				},
			},
			locationBefore: "prefix/foo/path",
			expLocation:    "prefix/bar/path",
		},
		{
			desc: "should replace http by https in location header",
			rewrites: []Rewrite{
				{
					Regex:       "^http://(.+)$",
					Replacement: "https://$1",
				},
			},
			locationBefore: "http://test:1000",
			expLocation:    "https://test:1000",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				Default:  false,
				Rewrites: test.rewrites,
			}

			next := func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Add("Location", test.locationBefore)
				rw.WriteHeader(301)
			}

			redirectLocation, err := New(context.Background(), http.HandlerFunc(next), config, "redirectLocation")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodGet, "/", nil)

			redirectLocation.ServeHTTP(recorder, req)

			location := recorder.Header().Get("Location")

			if test.expLocation != location {
				t.Errorf("Unexpected redirect Location: expected %+v, result: %+v", test.expLocation, location)
			}
		})
	}
}

func TestDefaultHandling(t *testing.T) {
	tests := []struct {
		desc            string
		forwardedPrefix string
		forwardedHost   string
		locationBefore  string
		expLocation     string
	}{
		{
			desc:           "No forwarded Prefix and relative path",
			locationBefore: "somevalue",
			expLocation:    "somevalue",
		},
		{
			desc:           "No forwarded Prefix and absolute path",
			locationBefore: "http://host:815/path",
			expLocation:    "http://host:815/path",
		},
		{
			desc:            "Forwarded Prefix and relative path",
			forwardedPrefix: "/test",
			locationBefore:  "somevalue",
			expLocation:     "/test/somevalue",
		},
		{
			desc:            "Forwarded Prefix and relative path already containing prefix",
			forwardedPrefix: "/test",
			locationBefore:  "/test/somevalue",
			expLocation:     "/test/somevalue",
		},
		{
			desc:            "Forwarded Prefix and absolute path",
			forwardedPrefix: "/test",
			forwardedHost:   "host",
			locationBefore:  "http://host:815/path",
			expLocation:     "http://host:815/test/path",
		},
		{
			desc:            "Forwarded Prefix and absolute path already containing prefix",
			forwardedPrefix: "/test",
			forwardedHost:   "host",
			locationBefore:  "http://host:815/test/path",
			expLocation:     "http://host:815/test/path",
		},
	}

	config := &Config{
		Default:  true,
		Rewrites: make([]Rewrite, 0),
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			next := func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Add("Location", test.locationBefore)
				rw.WriteHeader(301)
			}

			redirectLocation, err := New(context.Background(), http.HandlerFunc(next), config, "redirectLocation")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if len(test.forwardedPrefix) > 0 {
				req.Header.Add("X-Forwarded-Prefix", test.forwardedPrefix)
			}
			if len(test.forwardedHost) > 0 {
				req.Header.Add("X-Forwarded-Host", test.forwardedHost)
			}

			redirectLocation.ServeHTTP(recorder, req)

			location := recorder.Header().Get("Location")

			if test.expLocation != location {
				t.Errorf("Unexpected redirect Location: expected %+v, result: %+v", test.expLocation, location)
			}
		})
	}
}
