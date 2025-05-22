package config

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/obot-platform/nanobot/pkg/types"
	"sigs.k8s.io/yaml"
)

func Load(path string, profiles ...string) (*types.Config, string, error) {
	var (
		last types.Config
		data []byte
		dir  = "."
	)

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", err
		}
	} else {
		s, err := os.Stat(path)
		if err != nil {
			return nil, "", fmt.Errorf("error reading %s: %w", path, err)
		}

		if s.IsDir() {
			path = filepath.Join(path, "nanobot.yaml")
		}

		dir = filepath.Dir(path)
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
	}

	if err := yaml.Unmarshal(data, &last); err != nil {
		return nil, "", err
	}

	if last.Extends != "" {
		parent, _, err := Load(filepath.Join(dir, last.Extends), profiles...)
		if err != nil {
			return nil, "", fmt.Errorf("error reading %s: %w", last.Extends, err)
		}
		last, err = merge(*parent, last)
		if err != nil {
			return nil, "", fmt.Errorf("error merging %s: %w", last.Extends, err)
		}
	}

	for _, profile := range profiles {
		profile, _, optional := strings.Cut(profile, "?")
		profileConfig, found := last.Profiles[profile]
		if !found && !optional {
			return nil, "", fmt.Errorf("profile %s not found", profile)
		} else if !found {
			continue
		}
		var err error
		last, err = merge(last, profileConfig)
		if err != nil {
			return nil, "", fmt.Errorf("error merging profile %s: %w", profile, err)
		}
	}

	seen := map[string]string{}
	if err := checkDup(seen, "mcpServer", slices.Collect(maps.Keys(last.MCPServers))); err != nil {
		return nil, "", err
	}
	if err := checkDup(seen, "agent", slices.Collect(maps.Keys(last.Agents))); err != nil {
		return nil, "", err
	}
	if len(last.Agents) > 1 && last.Publish.Entrypoint == "" {
		keys := slices.Sorted(maps.Keys(last.Agents))
		return nil, "", fmt.Errorf("multiple agents defined, but no entrypoint specified, " +
			"please specify one in nanobot.yaml, for example:\n" +
			"\n" +
			"  publish:\n" +
			"    entrypoint: " + keys[0] + "\n" +
			"  agents:\n" +
			"    " + keys[0] + ": ...\n" +
			"    " + keys[1] + ": ...\n")

	}

	return &last, dir, nil
}

func toMap(cfg types.Config) (map[string]any, error) {
	result := map[string]any{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return result, json.Unmarshal(data, &result)
}

func mergeObject(base, overlay any) any {
	if baseMap, ok := base.(map[string]any); ok {
		if overlayMap, ok := overlay.(map[string]any); ok {
			newMap := maps.Clone(baseMap)
			for k, v := range overlayMap {
				newMap[k] = mergeObject(baseMap[k], v)
			}
			return newMap
		}
	}
	return overlay
}

func merge(base, overlay types.Config) (types.Config, error) {
	baseMap, err := toMap(base)
	if err != nil {
		return types.Config{}, err
	}
	overlayMap, err := toMap(overlay)
	if err != nil {
		return types.Config{}, err
	}

	merged := mergeObject(baseMap, overlayMap)
	mergedData, err := json.Marshal(merged)
	if err != nil {
		return types.Config{}, err
	}

	var result types.Config
	return result, json.Unmarshal(mergedData, &result)
}

func checkDup(seen map[string]string, category string, keys []string) error {
	for _, k := range keys {
		if oldCategory, ok := seen[k]; ok {
			return fmt.Errorf("duplicate name [%s] used for both [%s] and [%s]", k, oldCategory, category)
		}
		seen[k] = category
	}
	return nil
}
