package hcaptcha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cdk-airdrop-station/server/internal/model"
)

type Verifier struct {
	Secret    string
	VerifyURL string
	SiteKey   string
	Client    *http.Client
}

type verifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

func New(secret, verifyURL, siteKey string) Verifier {
	if strings.TrimSpace(verifyURL) == "" {
		verifyURL = "https://api.hcaptcha.com/siteverify"
	}
	return Verifier{
		Secret:    secret,
		VerifyURL: verifyURL,
		SiteKey:   siteKey,
		Client:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (v Verifier) VerifyHCaptcha(ctx context.Context, token string, remoteIP string, expectedSitekey string) error {
	if strings.TrimSpace(token) == "" {
		return model.ErrCaptchaRequired
	}
	if strings.TrimSpace(v.Secret) == "" {
		return model.ErrHCaptchaNotConfigured
	}
	siteKey := strings.TrimSpace(expectedSitekey)
	if siteKey == "" {
		siteKey = strings.TrimSpace(v.SiteKey)
	}
	form := url.Values{}
	form.Set("secret", v.Secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	if siteKey != "" {
		form.Set("sitekey", siteKey)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.VerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return model.ErrCaptchaServiceUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return model.ErrCaptchaServiceUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.ErrCaptchaServiceUnavailable
	}
	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return model.ErrCaptchaServiceUnavailable
	}
	if !result.Success {
		return model.ErrCaptchaInvalid
	}
	return nil
}
