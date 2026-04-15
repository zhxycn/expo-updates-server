package model

type AssetIndex struct {
	Path          string `json:"path"`
	Hash          string `json:"hash"`
	Key           string `json:"key"`
	ContentType   string `json:"contentType"`
	FileExtension string `json:"fileExtension,omitempty"`
}

type PlatformIndex struct {
	Bundle AssetIndex   `json:"bundle"`
	Assets []AssetIndex `json:"assets"`
}
