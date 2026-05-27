package app

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type layoutLoadResult struct {
	LayoutJSON    string
	RenderOptions map[string]interface{}
	SourceKind    string
	LayoutMember  string
}

func layoutJSONFromPathOrDefault(layoutPath string, cfg Config, dataCtx DataContext) (*layoutLoadResult, error) {
	if strings.TrimSpace(layoutPath) == "" {
		layoutJSON, renderOptions, err := layoutJSONFromObjectOrDefault(nil, cfg, dataCtx)
		if err != nil {
			return nil, err
		}
		return &layoutLoadResult{LayoutJSON: layoutJSON, RenderOptions: renderOptions, SourceKind: "default"}, nil
	}

	if strings.EqualFold(filepath.Ext(layoutPath), ".zip") {
		return layoutJSONFromZipBundle(layoutPath, cfg, dataCtx)
	}

	obj, err := readLayoutObjectFromFile(layoutPath)
	if err != nil {
		return nil, err
	}
	layoutJSON, renderOptions, err := layoutJSONFromObjectOrDefault(obj, cfg, dataCtx)
	if err != nil {
		return nil, err
	}
	return &layoutLoadResult{LayoutJSON: layoutJSON, RenderOptions: renderOptions, SourceKind: "file", LayoutMember: filepath.Base(layoutPath)}, nil
}

func readLayoutObjectFromFile(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read layout file %s: %w", filePath, err)
	}
	obj, err := parseLayoutObject(data, filePath)
	if err != nil {
		return nil, fmt.Errorf("parse layout file %s: %w", filePath, err)
	}
	return obj, nil
}

func parseLayoutObject(data []byte, name string) (map[string]interface{}, error) {
	var obj map[string]interface{}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json":
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
	default:
		if err := yaml.Unmarshal(data, &obj); err != nil {
			return nil, err
		}
	}
	if obj == nil {
		return nil, fmt.Errorf("layout document is empty or not an object")
	}
	return obj, nil
}

func layoutJSONFromZipBundle(zipPath string, cfg Config, dataCtx DataContext) (*layoutLoadResult, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open layout zip %s: %w", zipPath, err)
	}
	defer func() { _ = zr.Close() }()

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		name := cleanZipMemberName(f.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		files[name] = f
	}

	layoutName, err := findLayoutMember(files)
	if err != nil {
		return nil, fmt.Errorf("find layout in %s: %w", zipPath, err)
	}
	layoutData, err := readZipMember(files[layoutName])
	if err != nil {
		return nil, fmt.Errorf("read layout member %s: %w", layoutName, err)
	}
	obj, err := parseLayoutObject(layoutData, layoutName)
	if err != nil {
		return nil, fmt.Errorf("parse layout member %s: %w", layoutName, err)
	}
	if err := inlineZipImageAssets(obj, files, path.Dir(layoutName)); err != nil {
		return nil, fmt.Errorf("inline zip image assets: %w", err)
	}
	layoutJSON, renderOptions, err := layoutJSONFromObjectOrDefault(obj, cfg, dataCtx)
	if err != nil {
		return nil, err
	}
	return &layoutLoadResult{LayoutJSON: layoutJSON, RenderOptions: renderOptions, SourceKind: "zip", LayoutMember: layoutName}, nil
}

func findLayoutMember(files map[string]*zip.File) (string, error) {
	preferred := []string{
		"layout.yaml",
		"layout.yml",
		"layout.json",
		"almanach.yaml",
		"almanach.yml",
		"almanach.json",
	}
	for _, name := range preferred {
		if _, ok := files[name]; ok {
			return name, nil
		}
	}

	var candidates []string
	for name := range files {
		if path.Dir(name) == "." && isLayoutExt(name) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("zip contains no root-level layout.yaml, layout.yml, layout.json, almanach.yaml, almanach.yml, or almanach.json")
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("zip contains multiple root-level layout candidates %v; use layout.yaml or layout.json", candidates)
	}
}

func isLayoutExt(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func inlineZipImageAssets(obj map[string]interface{}, files map[string]*zip.File, layoutDir string) error {
	layoutObj := any(obj)
	if wrapped, ok := obj["layout"]; ok {
		layoutObj = wrapped
	}
	layoutMap, ok := layoutObj.(map[string]interface{})
	if !ok {
		return nil
	}
	blocks, ok := layoutMap["blocks"].([]interface{})
	if !ok {
		return nil
	}

	for i, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]interface{})
		if !ok || block["type"] != "image" {
			continue
		}
		data, ok := block["data"].(map[string]interface{})
		if !ok {
			continue
		}
		src, ok := data["src"].(string)
		if !ok || !shouldInlineBundleSrc(src) {
			continue
		}
		assetName, err := resolveZipAsset(layoutDir, src)
		if err != nil {
			return fmt.Errorf("block %d image src %q: %w", i, src, err)
		}
		zf, ok := files[assetName]
		if !ok {
			return fmt.Errorf("block %d image src %q resolved to missing zip member %q", i, src, assetName)
		}
		assetData, err := readZipMember(zf)
		if err != nil {
			return fmt.Errorf("block %d image src %q: %w", i, src, err)
		}
		data["src"] = dataURLForAsset(assetName, assetData)
	}
	return nil
}

func shouldInlineBundleSrc(src string) bool {
	s := strings.TrimSpace(src)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return false
	}
	return true
}

func resolveZipAsset(layoutDir, src string) (string, error) {
	s := strings.ReplaceAll(strings.TrimSpace(src), "\\", "/")
	s = strings.TrimPrefix(s, "./")
	base := layoutDir
	if base == "." || base == "/" {
		base = ""
	}
	cleaned := cleanZipMemberName(path.Join(base, s))
	if cleaned == "" || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("invalid zip-relative path")
	}
	return cleaned, nil
}

func cleanZipMemberName(name string) string {
	s := strings.ReplaceAll(name, "\\", "/")
	s = strings.TrimPrefix(s, "./")
	s = path.Clean(s)
	if s == "." || s == ".." || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "/") {
		return ""
	}
	return s
}

func readZipMember(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func dataURLForAsset(name string, data []byte) string {
	mimeType := mime.TypeByExtension(strings.ToLower(path.Ext(name)))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}
