// Package copilot provides GitHub Copilot integration.
package copilot

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/charmbracelet/crush/internal/log"
)

var assistantRolePattern = regexp.MustCompile(`"role"\s*:\s*"assistant"`)

// NewClient creates a new HTTP client with a custom transport that adds the
// X-Initiator header based on message history in the request body.
func NewClient(isSubAgent, debug, insecure bool) *http.Client {
	return &http.Client{
		Transport: &initiatorTransport{debug: debug, insecure: insecure, isSubAgent: isSubAgent},
	}
}

type initiatorTransport struct {
	debug      bool
	insecure   bool
	isSubAgent bool
}

func (t *initiatorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	const (
		xInitiatorHeader = "X-Initiator"
		userInitiator    = "user"
		agentInitiator   = "agent"
	)

	if req == nil {
		return nil, fmt.Errorf("HTTP request is nil")
	}
	if req.Body == nil || req.Body == http.NoBody {
		// No body to inspect; default to user. A nil Body is valid for
		// bodyless requests (e.g. GET), and is distinct from http.NoBody,
		// so both must be handled before reading below.
		req.Header.Set(xInitiatorHeader, userInitiator)
		slog.Debug("Setting X-Initiator header to user (no request body)")
		return t.roundTrip(req)
	}

	// Clone request to avoid modifying the original.
	req = req.Clone(req.Context())

	// Read the original body into bytes so we can examine it.
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer req.Body.Close()

	// Restore the original body using the preserved bytes.
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Check for assistant messages using regex to handle whitespace
	// variations in the JSON while avoiding full unmarshalling overhead.
	initiator := userInitiator
	if assistantRolePattern.Match(bodyBytes) || t.isSubAgent {
		slog.Debug("Setting X-Initiator header to agent (found assistant messages in history)")
		initiator = agentInitiator
	} else {
		slog.Debug("Setting X-Initiator header to user (no assistant messages)")
	}
	req.Header.Set(xInitiatorHeader, initiator)

	return t.roundTrip(req)
}

func (t *initiatorTransport) roundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case t.debug:
		return log.NewHTTPClient().Transport.RoundTrip(req)
	case t.insecure:
		return insecureTransport.RoundTrip(req)
	default:
		return http.DefaultTransport.RoundTrip(req)
	}
}

// insecureTransport skips TLS certificate verification for providers that
// serve self-signed certificates.
var insecureTransport = &http.Transport{
	Proxy:           http.ProxyFromEnvironment,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
