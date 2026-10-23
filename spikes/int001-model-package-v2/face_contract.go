package modelpackagev2

import (
	"bytes"
	"encoding/json"
	"path"
	"regexp"
	"strings"
)

const (
	FaceFormatVersion = 3

	FaceDecodeContract       = "libvips-srgb-longedge1600-v1"
	FaceDetectorContract     = "yunet-bgr640-letterbox-score-sqrt-v1"
	FacePostprocessContract  = "yunet-stride8-16-32-opencv-int-nms-v1"
	FaceAlignmentContract    = "arcface-5point-similarity112-bilinear-zero-v1"
	FaceEmbeddingContract    = "auraface-rgb-minus127.5-div127.5-512-v1"
	FaceStorageContract      = "face-512-l2-f16le-v1"
	FaceThresholdContract    = "face-threshold-profile-json-v1"
	faceThresholdProfileRole = "face_threshold_profile"
)

var (
	facePackageID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)
	faceVersion   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)
	spdxLicenseID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+-]{0,127}$`)
)

type FaceContracts struct {
	Decode           string `json:"decode"`
	Detector         string `json:"detector"`
	Postprocess      string `json:"postprocess"`
	Alignment        string `json:"alignment"`
	Embedding        string `json:"embedding"`
	Storage          string `json:"storage"`
	ThresholdProfile string `json:"thresholdProfile"`
}

type FaceFile struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	Role      string `json:"role"`
	LicenseID string `json:"licenseId"`
}

type FaceManifest struct {
	FormatVersion int           `json:"formatVersion"`
	PackageID     string        `json:"packageId"`
	Purpose       string        `json:"purpose"`
	Version       string        `json:"version"`
	Architecture  string        `json:"architecture"`
	Contracts     FaceContracts `json:"contracts"`
	Files         []FaceFile    `json:"files"`
}

func ParseFaceV3(data []byte) (FaceManifest, error) {
	if len(data) == 0 || len(data) > maxManifestBytes || duplicateKey(data) {
		return FaceManifest{}, errInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result FaceManifest
	if decoder.Decode(&result) != nil || !jsonEOF(decoder) || !validFaceManifest(result) {
		return FaceManifest{}, errInvalid
	}
	return result, nil
}

func validFaceManifest(value FaceManifest) bool {
	if value.FormatVersion != FaceFormatVersion || !facePackageID.MatchString(value.PackageID) ||
		value.Purpose != "face_detection_embedding" || !faceVersion.MatchString(value.Version) ||
		value.Architecture != "portable-onnx" || value.Contracts.Decode != FaceDecodeContract ||
		value.Contracts.Detector != FaceDetectorContract || value.Contracts.Postprocess != FacePostprocessContract ||
		value.Contracts.Alignment != FaceAlignmentContract || value.Contracts.Embedding != FaceEmbeddingContract ||
		value.Contracts.Storage != FaceStorageContract || value.Contracts.ThresholdProfile != FaceThresholdContract ||
		len(value.Files) != 3 {
		return false
	}
	required := map[string]bool{"face_detector": false, "face_embedder": false, faceThresholdProfileRole: false}
	names := map[string]bool{}
	var total int64
	for _, file := range value.Files {
		if file.Name == "" || len(file.Name) > 255 || path.Base(file.Name) != file.Name ||
			strings.ContainsAny(file.Name, "\\\x00") || file.Name == "." || file.Name == ".." ||
			file.Size <= 0 || file.Size > maxPackageBytes || !hexSHA256.MatchString(file.SHA256) ||
			!spdxLicenseID.MatchString(file.LicenseID) || names[file.Name] || !requiredRole(required, file.Role) ||
			required[file.Role] || total > maxPackageBytes-file.Size {
			return false
		}
		names[file.Name] = true
		required[file.Role] = true
		total += file.Size
	}
	return required["face_detector"] && required["face_embedder"] && required[faceThresholdProfileRole]
}
