package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cliamp/internal/appdir"
)

// CustomPlaylist represents an extra user-defined playlist from YouTube or similar.
type CustomPlaylist struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Customisations holds the user-customised aspects of cliamp.
type Customisations struct {
	ProviderNames    map[string]string `json:"provider_names"`
	RadioReplacement string            `json:"radio_replacement"`
	ExtraPlaylists   []CustomPlaylist  `json:"extra_playlists"`
	HideLocal        bool              `json:"hide_local"`
}

// customisationsPath returns the path to the user_customisations.config file.
func customisationsPath() (string, error) {
	dir, err := appdir.Dir()
	if err != nil {
		return "", fmt.Errorf("getting app dir: %w", err)
	}
	return filepath.Join(dir, "user_customisations.config"), nil
}

// LoadCustomisations reads the customisations file. Returns an empty struct if not found.
func LoadCustomisations() (Customisations, error) {
	var c Customisations
	c.ProviderNames = make(map[string]string)

	path, err := customisationsPath()
	if err != nil {
		return c, fmt.Errorf("customisations: resolving config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c, nil
		}
		return c, fmt.Errorf("customisations: reading file: %w", err)
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("customisations: parsing file: %w", err)
	}
	if c.ProviderNames == nil {
		c.ProviderNames = make(map[string]string)
	}

	return c, nil
}

// SaveCustomisations writes the customisations struct to disk.
func SaveCustomisations(c Customisations) error {
	path, err := customisationsPath()
	if err != nil {
		return fmt.Errorf("customisations: resolving config path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("customisations: creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("customisations: marshalling JSON: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("customisations: writing file: %w", err)
	}
	return nil
}

// ClearCustomisations deletes the customisations file to return to defaults.
func ClearCustomisations() error {
	path, err := customisationsPath()
	if err != nil {
		return fmt.Errorf("customisations: resolving config path: %w", err)
	}
	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("customisations: removing file: %w", err)
	}
	return nil
}
