package model

type ExportMetadata struct {
	Version      int                         `json:"version"`
	Bundler      string                      `json:"bundler"`
	FileMetadata map[string]PlatformMetadata `json:"fileMetadata"`
}

type PlatformMetadata struct {
	Bundle string          `json:"bundle"`
	Assets []AssetMetadata `json:"assets"`
}

type AssetMetadata struct {
	Path string `json:"path"`
	Ext  string `json:"ext"`
}
