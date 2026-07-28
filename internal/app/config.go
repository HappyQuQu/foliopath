package app

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"net/netip"
)

const (
	allowedMediaRoot     = "/library"
	persistentDataRoot   = "/app/data"
	defaultListen        = "127.0.0.1:8080"
	listenEnvironment    = "FOLIOPATH_LISTEN"
	proxiesEnvironment   = "FOLIOPATH_TRUSTED_PROXIES"
	listenArgument       = "--listen"
	applicationEnvPrefix = "FOLIOPATH_"
)

var errInvalidConfiguration = errors.New("invalid application configuration")

type configuration struct {
	listenAddress  string
	mediaRoot      string
	dataRoot       string
	trustedProxies string
	requireProxy   bool
}

func loadConfiguration(input Input) (configuration, error) {
	trustedProxies, proxiesSet, err := trustedProxiesFromEnvironment(input.Environ)
	if err != nil {
		return configuration{}, err
	}
	environmentListen, environmentSet, err := listenFromEnvironment(input.Environ)
	if err != nil {
		return configuration{}, err
	}

	argumentListen, argumentSet, err := listenFromArguments(input.Args)
	if err != nil {
		return configuration{}, err
	}

	listen := defaultListen
	if environmentSet {
		listen = environmentListen
	}
	if argumentSet {
		listen = argumentListen
	}

	listen, requireProxy, err := validateListenAddress(listen, proxiesSet)
	if err != nil {
		return configuration{}, err
	}

	return configuration{
		listenAddress:  listen,
		mediaRoot:      allowedMediaRoot,
		dataRoot:       persistentDataRoot,
		trustedProxies: trustedProxies,
		requireProxy:   requireProxy,
	}, nil
}

func listenFromEnvironment(environ []string) (string, bool, error) {
	var (
		listen string
		found  bool
	)
	for _, entry := range environ {
		name, value, hasValue := strings.Cut(entry, "=")
		if !strings.HasPrefix(name, applicationEnvPrefix) {
			continue
		}
		if !hasValue {
			return "", false, fmt.Errorf(
				"%w: environment variable %q has no value",
				errInvalidConfiguration,
				name,
			)
		}
		if name == proxiesEnvironment {
			continue
		}
		if name != listenEnvironment {
			return "", false, fmt.Errorf(
				"%w: unknown environment variable %q",
				errInvalidConfiguration,
				name,
			)
		}
		if found {
			return "", false, fmt.Errorf(
				"%w: environment variable %q is repeated",
				errInvalidConfiguration,
				name,
			)
		}
		listen = value
		found = true
	}
	return listen, found, nil
}

func trustedProxiesFromEnvironment(environ []string) (string, bool, error) {
	var (
		value string
		found bool
	)
	for _, entry := range environ {
		name, candidate, hasValue := strings.Cut(entry, "=")
		if name != proxiesEnvironment {
			continue
		}
		if !hasValue || found {
			return "", false, fmt.Errorf(
				"%w: environment variable %q is invalid or repeated",
				errInvalidConfiguration,
				name,
			)
		}
		found = true
		value = candidate
	}
	if !found {
		return "", false, nil
	}

	parts := strings.Split(value, ",")
	canonical := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		text := strings.TrimSpace(part)
		prefix, err := netip.ParsePrefix(text)
		if err != nil {
			return "", false, fmt.Errorf(
				"%w: trusted proxies must be comma-separated IP CIDRs",
				errInvalidConfiguration,
			)
		}
		prefix = prefix.Masked()
		if prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return "", false, fmt.Errorf(
				"%w: trusted proxy CIDRs cannot be universal or IPv4-mapped",
				errInvalidConfiguration,
			)
		}
		normalized := prefix.String()
		if _, duplicate := seen[normalized]; duplicate {
			return "", false, fmt.Errorf(
				"%w: trusted proxy CIDRs must be unique",
				errInvalidConfiguration,
			)
		}
		seen[normalized] = struct{}{}
		canonical = append(canonical, normalized)
	}
	if len(canonical) == 0 {
		return "", false, fmt.Errorf(
			"%w: at least one trusted proxy CIDR is required",
			errInvalidConfiguration,
		)
	}
	return strings.Join(canonical, ","), true, nil
}

func listenFromArguments(arguments []string) (string, bool, error) {
	var (
		listen string
		found  bool
	)
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		var value string

		switch {
		case argument == listenArgument:
			if index+1 >= len(arguments) {
				return "", false, fmt.Errorf(
					"%w: %s requires a value",
					errInvalidConfiguration,
					listenArgument,
				)
			}
			index++
			value = arguments[index]
		case strings.HasPrefix(argument, listenArgument+"="):
			value = strings.TrimPrefix(argument, listenArgument+"=")
		default:
			name, _, _ := strings.Cut(argument, "=")
			return "", false, fmt.Errorf(
				"%w: unknown argument %q",
				errInvalidConfiguration,
				name,
			)
		}

		if found {
			return "", false, fmt.Errorf(
				"%w: %s is repeated",
				errInvalidConfiguration,
				listenArgument,
			)
		}
		listen = value
		found = true
	}
	return listen, found, nil
}

func validateListenAddress(address string, trustedProxiesSet bool) (string, bool, error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return "", false, fmt.Errorf(
			"%w: listen address must be a numeric loopback host and port",
			errInvalidConfiguration,
		)
	}

	hostIP := net.ParseIP(host)
	if hostIP == nil {
		return "", false, fmt.Errorf(
			"%w: listen host must be a numeric IP address",
			errInvalidConfiguration,
		)
	}
	requireProxy := !hostIP.IsLoopback()
	if requireProxy && !trustedProxiesSet {
		return "", false, fmt.Errorf(
			"%w: non-loopback listen requires trusted proxy CIDRs",
			errInvalidConfiguration,
		)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != portText {
		return "", false, fmt.Errorf(
			"%w: listen port must be an integer from 1 through 65535",
			errInvalidConfiguration,
		)
	}

	return net.JoinHostPort(hostIP.String(), portText), requireProxy, nil
}

func parseTrustedProxyPrefixes(value string) []netip.Prefix {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			return nil
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}
