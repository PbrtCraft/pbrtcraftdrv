package mc

import (
	"errors"
	"os"
	"path"
	"runtime"
)

// ErrMinecraftClientNotFound returned when minecraft client folder not found
var ErrMinecraftClientNotFound = errors.New("Minecraft client folder not found")

// FindMinecraft return path of minecraft
func FindMinecraft() (string, error) {
	var mcDir string
	if runtime.GOOS == "windows" {
		mcDir = path.Join(os.Getenv("APPDATA"), ".minecraft")
	} else if runtime.GOOS == "linux" {
		mcDir = path.Join(os.Getenv("HOME"), ".minecraft")
	} else if runtime.GOOS == "darwin" {
		mcDir = path.Join(os.Getenv("HOME"), "Library", "Application Support", "minecraft")
	}

	if mcDir == "" {
		return "", ErrMinecraftClientNotFound
	}

	if _, err := os.Stat(mcDir); os.IsNotExist(err) {
		return "", ErrMinecraftClientNotFound
	}
	return mcDir, nil
}
