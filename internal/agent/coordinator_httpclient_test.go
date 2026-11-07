package agent

import (
	"net/http"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/log"
	"github.com/stretchr/testify/require"
)

func TestProviderHTTPClient(t *testing.T) {
	t.Parallel()

	cfg, err := config.Init(t.Context(), t.TempDir(), "", false)
	require.NoError(t, err)
	coord := &coordinator{cfg: cfg}

	t.Run("neither enabled returns nil", func(t *testing.T) {
		cfg.Config().Options.Debug = false
		cfg.Config().Options.InsecureTLS = false
		require.Nil(t, coord.providerHTTPClient())
	})

	t.Run("insecure only skips TLS verification", func(t *testing.T) {
		cfg.Config().Options.Debug = false
		cfg.Config().Options.InsecureTLS = true
		client := coord.providerHTTPClient()
		require.NotNil(t, client)
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		require.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("debug only wraps default transport with logger", func(t *testing.T) {
		cfg.Config().Options.Debug = true
		cfg.Config().Options.InsecureTLS = false
		client := coord.providerHTTPClient()
		require.NotNil(t, client)
		logger, ok := client.Transport.(*log.HTTPRoundTripLogger)
		require.True(t, ok)
		require.Same(t, http.DefaultTransport, logger.Transport)
	})

	t.Run("both enabled wraps insecure transport with logger", func(t *testing.T) {
		cfg.Config().Options.Debug = true
		cfg.Config().Options.InsecureTLS = true
		client := coord.providerHTTPClient()
		require.NotNil(t, client)
		logger, ok := client.Transport.(*log.HTTPRoundTripLogger)
		require.True(t, ok)
		inner, ok := logger.Transport.(*http.Transport)
		require.True(t, ok)
		require.True(t, inner.TLSClientConfig.InsecureSkipVerify)
	})
}
