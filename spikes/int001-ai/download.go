package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	errDownloadOrigin    = errors.New("artifact URL escaped the trusted origin")
	errDownloadQuota     = errors.New("artifact exceeds the temporary download quota")
	errDownloadResume    = errors.New("server rejected the trusted resume validator")
	errDownloadIntegrity = errors.New("downloaded artifact failed integrity verification")
)

// trustedArtifact is populated by an already-reviewed catalog in this spike.
// It is deliberately not an API DTO: callers must not supply URLs or hashes.
type trustedArtifact struct {
	URL    string
	Origin string
	ETag   string
	Size   int64
	SHA256 string
	// allowLoopbackForTest exists only so httptest can exercise transport
	// behavior. Reviewed catalog entries must leave it false.
	allowLoopbackForTest bool
}

// fetchTrustedArtifact downloads into a same-directory partial file, verifies
// the pinned size and digest, then publishes without replacing an existing
// verified file. A partial file is resumable only with the catalog's ETag.
func fetchTrustedArtifact(
	ctx context.Context,
	client *catalogHTTPClient,
	artifact trustedArtifact,
	parent, partialName, verifiedName string,
	quota int64,
) error {
	if client == nil || client.client == nil || quota <= 0 || artifact.Size <= 0 || artifact.Size > quota {
		return errDownloadQuota
	}
	if !packageSegmentPattern.MatchString(partialName) || !packageSegmentPattern.MatchString(verifiedName) {
		return errors.New("download names must be safe path segments")
	}
	if err := validateArtifactOrigin(artifact.URL, artifact.Origin, artifact.allowLoopbackForTest); err != nil {
		return err
	}
	if artifact.ETag == "" || len(artifact.SHA256) != sha256.Size*2 {
		return errors.New("trusted catalog entry is incomplete")
	}

	partialPath := filepath.Join(parent, partialName)
	verifiedPath := filepath.Join(parent, verifiedName)
	if _, err := os.Lstat(verifiedPath); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	partialSize, err := regularFileSize(partialPath)
	if err != nil {
		return err
	}
	if partialSize > artifact.Size || partialSize > quota {
		return errDownloadQuota
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return err
	}
	if partialSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", partialSize))
		req.Header.Set("If-Range", artifact.ETag)
	}
	boundedClient := *client.client
	previousRedirect := boundedClient.CheckRedirect
	boundedClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many artifact redirects")
		}
		if err := validateArtifactOrigin(req.URL.String(), artifact.Origin, artifact.allowLoopbackForTest); err != nil {
			return err
		}
		if previousRedirect != nil {
			return previousRedirect(req, via)
		}
		return nil
	}
	response, err := boundedClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.Header.Get("ETag") != artifact.ETag {
		if partialSize > 0 {
			_ = os.Remove(partialPath)
		}
		return errDownloadResume
	}
	if partialSize == 0 {
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("initial artifact response: %s", response.Status)
		}
	} else {
		if response.StatusCode != http.StatusPartialContent ||
			!strings.HasPrefix(response.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-", partialSize)) {
			_ = os.Remove(partialPath)
			return errDownloadResume
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if partialSize > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	partial, err := os.OpenFile(partialPath, flags, 0o600)
	if err != nil {
		return err
	}
	remaining := artifact.Size - partialSize
	written, copyErr := io.Copy(partial, io.LimitReader(response.Body, remaining+1))
	syncErr := partial.Sync()
	closeErr := partial.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != remaining {
		if written > remaining {
			return errDownloadQuota
		}
		return io.ErrUnexpectedEOF
	}
	digest, err := fileDigest(partialPath)
	if err != nil {
		return err
	}
	if digest != strings.ToLower(artifact.SHA256) {
		_ = os.Remove(partialPath)
		return errDownloadIntegrity
	}

	parentDirectory, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer parentDirectory.Close()
	if err := atomicRenameNoReplace(int(parentDirectory.Fd()), partialName, verifiedName); err != nil {
		return fmt.Errorf("publish verified artifact: %w", err)
	}
	return parentDirectory.Sync()
}

func validateArtifactOrigin(rawURL, trustedOrigin string, allowLoopbackForTest bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errDownloadOrigin
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if origin != trustedOrigin {
		return errDownloadOrigin
	}
	hostname := parsed.Hostname()
	address := net.ParseIP(hostname)
	loopbackTarget := strings.EqualFold(hostname, "localhost") || address != nil && (address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast())
	if parsed.Scheme != "https" || loopbackTarget {
		if !allowLoopbackForTest || parsed.Scheme != "http" || address == nil || !address.IsLoopback() {
			return errDownloadOrigin
		}
	}
	if parsed.Scheme != "https" && !allowLoopbackForTest {
		return errDownloadOrigin
	}
	return nil
}

func regularFileSize(path string) (int64, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("partial download is not a regular file")
	}
	return info.Size(), nil
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func contentRange(start, end, total int64) string {
	return "bytes " + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10) + "/" + strconv.FormatInt(total, 10)
}
