package connection

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/cli/cli/v2/internal/codespaces/api"
	"github.com/microsoft/dev-tunnels/go/tunnels"
)

const (
	clientName = "gh"
)

type TunnelClient struct {
	*tunnels.Client
	connected bool
	mu        sync.Mutex
}

type CodespaceConnection struct {
	tunnelProperties           api.TunnelProperties
	TunnelManager              *tunnels.Manager
	TunnelClient               *TunnelClient
	Options                    *tunnels.TunnelRequestOptions
	Tunnel                     *tunnels.Tunnel
	AllowedPortPrivacySettings []string
}

// NewCodespaceConnection initializes a connection to a codespace.
// This connections allows for port forwarding which enables the
// use of most features of the codespace command.
func NewCodespaceConnection(ctx context.Context, codespace *api.Codespace, httpClient *http.Client) (connection *CodespaceConnection, err error) {
	// Get the tunnel properties
	tunnelProperties := codespace.Connection.TunnelProperties

	// Create the tunnel manager
	tunnelManager, err := getTunnelManager(tunnelProperties, httpClient)
	if err != nil {
		return nil, fmt.Errorf("error getting tunnel management client: %w", err)
	}

	// Calculate allowed port privacy settings
	allowedPortPrivacySettings := codespace.RuntimeConstraints.AllowedPortPrivacySettings

	// Get the access tokens
	connectToken := tunnelProperties.ConnectAccessToken
	managementToken := tunnelProperties.ManagePortsAccessToken

	// Create the tunnel definition
	tunnel := &tunnels.Tunnel{
		TunnelID:  tunnelProperties.TunnelId,
		ClusterID: tunnelProperties.ClusterId,
		Domain:    tunnelProperties.Domain,
		AccessPolicies: &tunnels.TunnelAccessPolicies{
			Connect: &tunnels.TunnelAccessPolicy{
				AuthenticationRequired: true,
				Scopes: []string{
					string(tunnels.TunnelAccessScopeConnect),
				},
			},
			ManagePorts: &tunnels.TunnelAccessPolicy{
				AuthenticationRequired: true,
				Scopes: []string{
					string(tunnels.TunnelAccessScopeManagePorts),
				},
			},
		},
	}

	// Create options
	options := &tunnels.TunnelRequestOptions{
		IncludePorts: true,
	}

	// Create the tunnel client (not connected yet)
	tunnelClient, err := getTunnelClient(ctx, tunnelManager, tunnel, options, connectToken, managementToken)
	if err != nil {
		return nil, fmt.Errorf("error getting tunnel client: %w", err)
	}

	return &CodespaceConnection{
		tunnelProperties:           tunnelProperties,
		TunnelManager:              tunnelManager,
		TunnelClient:               tunnelClient,
		Options:                    options,
		Tunnel:                     tunnel,
		AllowedPortPrivacySettings: allowedPortPrivacySettings,
	}, nil
}

// Connect connects the client to the tunnel.
func (c *CodespaceConnection) Connect(ctx context.Context) error {
	// Lock the mutex to prevent race conditions with the underlying SSH connection
	c.TunnelClient.mu.Lock()
	defer c.TunnelClient.mu.Unlock()

	// If already connected, return
	if c.TunnelClient.connected {
		return nil
	}

	// Connect to the tunnel
	if err := c.TunnelClient.Client.Connect(ctx); err != nil {
		return fmt.Errorf("error connecting to tunnel: %w", err)
	}

	// Set the connected flag so we know we're connected
	c.TunnelClient.connected = true

	return nil
}

// Close closes the underlying tunnel client SSH connection.
func (c *CodespaceConnection) Close() error {
	// Lock the mutex to prevent race conditions with the underlying SSH connection
	c.TunnelClient.mu.Lock()
	defer c.TunnelClient.mu.Unlock()

	// Don't close if we're not connected
	if c.TunnelClient != nil && c.TunnelClient.connected {
		if err := c.TunnelClient.Close(); err != nil {
			return fmt.Errorf("failed to close tunnel client connection: %w", err)
		}

		c.TunnelClient.connected = false
	}

	return nil
}

// getTunnelManager creates a tunnel manager for the given codespace.
// The tunnel manager is used to get the tunnel hosted in the codespace that we
// want to connect to and perform operations on ports (add, remove, list, etc.).
func getTunnelManager(tunnelProperties api.TunnelProperties, httpClient *http.Client) (tunnelManager *tunnels.Manager, err error) {
	userAgent := []tunnels.UserAgent{{Name: clientName}}
	uri, err := url.Parse(tunnelProperties.ServiceUri)
	if err != nil {
		return nil, fmt.Errorf("error parsing tunnel service uri: %w", err)
	}
	options := tunnels.ManagerOptions{
		UserAgents: userAgent,
		HTTPClient: httpClient,
		TunnelServiceURI: uri,
	}

	// Create the tunnel manager
	tunnelManager, err = tunnels.NewManager(options)
	if err != nil {
		return nil, fmt.Errorf("error creating tunnel manager: %w", err)
	}

	return tunnelManager, nil
}

// getTunnelClient creates a tunnel client for the given tunnel.
// The tunnel client is used to connect to the tunnel and allows
// for ports to be forwarded locally.
func getTunnelClient(ctx context.Context, tunnelManager *tunnels.Manager, tunnel *tunnels.Tunnel, options *tunnels.TunnelRequestOptions, connectToken string, managePortsToken string) (tunnelClient *TunnelClient, err error) {
	// Get the tunnel that we want to connect to
	codespaceTunnel, err := tunnelManager.GetTunnel(ctx, tunnel, options)
	if err != nil {
		return nil, fmt.Errorf("error getting tunnel: %w", err)
	}

	// We need to pass false for accept local connections because we don't want to automatically connect to all forwarded ports
	clientOptions := tunnels.ClientOptions{
		Log:                       log.New(io.Discard, "", log.LstdFlags),
		Tunnel:                    codespaceTunnel,
		EnableAutomaticReconnection: true,
		AccessTokens: map[tunnels.TunnelAccessScope]string{
			tunnels.TunnelAccessScopeConnect:    connectToken,
			tunnels.TunnelAccessScopeManagePorts: managePortsToken,
		},
	}
	client, err := tunnels.NewClient(clientOptions)

	if err != nil {
		return nil, fmt.Errorf("error creating tunnel client: %w", err)
	}

	tunnelClient = &TunnelClient{
		Client:    client,
		connected: false,
	}

	return tunnelClient, nil
}