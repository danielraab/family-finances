// Package oidcauth implements auth.OIDCClient for a single OpenID Connect
// provider, wrapping github.com/coreos/go-oidc/v3 (discovery + id_token
// verification against the provider JWKS) and golang.org/x/oauth2 (the
// authorization-code exchange with PKCE). Discovery runs once, at
// construction; package main builds one Client from config.OIDCConfig and
// injects it into the auth service, or passes nil when no issuer is set.
package oidcauth

import (
	"context"
	"errors"
	"fmt"

	"at.draab/familyfinances/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config is the provider configuration.
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	// RedirectURL is the absolute callback URL, i.e.
	// AUTH_BASE_URL + "/api/auth/oidc/callback".
	RedirectURL string
	Scopes      []string
}

// Client is the single-provider auth.OIDCClient implementation.
type Client struct {
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
	issuer   string
}

// New runs OIDC discovery against cfg.Issuer and returns a ready Client. The
// context bounds only the discovery request.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Issuer == "" {
		return nil, errors.New("oidcauth: issuer is empty")
	}
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: discovery for %s: %w", cfg.Issuer, err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	return &Client{
		oauth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		issuer:   cfg.Issuer,
	}, nil
}

// AuthCodeURL builds the provider authorization URL with the S256 PKCE
// challenge derived from verifier and the replay nonce.
func (c *Client) AuthCodeURL(state, nonce, verifier string) string {
	return c.oauth.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
}

// Exchange trades the authorization code for the raw id_token, presenting the
// PKCE verifier.
func (c *Client) Exchange(ctx context.Context, code, verifier string) (string, error) {
	tok, err := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", fmt.Errorf("oidcauth: code exchange: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return "", errors.New("oidcauth: token response carried no id_token")
	}
	return raw, nil
}

// VerifyIDToken verifies the id_token (signature via the provider JWKS, iss,
// aud, exp) and the nonce, then returns the identity claims.
func (c *Client) VerifyIDToken(ctx context.Context, rawIDToken, nonce string) (auth.OIDCClaims, error) {
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return auth.OIDCClaims{}, fmt.Errorf("oidcauth: id_token verification: %w", err)
	}
	if idToken.Nonce != nonce {
		return auth.OIDCClaims{}, errors.New("oidcauth: id_token nonce mismatch")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return auth.OIDCClaims{}, fmt.Errorf("oidcauth: reading claims: %w", err)
	}

	return auth.OIDCClaims{
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
}
