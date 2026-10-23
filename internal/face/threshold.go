package face

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
)

const MaxThresholdProfileBytes = 16 << 10

var (
	thresholdProfileIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	thresholdSHA256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ThresholdProfile struct {
	SchemaVersion        int     `json:"schemaVersion"`
	ProfileID            string  `json:"profileId"`
	CoreSimilarity       float64 `json:"coreSimilarity"`
	EdgeSimilarity       float64 `json:"edgeSimilarity"`
	MinCoreSize          int     `json:"minCoreSize"`
	QualitySummarySHA256 string  `json:"qualitySummarySha256"`
}

func ParseThresholdProfile(data []byte) (ThresholdProfile, error) {
	if len(data) == 0 || len(data) > MaxThresholdProfileBytes {
		return ThresholdProfile{}, ErrInvalidInput
	}
	if err := rejectDuplicateThresholdJSONKeys(data); err != nil {
		return ThresholdProfile{}, ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile ThresholdProfile
	if err := decoder.Decode(&profile); err != nil {
		return ThresholdProfile{}, ErrInvalidInput
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ThresholdProfile{}, ErrInvalidInput
	}
	if profile.SchemaVersion != 1 || !thresholdProfileIDPattern.MatchString(profile.ProfileID) ||
		math.IsNaN(profile.CoreSimilarity) || math.IsInf(profile.CoreSimilarity, 0) ||
		math.IsNaN(profile.EdgeSimilarity) || math.IsInf(profile.EdgeSimilarity, 0) ||
		profile.CoreSimilarity <= 0 || profile.CoreSimilarity > 1 ||
		profile.EdgeSimilarity <= 0 || profile.EdgeSimilarity > profile.CoreSimilarity ||
		profile.MinCoreSize < 2 || profile.MinCoreSize > 100 ||
		!thresholdSHA256Pattern.MatchString(profile.QualitySummarySHA256) {
		return ThresholdProfile{}, ErrInvalidInput
	}
	return profile, nil
}

func rejectDuplicateThresholdJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("invalid JSON object key")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}
