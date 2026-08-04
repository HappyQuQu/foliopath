package app

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadConfigurationDefaultsToFixedRuntimeBoundaries(t *testing.T) {
	got, err := loadConfiguration(Input{})
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}

	want := configuration{
		listenAddress: "127.0.0.1:8080",
		mediaRoot:     "/library",
		dataRoot:      "/app/data",
	}
	if got != want {
		t.Fatalf("loadConfiguration() = %#v, want %#v", got, want)
	}
}

func TestLoadConfigurationAllowsNonLoopbackWithTrustedProxyCIDRs(t *testing.T) {
	got, err := loadConfiguration(Input{
		Environ: []string{
			"FOLIOPATH_LISTEN=0.0.0.0:8080",
			"FOLIOPATH_TRUSTED_PROXIES=192.0.2.19/24, 2001:db8::1/64",
		},
	})
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}
	if got.listenAddress != "0.0.0.0:8080" ||
		got.trustedProxies != "192.0.2.0/24,2001:db8::/64" ||
		!got.requireProxy {
		t.Fatalf("loadConfiguration() = %#v", got)
	}
}

func TestLoadConfigurationAllowsAuthenticatedLANHTTPWithoutProxy(t *testing.T) {
	got, err := loadConfiguration(Input{
		Environ: []string{"FOLIOPATH_LISTEN=0.0.0.0:8080"},
	})
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}
	if got.listenAddress != "0.0.0.0:8080" ||
		got.trustedProxies != "" ||
		got.requireProxy {
		t.Fatalf("loadConfiguration() = %#v", got)
	}
}

func TestLoadConfigurationRejectsInvalidTrustedProxyConfiguration(t *testing.T) {
	tests := [][]string{
		{"FOLIOPATH_TRUSTED_PROXIES="},
		{"FOLIOPATH_TRUSTED_PROXIES=192.0.2.10"},
		{"FOLIOPATH_TRUSTED_PROXIES=0.0.0.0/0"},
		{"FOLIOPATH_TRUSTED_PROXIES=::/0"},
		{"FOLIOPATH_TRUSTED_PROXIES=::ffff:192.0.2.0/120"},
		{"FOLIOPATH_TRUSTED_PROXIES=192.0.2.0/24,192.0.2.1/24"},
		{
			"FOLIOPATH_TRUSTED_PROXIES=192.0.2.0/24",
			"FOLIOPATH_TRUSTED_PROXIES=198.51.100.0/24",
		},
	}
	for _, environ := range tests {
		if _, err := loadConfiguration(Input{Environ: environ}); !errors.Is(
			err,
			errInvalidConfiguration,
		) {
			t.Fatalf("loadConfiguration(%q) error = %v", environ, err)
		}
	}
}

func TestLoadConfigurationAcceptsListenEnvironment(t *testing.T) {
	got, err := loadConfiguration(Input{
		Environ: []string{"TZ=Asia/Shanghai", "FOLIOPATH_LISTEN=[::1]:9090"},
	})
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}
	if got.listenAddress != "[::1]:9090" {
		t.Fatalf("listenAddress = %q, want [::1]:9090", got.listenAddress)
	}
}

func TestLoadConfigurationAcceptsListenArgumentForms(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "equals", args: []string{"--listen=127.0.0.2:9090"}},
		{name: "separate", args: []string{"--listen", "127.0.0.2:9090"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := loadConfiguration(Input{Args: test.args})
			if err != nil {
				t.Fatalf("loadConfiguration() error = %v", err)
			}
			if got.listenAddress != "127.0.0.2:9090" {
				t.Fatalf("listenAddress = %q, want 127.0.0.2:9090", got.listenAddress)
			}
		})
	}
}

func TestLoadConfigurationArgumentsOverrideEnvironment(t *testing.T) {
	got, err := loadConfiguration(Input{
		Args:    []string{"--listen=127.0.0.1:9090"},
		Environ: []string{"FOLIOPATH_LISTEN=127.0.0.1:8081"},
	})
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}
	if got.listenAddress != "127.0.0.1:9090" {
		t.Fatalf("listenAddress = %q, want argument value", got.listenAddress)
	}
}

