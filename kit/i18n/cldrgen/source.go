package cldrgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// source reads cldr-json files, fetching each once into the cache directory.
type source struct {
	base  string // …/cldr-json/<tag>/cldr-json
	cache string
}

var errNotFound = errors.New("not in cldr-json")

type obj = map[string]any

// raw returns the file at rel, fetching it into the cache on first use.
func (s *source) raw(rel string) ([]byte, error) {
	path := filepath.Join(s.cache, filepath.FromSlash(rel))
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if _, err := os.Stat(path + ".404"); err == nil {
			return nil, errNotFound
		}
		b, err = s.fetch(rel, path)
	}
	return b, err
}

// read returns the parsed JSON file at rel (e.g. "cldr-core/supplemental/plurals.json").
func (s *source) read(rel string) (obj, error) {
	b, err := s.raw(rel)
	if err != nil {
		return nil, err
	}
	var o obj
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return o, nil
}

func (s *source) fetch(rel, path string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Get(s.base + "/" + rel)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		os.WriteFile(path+".404", nil, 0o644)
		return nil, errNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", rel, res.Status)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return nil, err
	}
	return b, os.Rename(tmp, path)
}

func get(o any, path ...string) any {
	for _, p := range path {
		m, ok := o.(obj)
		if !ok {
			return nil
		}
		o = m[p]
	}
	return o
}

func str(o any, path ...string) string {
	s, _ := get(o, path...).(string)
	return s
}

func mapAt(o any, path ...string) obj {
	m, _ := get(o, path...).(obj)
	return m
}

// merge fills missing keys of child from parent, recursively.
func merge(child, parent obj) {
	for k, pv := range parent {
		cv, ok := child[k]
		if !ok {
			child[k] = pv
			continue
		}
		if cm, ok := cv.(obj); ok {
			if pm, ok := pv.(obj); ok {
				merge(cm, pm)
			}
		}
	}
}

// localeFile loads pkg/main/<id>/file.json merged over its parent chain and returns the locale object
// (the value under "main" → id).
func (g *gen) localeFile(pkg, file, id string) (obj, error) {
	var res obj
	for _, c := range g.chain(id) {
		if !g.sup.available[c] {
			continue
		}
		o, err := g.src.read(pkg + "/main/" + c + "/" + file)
		if errors.Is(err, errNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		inner := mapAt(o, "main", c)
		if inner == nil {
			return nil, fmt.Errorf("%s/main/%s/%s: no main.%s", pkg, c, file, c)
		}
		if res == nil {
			res = inner
		} else {
			merge(res, inner)
		}
	}
	if res == nil {
		return nil, fmt.Errorf("no %s/%s for %s", pkg, file, id)
	}
	return res, nil
}
