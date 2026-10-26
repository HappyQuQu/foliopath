package face

import "testing"

func TestReviewIdentityBindsKeyAndRequestWithoutPlaintext(t *testing.T) {
	type request struct{ Action, FaceID string }
	id, hash, err := ReviewIdentity("face-review-key-001", request{Action: "assign_face", FaceID: "face_001"})
	if err != nil {
		t.Fatal(err)
	}
	idAgain, hashAgain, err := ReviewIdentity("face-review-key-001", request{Action: "assign_face", FaceID: "face_001"})
	if err != nil || idAgain != id || hashAgain != hash {
		t.Fatalf("id=%q/%q hash=%q/%q err=%v", id, idAgain, hash, hashAgain, err)
	}
	_, different, err := ReviewIdentity("face-review-key-001", request{Action: "exclude_face", FaceID: "face_001"})
	if err != nil || different == hash {
		t.Fatalf("different=%q err=%v", different, err)
	}
	if id == "face-review-key-001" || hash == "face-review-key-001" || !ValidReviewDigest(hash) {
		t.Fatalf("unsafe identity id=%q hash=%q", id, hash)
	}
}
