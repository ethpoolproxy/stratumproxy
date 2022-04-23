package webui

import (
	"embed"
	_ "embed"
)

//go:embed assets/*
var assets embed.FS

//go:embed template/*
var pageTemplate embed.FS
