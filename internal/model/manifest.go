package model

import "time"

type Manifest struct {
	ID             string            `json:"id"`
	CreatedAt      time.Time         `json:"createdAt"`
	RuntimeVersion string            `json:"runtimeVersion"`
	LaunchAsset    Asset             `json:"launchAsset"`
	Assets         []Asset           `json:"assets"`
	Metadata       map[string]string `json:"metadata"`
	Extra          map[string]any    `json:"extra"`
}

type Asset struct {
	Hash          string `json:"hash,omitempty"`
	Key           string `json:"key"`
	ContentType   string `json:"contentType"`
	FileExtension string `json:"fileExtension,omitempty"`
	URL           string `json:"url"`
}

type Directive struct {
	Type       string         `json:"type"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

type Extensions struct {
	AssetRequestHeaders map[string]map[string]string `json:"assetRequestHeaders"`
}

type ResolveParams struct {
	Project          string
	RuntimeVersion   string
	Platform         string
	ProtocolVersion  int
	CurrentUpdateID  string
	EmbeddedUpdateID string
}

type ManifestResult struct {
	Manifest  *Manifest
	Directive *Directive
}
