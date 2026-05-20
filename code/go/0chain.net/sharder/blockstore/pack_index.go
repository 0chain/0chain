package blockstore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// packRange tracks the hash range covered by a single pack file.
type packRange struct {
	path    string // full path to .pack file
	minHash string
	maxHash string
}

// packManifest is an in-memory index of all pack files, sorted by minHash.
// It enables quick lookup of which pack file(s) might contain a given hash.
type packManifest struct {
	mu    sync.RWMutex
	packs []packRange

	// cache of opened pack indices (path → *packReader)
	cacheMu sync.RWMutex
	cache   map[string]*packReader
}

func newPackManifest() *packManifest {
	return &packManifest{
		cache: make(map[string]*packReader),
	}
}

// load scans the packs directory and builds the manifest from pack files.
func (m *packManifest) load(packsDir string) error {
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var ranges []packRange
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pack") {
			continue
		}
		path := filepath.Join(packsDir, e.Name())
		minH, maxH, err := readPackHashRange(path)
		if err != nil {
			logging.Logger.Warn("skipping corrupt pack file",
				zap.String("path", path), zap.Error(err))
			continue
		}
		if minH == "" {
			continue
		}
		ranges = append(ranges, packRange{
			path:    path,
			minHash: minH,
			maxHash: maxH,
		})
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].minHash < ranges[j].minHash
	})

	m.mu.Lock()
	m.packs = ranges
	m.mu.Unlock()

	logging.Logger.Info("pack manifest loaded",
		zap.Int("pack_count", len(ranges)))
	return nil
}

// addPack adds a newly created pack to the manifest.
func (m *packManifest) addPack(path string, pr *packReader) {
	r := packRange{
		path:    path,
		minHash: pr.minHash(),
		maxHash: pr.maxHash(),
	}

	m.mu.Lock()
	m.packs = append(m.packs, r)
	sort.Slice(m.packs, func(i, j int) bool {
		return m.packs[i].minHash < m.packs[j].minHash
	})
	m.mu.Unlock()

	m.cacheMu.Lock()
	m.cache[path] = pr
	m.cacheMu.Unlock()
}

// findCandidates returns pack file paths that might contain the given hash.
func (m *packManifest) findCandidates(hash string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidates []string
	for _, p := range m.packs {
		if hash >= p.minHash && hash <= p.maxHash {
			candidates = append(candidates, p.path)
		}
	}
	return candidates
}

// lookup searches all candidate packs for a block hash.
// Returns the raw compressed bytes or nil if not found.
func (m *packManifest) lookup(hash string) ([]byte, error) {
	candidates := m.findCandidates(hash)
	for _, path := range candidates {
		pr, err := m.getReader(path)
		if err != nil {
			logging.Logger.Warn("failed to open pack index",
				zap.String("path", path), zap.Error(err))
			continue
		}
		data, err := pr.lookup(hash)
		if err != nil {
			return nil, err
		}
		if data != nil {
			return data, nil
		}
	}
	return nil, nil
}

// getReader returns a cached pack reader or opens one.
func (m *packManifest) getReader(path string) (*packReader, error) {
	m.cacheMu.RLock()
	pr, ok := m.cache[path]
	m.cacheMu.RUnlock()
	if ok {
		return pr, nil
	}

	pr, err := openPackIndex(path)
	if err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache[path] = pr
	m.cacheMu.Unlock()
	return pr, nil
}

// packCount returns the number of loaded packs.
func (m *packManifest) packCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.packs)
}