func TestLoadConfigurationRejectsInvalidListenAddresses(t *testing.T) {
	addresses := []string{
		"",
		"localhost:8080",
		"127.0.0.1",
		"127.0.0.1:http",
		"127.0.0.1:0",
		"127.0.0.1:08080",
		"127.0.0.1:65536",
		"127.0.0.1:8080,127.0.0.1:8081",
	}

	for _, address := range addresses {
		t.Run(address, func(t *testing.T) {
			_, err := loadConfiguration(Input{
				Args: []string{"--listen=" + address},
			})
			if !errors.Is(err, errInvalidConfiguration) {
				t.Fatalf("loadConfiguration() error = %v, want errInvalidConfiguration", err)
			}
		})
	}
}

func TestLoadConfigurationRejectsInvalidArguments(t *testing.T) {
	tests := [][]string{
		{"--unknown=value"},
		{"--listen"},
		{"--listen=127.0.0.1:8080", "--listen=127.0.0.1:9090"},
	}

	for _, arguments := range tests {
		_, err := loadConfiguration(Input{Args: arguments})
		if !errors.Is(err, errInvalidConfiguration) {
			t.Fatalf("loadConfiguration(%q) error = %v, want errInvalidConfiguration", arguments, err)
		}
	}
}

func TestLoadConfigurationDoesNotEchoUnknownArgumentValue(t *testing.T) {
	_, err := loadConfiguration(Input{Args: []string{"--unknown=sensitive-value"}})
	if !errors.Is(err, errInvalidConfiguration) {
		t.Fatalf("loadConfiguration() error = %v, want errInvalidConfiguration", err)
	}
	if strings.Contains(err.Error(), "sensitive-value") {
		t.Fatalf("loadConfiguration() error leaked argument value: %q", err)
	}
}

func TestLoadConfigurationRejectsUnknownOrRepeatedApplicationEnvironment(t *testing.T) {
	tests := [][]string{
		{"FOLIOPATH_DATA_DIR=/tmp/data"},
		{"FOLIOPATH_LIBRARY_ROOT=/tmp/photos"},
		{"FOLIOPATH_UNKNOWN=value"},
		{"FOLIOPATH_LISTEN"},
		{
			"FOLIOPATH_LISTEN=127.0.0.1:8080",
			"FOLIOPATH_LISTEN=127.0.0.1:9090",
		},
	}

	for _, environ := range tests {
		_, err := loadConfiguration(Input{Environ: environ})
		if !errors.Is(err, errInvalidConfiguration) {
			t.Fatalf("loadConfiguration(%q) error = %v, want errInvalidConfiguration", environ, err)
		}
	}
}

func TestComposeOwnsValidatedConfiguration(t *testing.T) {
	application, err := compose(Input{
		Args: []string{"--listen=127.0.0.1:9090"},
	})
	if err != nil {
		t.Fatalf("compose() error = %v", err)
	}
	if application.configuration.listenAddress != "127.0.0.1:9090" {
		t.Fatalf(
			"application listen address = %q, want 127.0.0.1:9090",
			application.configuration.listenAddress,
		)
	}
	componentNames := make([]string, 0, len(application.components))
	for _, candidate := range application.components {
		componentNames = append(componentNames, candidate.name)
	}
	if got, want := strings.Join(componentNames, ","), "database,system-events,resource-limits,media-root,image-runtime,media-worker,scan-worker,automatic-discovery,library-removal-worker,http,readiness"; got != want {
		t.Fatalf("composed components = %q, want %q", got, want)
	}

	application, err = compose(Input{Args: []string{"--listen=0.0.0.0:9090"}})
	if err != nil {
		t.Fatalf("compose() LAN listen error = %v", err)
	}
	if application.configuration.listenAddress != "0.0.0.0:9090" ||
		application.configuration.requireProxy {
		t.Fatalf("LAN application configuration = %#v", application.configuration)
	}
}
