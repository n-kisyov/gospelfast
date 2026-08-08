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
	"regexp"
	"strings"
	"unicode/utf8"
)

var tagRe = regexp.MustCompile(`<[^>]*>`)

type ModuleMeta struct {
	ModName       string
	ModDrv        string
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

	modName := meta.ModName
	if modName == "" {
		modName = findModName(extractDir)
	}
	if modName == "" {
		return "", nil, fmt.Errorf("could not determine module name")
	}

	isGenbook := meta.ModDrv == "RawGenBook"

	if !isGenbook && meta.ModDrv != "" &&
		meta.ModDrv != "RawText" && meta.ModDrv != "RawText4" &&
		meta.ModDrv != "zText" && meta.ModDrv != "zText4" &&
		meta.ModDrv != "RawCom" && meta.ModDrv != "RawCom4" &&
		meta.ModDrv != "zCom" && meta.ModDrv != "zCom4" {
		return "", nil, fmt.Errorf("unsupported module type %q — only Bible texts, commentaries, and general books are supported", meta.ModDrv)
	}

	cmd := exec.Command("mod2imp", modName)
	cmd.Env = append(os.Environ(), "SWORD_PATH="+extractDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("mod2imp: %s: %w", string(out), err)
	}

	if isGenbook {
		impPath := filepath.Join(tmpDir, modName+".imp")
		if err := os.WriteFile(impPath, out, 0644); err != nil {
			return "", nil, err
		}
		return impPath, meta, nil
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

		text := tagRe.ReplaceAllString(trimmed, "")
		text = cleanUTF8(text)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		fmt.Fprintf(w, "%s %s\n", currentRef, text)
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
		if strings.HasPrefix(line, "ModDrv=") {
			meta.ModDrv = strings.TrimSpace(line[len("ModDrv="):])
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

func cleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, " ")
}

type GenbookEntry struct {
	Path    string
	Title   string
	Content string
}

func ParseGenbookIMP(data []byte) []GenbookEntry {
	var entries []GenbookEntry
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var currentEntry *GenbookEntry
	var contentBuf strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "$$$") {
			if currentEntry != nil {
				currentEntry.Content = cleanUTF8(strings.TrimSpace(contentBuf.String()))
				currentEntry.Content = tagRe.ReplaceAllString(currentEntry.Content, "")
				currentEntry.Content = strings.TrimSpace(currentEntry.Content)
				if currentEntry.Content != "" {
					entries = append(entries, *currentEntry)
				}
			}
			currentEntry = &GenbookEntry{Path: strings.TrimSpace(trimmed[3:])}
			contentBuf.Reset()
			continue
		}

		if currentEntry != nil {
			contentBuf.WriteString(line)
			contentBuf.WriteString("\n")
		}
	}

	if currentEntry != nil {
		currentEntry.Content = cleanUTF8(strings.TrimSpace(contentBuf.String()))
		currentEntry.Content = tagRe.ReplaceAllString(currentEntry.Content, "")
		currentEntry.Content = strings.TrimSpace(currentEntry.Content)
		if currentEntry.Content != "" {
			entries = append(entries, *currentEntry)
		}
	}

	for i := range entries {
		entries[i].Title = extractGenbookTitle(entries[i].Content)
		if entries[i].Title != "" {
			entries[i].Content = strings.TrimSpace(strings.TrimPrefix(entries[i].Content,
				strings.Join(strings.Fields(entries[i].Title), " ")))
		}
	}

	return entries
}

func extractGenbookTitle(content string) string {
	if idx := strings.Index(content, "<title>"); idx != -1 {
		end := strings.Index(content[idx:], "</title>")
		if end != -1 {
			title := content[idx+7 : idx+end]
			title = tagRe.ReplaceAllString(title, "")
			return strings.TrimSpace(title)
		}
	}
	return ""
}
