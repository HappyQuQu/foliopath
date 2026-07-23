package app

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	allowedMediaRoot     = "/library"
	persistentDataRoot   = "/app/data"
	defaultListen        = "127.0.0.1:8080"
	listenEnvironment    = "FOLIOPATH_LISTEN"
	listenArgument       = "--listen"
	applicationEnvPrefix = "FOLIOPATH_"
)

var errInvalidConfiguration = errors.New("invalid application configuration")

type configuration struct {
	listenAddress string
	mediaRoot     string
	dataRoot      string
}

func loadConfiguration(input Input) (configuration, error) {
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

	listen, err = validateListenAddress(listen)
	if err != nil {
		return configuration{}, err
	}

	return configuration{
		listenAddress: listen,
		mediaRoot:     allowedMediaRoot,
		dataRoot:      persistentDataRoot,
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

func validateListenAddress(address string) (string, error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf(
			"%w: listen address must be a numeric loopback host and port",
			errInvalidConfiguration,
		)
	}

	hostIP := net.ParseIP(host)
	if hostIP == nil || !hostIP.IsLoopback() {
		return "", fmt.Errorf(
			"%w: listen host must be a numeric loopback address until authentication is ready",
			errInvalidConfiguration,
		)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != portText {
		return "", fmt.Errorf(
			"%w: listen port must be an integer from 1 through 65535",
			errInvalidConfiguration,
		)
	}

	return net.JoinHostPort(hostIP.String(), portText), nil
}
