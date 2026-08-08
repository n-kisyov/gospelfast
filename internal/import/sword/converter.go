package sword

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ModuleMeta struct {
	ModName       string
	Versification string
	Lang          string
	Description   string
}

func ConvertRawzip(sourceURL, tmpDir string) (string, *ModuleMeta, error) {
	zipPath := filepath.Join(tmpDir, "module.zip")
	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		if err := downloadFile(sourceURL, zipPath); err != nil {
			return "", nil, fmt.Errorf("download: %w", err)
		}
	} else {
		zipPath = sourceURL
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", nil, err
	}

	if err := extractZip(zipPath, extractDir); err != nil {
		return "", nil, fmt.Errorf("extract: %w", err)
	}

	meta, err := parseConf(extractDir)
	if err != nil {
		return "", nil, fmt.Errorf("parse conf: %w", err)
	}

	modName := findModName(extractDir)
	if modName == "" && meta != nil {
		modName = meta.ModName
	}
	if modName == "" {
		return "", nil, fmt.Errorf("could not determine module name")
	}

	cmd := exec.Command("mod2imp", modName)
	cmd.Env = append(os.Environ(), "SWORD_PATH="+extractDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("mod2imp: %s: %w", string(out), err)
	}

	vplPath := filepath.Join(tmpDir, modName+".vpl")
	if err := impToVPL(out, vplPath); err != nil {
		return "", nil, fmt.Errorf("convert imp to vpl: %w", err)
	}

	return vplPath, meta, nil
}

func impToVPL(data []byte, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var currentRef string
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "$$$") {
			currentRef = strings.TrimSpace(trimmed[3:])
			continue
		}

		if currentRef == "" || trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "$$$") {
			continue
		}

		fmt.Fprintf(w, "%s %s\n", currentRef, trimmed)
	}

	w.Flush()
	return scanner.Err()
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func parseConf(extractDir string) (*ModuleMeta, error) {
	confDir := filepath.Join(extractDir, "mods.d")
	entries, err := os.ReadDir(confDir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".conf" {
			confPath := filepath.Join(confDir, e.Name())
			return parseConfFile(confPath)
		}
	}
	return nil, fmt.Errorf("no .conf file found in mods.d/")
}

func parseConfFile(path string) (*ModuleMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	meta := &ModuleMeta{}
	lines := strings.Split(string(data), "\n")
	var modName string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			modName = line[1 : len(line)-1]
			meta.ModName = modName
		}
		if strings.HasPrefix(line, "Versification=") {
			meta.Versification = strings.TrimSpace(line[len("Versification="):])
		}
		if strings.HasPrefix(line, "Lang=") {
			meta.Lang = strings.TrimSpace(line[len("Lang="):])
		}
		if strings.HasPrefix(line, "Description=") {
			meta.Description = strings.TrimSpace(line[len("Description="):])
		}
	}
	if meta.Versification == "" {
		meta.Versification = "KJV"
	}
	if meta.Lang == "" {
		meta.Lang = "en"
	}
	return meta, nil
}

func findModName(extractDir string) string {
	modsDir := filepath.Join(extractDir, "modules")
	entries, _ := os.ReadDir(modsDir)
	for _, cat := range entries {
		if cat.IsDir() {
			catDir := filepath.Join(modsDir, cat.Name())
			subEntries, _ := os.ReadDir(catDir)
			for _, sub := range subEntries {
				if sub.IsDir() {
					subDir := filepath.Join(catDir, sub.Name())
					modEntries, _ := os.ReadDir(subDir)
					for _, m := range modEntries {
						if m.IsDir() {
							return m.Name()
						}
					}
				}
			}
		}
	}
	return ""
}
