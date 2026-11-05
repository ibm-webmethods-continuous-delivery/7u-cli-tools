package config

import (
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	if DefaultDataDir != "./ibmdocs-data" {
		t.Errorf("unexpected DefaultDataDir: %s", DefaultDataDir)
	}
	if DefaultLang != "en" {
		t.Errorf("unexpected DefaultLang: %s", DefaultLang)
	}
	if DefaultDepth != 3 {
		t.Errorf("unexpected DefaultDepth: %d", DefaultDepth)
	}
	if DefaultMaxTopics != 10 {
		t.Errorf("unexpected DefaultMaxTopics: %d", DefaultMaxTopics)
	}
	if DefaultDelay != 200*time.Millisecond {
		t.Errorf("unexpected DefaultDelay: %v", DefaultDelay)
	}
	if DefaultCDNBaseURL != "https://1.www.s81c.com" {
		t.Errorf("unexpected DefaultCDNBaseURL: %s", DefaultCDNBaseURL)
	}
	if DefaultRequestTimeout != 15*time.Second {
		t.Errorf("unexpected DefaultRequestTimeout: %v", DefaultRequestTimeout)
	}
	if MinContentChars != 350 {
		t.Errorf("unexpected MinContentChars: %d", MinContentChars)
	}
	if DefaultCacheTTL != 30*24*time.Hour {
		t.Errorf("unexpected DefaultCacheTTL: %v", DefaultCacheTTL)
	}

	if EnvCDNBaseURL != "IBMDOCS_CDN_BASE_URL" {
		t.Errorf("unexpected EnvCDNBaseURL: %s", EnvCDNBaseURL)
	}
	if EnvCacheTTL != "IBMDOCS_CACHE_TTL" {
		t.Errorf("unexpected EnvCacheTTL: %s", EnvCacheTTL)
	}
	if EnvProxy != "IBMDOCS_PROXY" {
		t.Errorf("unexpected EnvProxy: %s", EnvProxy)
	}
	if EnvTLSCABundle != "IBMDOCS_TLS_CA_BUNDLE" {
		t.Errorf("unexpected EnvTLSCABundle: %s", EnvTLSCABundle)
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DataDir:     DefaultDataDir,
		Lang:        DefaultLang,
		Verbose:     true,
		NoColor:     false,
		HTTPDebug:   true,
		CDNBaseURL:  DefaultCDNBaseURL,
		Delay:       DefaultDelay,
		CacheTTL:    DefaultCacheTTL,
		ProxyURL:    "http://proxy.corp:3128",
		TLSCABundle: "/etc/ssl/certs/custom.pem",
	}

	if cfg.DataDir != DefaultDataDir || cfg.Lang != DefaultLang || !cfg.Verbose || cfg.NoColor || !cfg.HTTPDebug {
		t.Errorf("unexpected config field values: %+v", cfg)
	}
	if cfg.ProxyURL != "http://proxy.corp:3128" || cfg.TLSCABundle != "/etc/ssl/certs/custom.pem" {
		t.Errorf("unexpected proxy/TLS fields: %+v", cfg)
	}
}
