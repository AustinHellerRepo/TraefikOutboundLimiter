// Package plugin_rewritebody a plugin to rewrite response body.
package traefikoutboundlimiter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"path"
)

// Config holds the plugin configuration.
type Config struct {
	LastModified 			  bool		  `json:"lastModified,omitempty"`
	ResetingIncrementerApiUrl string      `json:"resetingIncrementerApiUrl,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type limiter struct {
	name         				string
	next         				http.Handler
	lastModified				bool
	resetingIncrementerApiUrl	string
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	return &limiter{
		name:         			   name,
		next:         			   next,
		lastModified:			   config.LastModified,
		resetingIncrementerApiUrl: config.ResetingIncrementerApiUrl,
	}, nil
}

func (r *limiter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified:   r.lastModified,
		ResponseWriter: rw,
	}

	r.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.buffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")

	if contentEncoding != "" && contentEncoding != "identity" {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write body: %v", err)
		}

		return
	}

	// get the length of the bytes
	bodyBytesLength := len(bodyBytes)

	// send the length to the reseting incrementer API
	apiUrl := path.Join(r.resetingIncrementerApiUrl, "add")
	requestJsonString := fmt.Sprintf(`{"key": "traefik_outbound_limiter", "value": "%d"`, bodyBytesLength)
	requestJsonBytes := []byte(requestJsonString)
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestJsonBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	statusCode = resp.Status

	// react to a 409 error
	if statusCode == 409 {
		rw.WriteHeader(http.StatusConflict)
	} else {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
		}
	}
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool

	http.ResponseWriter
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if !r.lastModified {
		r.ResponseWriter.Header().Del("Last-Modified")
	}

	r.wroteHeader = true

	// Delegates the Content-Length Header creation to the final body write.
	r.ResponseWriter.Header().Del("Content-Length")

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.buffer.Write(p)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}