package aimodel

// ModelPreset describes a planned integration, never permission to install a package.
// Download manifests and their exact bytes remain owned by DownloadCatalog.
type ModelPreset struct {
	ID, Name, SourceLabel, Purpose, Readiness string
}

func BuiltinModelPresets() []ModelPreset {
	return []ModelPreset{
		{ID: "preset_siglip1", Name: "SigLIP Base Patch16 224", SourceLabel: "Google · Hugging Face", Purpose: "semantic_image_text", Readiness: "package_pending"},
		{ID: "preset_siglip2", Name: "SigLIP 2 Base Patch16 224", SourceLabel: "Google · Hugging Face", Purpose: "semantic_image_text", Readiness: "adapter_pending"},
		{ID: "preset_buffalo_l", Name: "buffalo_l", SourceLabel: "InsightFace · Immich", Purpose: "face_detection_embedding", Readiness: "adapter_pending"},
	}
}
