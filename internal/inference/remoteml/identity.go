package remoteml

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

var ErrIdentityMismatch = errors.New("ML service identity mismatch")

// ServiceIdentity is a trusted-deployment compatibility contract, not attestation.
// RuntimeContractSHA256 covers the pinned base and adapter, not the final OCI image.
type ServiceIdentity struct {
	Version               int    `json:"version"`
	Protocol              string `json:"protocol"`
	Model                 string `json:"model"`
	Dimension             int    `json:"dimension"`
	CacheManifestSHA256   string `json:"cache_manifest_sha256"`
	RuntimeContractSHA256 string `json:"runtime_contract_sha256"`
}

func (i ServiceIdentity) valid() bool {
	return i.Version == 1 && i.Protocol == "foliopath-ml-v1" && i.Model == ModelName && i.Dimension == Dimension &&
		identityDigest(i.CacheManifestSHA256) && identityDigest(i.RuntimeContractSHA256)
}

// Identity returns a copy of the independently configured expected identity.
func (c *Client) Identity() (ServiceIdentity, bool) {
	if c.identity == nil {
		return ServiceIdentity{}, false
	}
	return *c.identity, true
}

// VerifyIdentity rechecks the bound endpoint using the shared request budget.
func (c *Client) VerifyIdentity(ctx context.Context) error {
	if c.identity == nil {
		return ErrIdentityMismatch
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return ErrBusy
	}
	return c.verifyIdentity(ctx, *c.identity)
}

func identityDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// NewVerified requires an independently supplied deployment identity. It must not
// use a freshly fetched response as its own expected value (trust-on-first-use).
// Every subsequent inference also checks the response's immutable identity headers.
func NewVerified(ctx context.Context, origin string, expected ServiceIdentity) (*Client, error) {
	c, err := NewConfigured(origin, expected)
	if err != nil {
		return nil, err
	}
	if err := c.verifyIdentity(ctx, expected); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// NewConfigured binds deployment expectations without network I/O. Readiness is
// established by VerifyIdentity; every prediction verifies before sending input.
func NewConfigured(origin string, expected ServiceIdentity) (*Client, error) {
	if !expected.valid() {
		return nil, ErrIdentityMismatch
	}
	c, err := New(origin)
	if err != nil {
		return nil, err
	}
	c.identity = &expected
	return c, nil
}

func (c *Client) verifyIdentity(parent context.Context, expected ServiceIdentity) error {
	return c.verifyIdentityAt(parent, expected, "/api/v1/identity")
}

func (c *Client) verifyIdentityAt(parent context.Context, expected ServiceIdentity, path string) error {
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	endpoint := strings.TrimSuffix(c.endpoint, "/predict") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ErrInvalidInput
	}
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrUnavailable
	}
	var actual ServiceIdentity
	if len(data) > 4096 || json.Unmarshal(data, &actual) != nil || actual != expected {
		return ErrIdentityMismatch
	}
	return nil
}
