// Package traefik_plugin_redirect_location is a traefik plugin fixing the location header in a redirect response
package traefik_plugin_redirect_location //nolint
import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const locationHeader string = "Location"

// Definition of a replacement.
type Rewrite struct {
	Regex       string `json:"regex,omitempty" toml:"regex,omitempty" yaml:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty" toml:"replacement,omitempty" yaml:"replacement,omitempty"`
}

// Config of the plugin.
type Config struct {
	Default  bool      `json:"default" toml:"default" yaml:"default"`
	Rewrites []Rewrite `json:"rewrites,omitempty" toml:"rewrites,omitempty" yaml:"rewrites,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// RedirectLocation a traefik plugin fixing the location header in a redirect response
type RedirectLocation struct {
	defaultHandling bool
	rewrites        []rewrite
	next            http.Handler
	name            string
}

type rewrite struct {
	regex       *regexp.Regexp
	replacement string
}

// create a RedirectLocation plugin instance
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rewrites := make([]rewrite, len(config.Rewrites))

	for i, rewriteConfig := range config.Rewrites {
		regexp, err := regexp.Compile(rewriteConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
		}
		rewrites[i] = rewrite{
			regex:       regexp,
			replacement: rewriteConfig.Replacement,
		}
	}

	return &RedirectLocation{
		defaultHandling: config.Default,
		rewrites:        rewrites,
		next:            next,
		name:            name,
	}, nil
}

func (r *RedirectLocation) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	myWriter := &responseWriter{
		defaultHandling: r.defaultHandling,
		rewrites:        r.rewrites,
		writer:          rw,
		request:         req,
	}

	r.next.ServeHTTP(myWriter, req)
}

type responseWriter struct {
	defaultHandling bool
	rewrites        []rewrite

	writer  http.ResponseWriter
	request *http.Request
}

func (r *responseWriter) Header() http.Header {
	return r.writer.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	return r.writer.Write(bytes)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	// only manipulate if redirect
	if statusCode > 300 && statusCode < 400 {

		// get header value
		// as we are handling a redirect there should be one and only one location header
		location := r.writer.Header().Get(locationHeader)

		// default handling
		if r.defaultHandling {

			locationUrl, err := url.Parse(location)
			if err != nil {
				http.Error(r.writer, err.Error(), http.StatusInternalServerError)
				return
			}

			host := r.request.Header.Get("X-Forwarded-Host")

			if locationUrl.Host == host || locationUrl.Host == "" {

				// path prefix
				prefix := r.request.Header.Get("X-Forwarded-Prefix")
				if strings.HasPrefix(strings.TrimPrefix(locationUrl.Path, "/"), prefix) {
					// it seems the service has handled the removed prefix correct so do nothing
				} else {
					oldPath := locationUrl.Path
					locationUrl.Path = path.Join(prefix, locationUrl.Path)
					// some logging
					fmt.Println("Changed location path from ", oldPath, "to", locationUrl.Path)
				}
			}

			location = locationUrl.String()
		}

		// rewrites
		for _, rewrite := range r.rewrites {
			locationOld := location
			location = rewrite.regex.ReplaceAllString(location, rewrite.replacement)
			// some logging
			fmt.Println("Changed location from ", locationOld, "to", location)
		}

		r.writer.Header().Set(locationHeader, location)

	}

	// call the wrapped writer
	r.writer.WriteHeader(statusCode)
}
