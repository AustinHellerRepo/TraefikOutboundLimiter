// Package TraefikOutboundLimiter, a plugin to restrict the outbound traffic if it goes over the limit of bytes.
package TraefikOutboundLimiter

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
	LastModified 			  bool		`json:"lastModified,omitempty"`
	ResetingIncrementerApiUrl string	`json:"resetingIncrementerApiUrl,omitempty"`
	ResetingIncrementerKey    string	`json:"resetingIncrementerKey,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

type limiter struct {
	name         				string			`json:"name,omitempty"`
	next         				http.Handler	`json:"handler,omitempty"`
	lastModified				bool			`json:"lastModified,omitempty"`
	resetingIncrementerApiUrl	string			`json:"resetingIncrementerApiUrl,omitempty"`
	resetingIncrementerKey      string			`json:"resetingIncrementerKey,omitempty"`
}

// New creates and returns a new rewrite body plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	
	return &limiter{
		name:         			   name,
		next:         			   next,
		lastModified:			   config.LastModified,
		resetingIncrementerApiUrl: config.ResetingIncrementerApiUrl,
		resetingIncrementerKey:    config.ResetingIncrementerKey,
	}, nil
}

func (r *limiter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	log.Printf("started ServeHTTP")
	log.Printf("Limiter: %v", r)

	wrappedWriter := &responseWriter{
		lastModified:   r.lastModified,
		ResponseWriter: rw,
	}

	log.Printf("wrapped writer")

	r.next.ServeHTTP(wrappedWriter, req)

	log.Printf("served HTTP request")

	bodyBytes := wrappedWriter.buffer.Bytes()

	log.Printf("localized buffer bytes")

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")

	log.Printf("determined content encoding")

	if contentEncoding != "" && contentEncoding != "identity" {

		log.Printf("content encoding is special case")

		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write body: %v", err)
		}

		return
	}

	// get the length of the bytes
	bodyBytesLength := len(bodyBytes)

	log.Printf("determined number of bytes in body")

	// send the length to the reseting incrementer API
	apiUrl := path.Join(r.resetingIncrementerApiUrl, "add")

	log.Printf("joined apiUrl path: %s", apiUrl)

	requestJsonString := fmt.Sprintf(`{"key": "%s", "value": "%d"`, r.resetingIncrementerKey, bodyBytesLength)

	log.Printf("formatted request json string: %s", requestJsonString)

	requestJsonBytes := []byte(requestJsonString)

	log.Printf("created empty byte array")

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestJsonBytes))

	if err != nil {
		log.Printf("Error creating new request: %v", err)
		panic(err)
	}

	log.Printf("created request for resetingIncrementerApi")

	req.Header.Set("Content-Type", "application/json")

	log.Printf("set content type header")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Received error while attempting to send request to resetingIncrementerApi: %v", err)
		panic(err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	log.Printf("Localized status code from response: %d", statusCode)

	// react to a 409 error
	if statusCode == 409 {
		log.Printf("Found status code 409")
		rw.WriteHeader(http.StatusConflict)
		log.Printf("Set header to StatusConflict")
	} else {
		log.Printf("Found other than 409")
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
		}
	}

	log.Printf("Method completed")
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