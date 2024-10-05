package oauth

import (
	"context"
	"golang.org/x/oauth2"
)

var oauthConfig = oauth2.Config{
	ClientID:     "your-client-id",
	ClientSecret: "your-client-secret",
	RedirectURL:  "your-redirect-url",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://provider.com/oauth/authorize",
		TokenURL: "https://provider.com/oauth/token",
	},
}

func ValidateOAuthToken(accessToken string) (*oauth2.Token, error) {
	tokenSource := oauthConfig.TokenSource(context.Background(), &oauth2.Token{
		AccessToken: accessToken,
	})
	return tokenSource.Token()
}

