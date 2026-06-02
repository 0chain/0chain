package blockstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeHash(i int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("block_%d", i)))
	return hex.EncodeToString(h[:])
}

func TestPackWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	packPath := filepath.Join(dir, "test.pack")

	// Write a pack with 100 blocks
	pw, err := newPackWriter(packPath)
	require.NoError(t, err)

	hashes := make([]string, 100)
	for i := 0; i < 100; i++ {
		hashes[i] = makeHash(i)
		data := []byte(fmt.Sprintf("block_data_%d", i))
		require.NoError(t, pw.addBlock(hashes[i], data))
	}
	require.NoError(t, pw.finish(packPath))

	// Read back
	pr, err := openPackIndex(packPath)
	require.NoError(t, err)
	assert.Equal(t, 100, len(pr.entries))

	// Lookup each block
	for i := 0; i < 100; i++ {
		data, err := pr.lookup(hashes[i])
		require.NoError(t, err)
		require.NotNil(t, data, "block %d not found", i)
		assert.Equal(t, fmt.Sprintf("block_data_%d", i), string(data))
	}

	// Lookup non-existent
	data, err := pr.lookup("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestPackManifest(t *testing.T) {
	dir := t.TempDir()

	// Create two pack files with different hash ranges
	pack1 := filepath.Join(dir, "000000.pack")
	pw1, err := newPackWriter(pack1)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		h := fmt.Sprintf("1%063d", i) // hashes starting with "1"
		pw1.addBlock(h, []byte("data"))
	}
	require.NoError(t, pw1.finish(pack1))

	pack2 := filepath.Join(dir, "000001.pack")
	pw2, err := newPackWriter(pack2)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		h := fmt.Sprintf("a%063d", i) // hashes starting with "a"
		pw2.addBlock(h, []byte("data_a"))
	}
	require.NoError(t, pw2.finish(pack2))

	// Load manifest
	m := newPackManifest()
	require.NoError(t, m.load(dir))
	assert.Equal(t, 2, m.packCount())

	// Lookup from pack 1
	data, err := m.lookup(fmt.Sprintf("1%063d", 5))
	require.NoError(t, err)
	assert.Equal(t, "data", string(data))

	// Lookup from pack 2
	data, err = m.lookup(fmt.Sprintf("a%063d", 3))
	require.NoError(t, err)
	assert.Equal(t, "data_a", string(data))

	// Lookup miss
	data, err = m.lookup(fmt.Sprintf("f%063d", 0))
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestExtractHashFromPath(t *testing.T) {
	base := "/data/blocks"
	path := "/data/blocks/e/3/b/0/c/44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.dat.zlib"
	hash := extractHashFromPath(base, path)
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
}

func TestCompactorCollectAndPack(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, "packs")
	os.MkdirAll(packsDir, 0700)

	// Create loose files mimicking the hash-tree structure
	for i := 0; i < 20; i++ {
		hash := makeHash(i)
		bp, _ := getBlockFilePath(hash)
		fullPath := filepath.Join(dir, bp)
		os.MkdirAll(filepath.Dir(fullPath), 0700)
		os.WriteFile(fullPath, []byte(fmt.Sprintf("raw_block_%d", i)), 0644)
	}

	manifest := newPackManifest()
	comp := newCompactor(dir, packsDir, manifest)

	// Collect should find all 20 files
	files, err := comp.collectLooseFiles(100)
	require.NoError(t, err)
	assert.Equal(t, 20, len(files))

	// Verify hashes were extracted correctly
	for _, f := range files {
		assert.Len(t, f.hash, 64, "hash should be 64 chars: %s", f.hash)
	}
}

func TestCompactorPackAndDelete(t *testing.T) {
	dir := t.TempDir()
	packsDir := filepath.Join(dir, "packs")
	os.MkdirAll(packsDir, 0700)

	hashes := make([]string, 25)
	for i := 0; i < 25; i++ {
		hashes[i] = makeHash(i)
		bp, _ := getBlockFilePath(hashes[i])
		fullPath := filepath.Join(dir, bp)
		os.MkdirAll(filepath.Dir(fullPath), 0700)
		os.WriteFile(fullPath, []byte(fmt.Sprintf("block_%d", i)), 0644)
	}

	manifest := newPackManifest()
	comp := newCompactor(dir, packsDir, manifest)

	// Pack 25 files (less than blocksPerPack, so compact() won't pack them)
	// Use packFiles directly
	files, err := comp.collectLooseFiles(25)
	require.NoError(t, err)

	n, err := comp.packFiles(files)
	require.NoError(t, err)
	assert.Equal(t, 25, n)

	// Verify pack was created
	assert.Equal(t, 1, manifest.packCount())

	// Verify blocks are readable from pack
	for _, h := range hashes {
		data, err := manifest.lookup(h)
		require.NoError(t, err)
		require.NotNil(t, data, "block %s not found in pack", h)
	}

	// Verify loose files were deleted
	for _, h := range hashes {
		bp, _ := getBlockFilePath(h)
		_, err := os.Stat(filepath.Join(dir, bp))
		assert.True(t, os.IsNotExist(err), "loose file should be deleted: %s", h)
	}
}
