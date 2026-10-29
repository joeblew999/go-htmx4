// Command jsbundle bundles a browser module and the npm packages it imports into one ES module, with no Node (#156).
//
// npm's registry is read over HTTP, every tarball is checked against the integrity npm publishes, and esbuild (its Go
// API) does the bundling. What was resolved is kept in a lock file next to the entry, so a bundle is rebuilt from the
// same packages until the lock is updated on purpose:
//
//	go run ./internal/jsbundle -entry web/moq/moq.mjs -out web/moq/moq.js -lock web/moq/npm-lock.json [-update @moq/watch@0.6.6 @moq/publish@0.5.3]
//
// Without -update it installs exactly what the lock names. With -update it resolves the named packages and their
// dependencies afresh (the highest published version each range allows) and rewrites the lock.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/evanw/esbuild/pkg/api"
)

// Lock is what one -update resolved: every package of the bundle, flat (one version of each name).
type Lock struct {
	Roots    []string           `json:"roots"`
	Packages map[string]Package `json:"packages"`
}

// Package is one resolved npm package.
type Package struct {
	Version   string `json:"version"`
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
}

const registry = "https://registry.npmjs.org/"

func main() {
	entry := flag.String("entry", "", "the module to bundle")
	out := flag.String("out", "", "the bundle to write")
	lockPath := flag.String("lock", "", "the lock file")
	update := flag.Bool("update", false, "resolve the packages named as arguments (name@version) and rewrite the lock")
	modules := flag.String("modules", "dist/npm", "where packages are unpacked (git-ignored)")
	flag.Parse()
	if *entry == "" || *out == "" || *lockPath == "" {
		fail(errors.New("-entry, -out and -lock are required"))
	}

	var lock Lock
	if *update {
		if flag.NArg() == 0 {
			fail(errors.New("-update needs the packages to resolve, e.g. @moq/watch@0.6.6"))
		}
		l, err := resolve(flag.Args())
		if err != nil {
			fail(err)
		}
		lock = l
		data, _ := json.MarshalIndent(lock, "", "  ")
		if err := os.WriteFile(*lockPath, append(data, '\n'), 0o644); err != nil {
			fail(err)
		}
	} else {
		data, err := os.ReadFile(*lockPath)
		if err != nil {
			fail(err)
		}
		if err := json.Unmarshal(data, &lock); err != nil {
			fail(fmt.Errorf("%s: %w", *lockPath, err))
		}
	}

	root := filepath.Join(*modules, "node_modules")
	if err := os.RemoveAll(*modules); err != nil {
		fail(err)
	}
	for _, name := range sortedNames(lock.Packages) {
		if err := install(root, name, lock.Packages[name]); err != nil {
			fail(err)
		}
	}

	abs, _ := filepath.Abs(root)
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{*entry},
		Outfile:           *out,
		Bundle:            true,
		Write:             true,
		Format:            api.FormatESModule,
		Platform:          api.PlatformBrowser,
		Target:            api.ES2022,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		LegalComments:     api.LegalCommentsEndOfFile,
		NodePaths:         []string{abs},
		LogLevel:          api.LogLevelWarning,
	})
	if len(result.Errors) > 0 {
		os.Exit(1)
	}
	st, _ := os.Stat(*out)
	fmt.Printf("✓ %s: %d packages, %d bytes\n", *out, len(lock.Packages), st.Size())
}

// resolve resolves roots ("name@version") and everything they depend on, highest matching version first; one version
// per name, which a bundle needs. A second range for a name must be met by the version already chosen.
func resolve(roots []string) (Lock, error) {
	lock := Lock{Roots: roots, Packages: map[string]Package{}}
	type want struct{ name, rng, from string }
	queue := []want{}
	for _, r := range roots {
		at := strings.LastIndex(r, "@")
		if at <= 0 {
			return lock, fmt.Errorf("%s: want name@version", r)
		}
		queue = append(queue, want{r[:at], r[at+1:], "root"})
	}
	docs := map[string]*packument{}
	for len(queue) > 0 {
		w := queue[0]
		queue = queue[1:]
		c, err := semver.NewConstraint(w.rng)
		if err != nil {
			return lock, fmt.Errorf("%s@%s (from %s): %w", w.name, w.rng, w.from, err)
		}
		if p, ok := lock.Packages[w.name]; ok {
			if v, _ := semver.NewVersion(p.Version); v != nil && c.Check(v) {
				continue
			}
			return lock, fmt.Errorf("%s: %s needs %s, but %s was chosen", w.name, w.from, w.rng, p.Version)
		}
		doc := docs[w.name]
		if doc == nil {
			if doc, err = fetchPackument(w.name); err != nil {
				return lock, err
			}
			docs[w.name] = doc
		}
		var best *semver.Version
		for v := range doc.Versions {
			sv, err := semver.NewVersion(v)
			if err != nil || sv.Prerelease() != "" || !c.Check(sv) {
				continue
			}
			if best == nil || sv.GreaterThan(best) {
				best = sv
			}
		}
		if best == nil {
			return lock, fmt.Errorf("%s: no published version matches %s (from %s)", w.name, w.rng, w.from)
		}
		meta := doc.Versions[best.Original()]
		lock.Packages[w.name] = Package{Version: best.Original(), Tarball: meta.Dist.Tarball, Integrity: meta.Dist.Integrity}
		for _, d := range sortedNames(meta.Dependencies) {
			queue = append(queue, want{d, meta.Dependencies[d], w.name + "@" + best.Original()})
		}
	}
	return lock, nil
}

type packument struct {
	Versions map[string]struct {
		Dependencies map[string]string `json:"dependencies"`
		Dist         struct {
			Tarball   string `json:"tarball"`
			Integrity string `json:"integrity"`
		} `json:"dist"`
	} `json:"versions"`
}

func fetchPackument(name string) (*packument, error) {
	resp, err := http.Get(registry + strings.Replace(url.PathEscape(name), "%40", "@", 1))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: registry answered %s", name, resp.Status)
	}
	var doc packument
	return &doc, json.NewDecoder(resp.Body).Decode(&doc)
}

// install downloads p, checks it against its published integrity, and unpacks it into root/name.
func install(root, name string, p Package) error {
	resp, err := http.Get(p.Tarball)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s@%s: %s", name, p.Version, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := checkIntegrity(data, p.Integrity); err != nil {
		return fmt.Errorf("%s@%s: %w", name, p.Version, err)
	}
	gz, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	dir := filepath.Join(root, filepath.FromSlash(name))
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		// npm tarballs hold one top directory (usually "package/"): drop it, and refuse anything that climbs out.
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok || rel == "" || strings.Contains(rel, "..") {
			continue
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
}

func checkIntegrity(data []byte, integrity string) error {
	algo, want, ok := strings.Cut(integrity, "-")
	if !ok {
		// Very old packages publish only a sha1 shasum as hex.
		sum := sha1.Sum(data)
		if hex.EncodeToString(sum[:]) == integrity {
			return nil
		}
		return errors.New("integrity mismatch")
	}
	var h hash.Hash
	switch algo {
	case "sha512":
		h = sha512.New()
	case "sha1":
		h = sha1.New()
	default:
		return fmt.Errorf("unsupported integrity %s", algo)
	}
	h.Write(data)
	if base64.StdEncoding.EncodeToString(h.Sum(nil)) != want {
		return errors.New("integrity mismatch")
	}
	return nil
}

func sortedNames[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "✗ jsbundle:", err)
	os.Exit(1)
}
