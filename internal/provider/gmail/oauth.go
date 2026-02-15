package gmail

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"
)

// No credentials are embedded in the binary. Users must supply their own
// Google Cloud OAuth credentials via one of:
//   - Config file (~/.config/termail/config.toml) under [gmail]
//   - Environment variables GMAIL_CLIENT_ID and GMAIL_CLIENT_SECRET

var oauthConfig = &oauth2.Config{
	Scopes: []string{
		gmailapi.GmailReadonlyScope,
		gmailapi.GmailSendScope,
		gmailapi.GmailModifyScope,
	},
	Endpoint: google.Endpoint,
}

// SetCredentials sets the OAuth client ID and secret.
func SetCredentials(clientID, clientSecret string) {
	oauthConfig.ClientID = clientID
	oauthConfig.ClientSecret = clientSecret
}

// HasCredentials reports whether OAuth credentials have been configured.
func HasCredentials() bool {
	return oauthConfig.ClientID != "" && oauthConfig.ClientSecret != ""
}

// EnsureCredentials returns nil if OAuth credentials have been configured via
// config file or environment variables. Otherwise it returns an error with setup
// instructions.
func EnsureCredentials() error {
	if HasCredentials() {
		return nil
	}
	return fmt.Errorf("gmail OAuth credentials not configured; set them in ~/.config/termail/config.toml under [gmail] or via GMAIL_CLIENT_ID / GMAIL_CLIENT_SECRET env vars")
}

func authenticate(ctx context.Context) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	oauthConfig.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.Query().Get("error"))
			fmt.Fprint(w, "Authentication failed. You can close this tab.")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authentication successful! You can close this tab.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Shutdown(ctx)

	url := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("\nOpen this URL in your browser to authorize termail:\n\n  %s\n\nWaiting for authorization...\n", url)

	select {
	case code := <-codeCh:
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange auth code: %w", err)
		}
		return token, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
