package yamlx

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gopkg.in/yaml.v3"
)

// LoadMap decodes YAML using several common encodings seen in the source repo.
func LoadMap(raw []byte) (map[string]any, error) {
	candidates := []string{
		string(raw),
		decodeGBK(raw),
		decodeLatin1(raw),
	}

	var lastErr error
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		parsed, err := parseYAML(candidate)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unable to decode yaml")
	}
	return nil, lastErr
}

func parseYAML(content string) (map[string]any, error) {
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(content), &parsed); err == nil {
		return parsed, nil
	}

	sanitizedTwo := bytes.ReplaceAll([]byte(content), []byte("\t"), []byte("  "))
	if err := yaml.Unmarshal(sanitizedTwo, &parsed); err == nil {
		return parsed, nil
	}

	sanitizedFour := bytes.ReplaceAll([]byte(content), []byte("\t"), []byte("    "))
	if err := yaml.Unmarshal(sanitizedFour, &parsed); err == nil {
		return parsed, nil
	}

	var final map[string]any
	err := yaml.Unmarshal(sanitizedFour, &final)
	return nil, err
}

func decodeGBK(raw []byte) string {
	if utf8.Valid(raw) {
		return ""
	}

	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func decodeLatin1(raw []byte) string {
	decoded, err := charmap.ISO8859_1.NewDecoder().Bytes(raw)
	if err != nil {
		return ""
	}
	return string(decoded)
}
