package store

import (
	"sync"
	"testing"

	"github.com/crossplane-contrib/function-hcl/function-hcl-ls/internal/funchcl/target"
	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopyMap tests the copyMap utility function.
func TestCopyMap(t *testing.T) {
	t.Run("copies string map", func(t *testing.T) {
		original := map[string]string{"a": "1", "b": "2"}
		copied := copyMap(original)

		assert.Equal(t, original, copied)
		// Verify it's a copy, not the same map
		assert.NotSame(t, &original, &copied)

		// Modify original and verify copy is unchanged
		original["c"] = "3"
		assert.Len(t, copied, 2)
		assert.NotContains(t, copied, "c")
	})

	t.Run("copies int map", func(t *testing.T) {
		original := map[string]int{"x": 10, "y": 20}
		copied := copyMap(original)

		assert.Equal(t, original, copied)
		original["z"] = 30
		assert.Len(t, copied, 2)
	})

	t.Run("copies empty map", func(t *testing.T) {
		original := map[string]string{}
		copied := copyMap(original)

		assert.Empty(t, copied)
		assert.NotNil(t, copied)
	})

	t.Run("copies hcl.File map", func(t *testing.T) {
		file1 := &hcl.File{}
		file2 := &hcl.File{}
		original := map[string]*hcl.File{"file1.hcl": file1, "file2.hcl": file2}
		copied := copyMap(original)

		assert.Equal(t, original, copied)
		assert.Same(t, file1, copied["file1.hcl"]) // pointers are copied

		// Verify it's a new map
		original["file3.hcl"] = &hcl.File{}
		assert.Len(t, copied, 2)
	})

	t.Run("copies hcl.Diagnostics map", func(t *testing.T) {
		diag1 := hcl.Diagnostics{&hcl.Diagnostic{Summary: "error1"}}
		diag2 := hcl.Diagnostics{&hcl.Diagnostic{Summary: "error2"}}
		original := map[string]hcl.Diagnostics{"file1": diag1, "file2": diag2}
		copied := copyMap(original)

		assert.Equal(t, original, copied)
		original["file3"] = hcl.Diagnostics{}
		assert.Len(t, copied, 2)
	})
}

// TestXRD tests the XRD struct.
func TestXRD(t *testing.T) {
	xrd := &XRD{
		APIVersion: "example.crossplane.io/v1",
		Kind:       "XComposite",
	}

	assert.Equal(t, "example.crossplane.io/v1", xrd.APIVersion)
	assert.Equal(t, "XComposite", xrd.Kind)
}

// TestContentToModule tests the Content.toModule() conversion.
func TestContentToModule(t *testing.T) {
	t.Run("converts with all fields populated", func(t *testing.T) {
		file := &hcl.File{}
		diags := hcl.Diagnostics{&hcl.Diagnostic{Summary: "test error"}}
		targets := &target.Targets{}
		refMap := &target.ReferenceMap{
			DefToRefs: target.DefToRefs{},
			RefsToDef: target.RefsToDef{},
		}
		xrd := &XRD{APIVersion: "v1", Kind: "Test"}

		content := &Content{
			Path:    "/path/to/module",
			Files:   map[string]*hcl.File{"test.hcl": file},
			Diags:   map[string]hcl.Diagnostics{"test.hcl": diags},
			Targets: targets,
			RefMap:  refMap,
			XRD:     xrd,
		}

		mod := content.toModule()

		require.NotNil(t, mod)
		assert.Equal(t, "/path/to/module", mod.path)
		assert.Equal(t, 1, len(mod.files))
		assert.Same(t, file, mod.files["test.hcl"])
		assert.Equal(t, 1, len(mod.diags))
		assert.Equal(t, diags, mod.diags["test.hcl"])
		assert.Same(t, targets, mod.targets)
		assert.Same(t, refMap, mod.refMap)
		assert.Same(t, xrd, mod.xrd)
	})

	t.Run("converts with nil optional fields", func(t *testing.T) {
		content := &Content{
			Path:    "/another/path",
			Files:   map[string]*hcl.File{},
			Diags:   map[string]hcl.Diagnostics{},
			Targets: nil,
			RefMap:  nil,
			XRD:     nil,
		}

		mod := content.toModule()

		require.NotNil(t, mod)
		assert.Equal(t, "/another/path", mod.path)
		assert.Empty(t, mod.files)
		assert.Empty(t, mod.diags)
		assert.Nil(t, mod.targets)
		assert.Nil(t, mod.refMap)
		assert.Nil(t, mod.xrd)
	})

	t.Run("creates independent copies of maps", func(t *testing.T) {
		files := map[string]*hcl.File{"test.hcl": {}}
		diags := map[string]hcl.Diagnostics{"test.hcl": {}}

		content := &Content{
			Path:  "/path",
			Files: files,
			Diags: diags,
		}

		mod := content.toModule()

		// Modify original maps
		files["new.hcl"] = &hcl.File{}
		diags["new.hcl"] = hcl.Diagnostics{}

		// Module should have independent copies
		assert.Len(t, mod.files, 1)
		assert.Len(t, mod.diags, 1)
	})
}

// TestModuleContent tests the module.Content() conversion.
func TestModuleContent(t *testing.T) {
	t.Run("converts back to Content", func(t *testing.T) {
		file := &hcl.File{}
		diags := hcl.Diagnostics{&hcl.Diagnostic{Summary: "test"}}
		targets := &target.Targets{}
		refMap := &target.ReferenceMap{}
		xrd := &XRD{APIVersion: "v1", Kind: "Test"}

		mod := &module{
			path:    "/module/path",
			files:   map[string]*hcl.File{"file.hcl": file},
			diags:   map[string]hcl.Diagnostics{"file.hcl": diags},
			targets: targets,
			refMap:  refMap,
			xrd:     xrd,
		}

		content := mod.Content()

		require.NotNil(t, content)
		assert.Equal(t, "/module/path", content.Path)
		assert.Equal(t, 1, len(content.Files))
		assert.Same(t, file, content.Files["file.hcl"])
		assert.Equal(t, 1, len(content.Diags))
		assert.Equal(t, diags, content.Diags["file.hcl"])
		assert.Same(t, targets, content.Targets)
		assert.Same(t, refMap, content.RefMap)
		assert.Same(t, xrd, content.XRD)
	})

	t.Run("creates independent copies of maps", func(t *testing.T) {
		mod := &module{
			path:  "/path",
			files: map[string]*hcl.File{"test.hcl": {}},
			diags: map[string]hcl.Diagnostics{"test.hcl": {}},
		}

		content := mod.Content()

		// Modify module maps
		mod.files["new.hcl"] = &hcl.File{}
		mod.diags["new.hcl"] = hcl.Diagnostics{}

		// Content should have independent copies
		assert.Len(t, content.Files, 1)
		assert.Len(t, content.Diags, 1)
	})
}

// TestNew tests the Store constructor.
func TestNew(t *testing.T) {
	store := New()

	require.NotNil(t, store)
	assert.NotNil(t, store.modules)
	assert.Empty(t, store.modules)
}

// TestStorePutAndGet tests adding and retrieving content.
func TestStorePutAndGet(t *testing.T) {
	t.Run("put and get single module", func(t *testing.T) {
		store := New()
		content := &Content{
			Path:  "/path/to/module",
			Files: map[string]*hcl.File{"test.hcl": {}},
			Diags: map[string]hcl.Diagnostics{"test.hcl": {}},
			XRD:   &XRD{APIVersion: "v1", Kind: "Test"},
		}

		store.Put(content)
		retrieved := store.Get("/path/to/module")

		require.NotNil(t, retrieved)
		assert.Equal(t, "/path/to/module", retrieved.Path)
		assert.Len(t, retrieved.Files, 1)
		assert.Len(t, retrieved.Diags, 1)
		assert.Equal(t, "v1", retrieved.XRD.APIVersion)
	})

	t.Run("get returns nil for non-existent module", func(t *testing.T) {
		store := New()
		retrieved := store.Get("/non/existent")

		assert.Nil(t, retrieved)
	})

	t.Run("put overwrites existing module", func(t *testing.T) {
		store := New()

		content1 := &Content{
			Path:  "/module",
			Files: map[string]*hcl.File{"file1.hcl": {}},
		}
		store.Put(content1)

		content2 := &Content{
			Path:  "/module",
			Files: map[string]*hcl.File{"file2.hcl": {}, "file3.hcl": {}},
		}
		store.Put(content2)

		retrieved := store.Get("/module")
		require.NotNil(t, retrieved)
		assert.Len(t, retrieved.Files, 2)
		assert.Contains(t, retrieved.Files, "file2.hcl")
		assert.Contains(t, retrieved.Files, "file3.hcl")
	})

	t.Run("get returns independent copy", func(t *testing.T) {
		store := New()
		content := &Content{
			Path:  "/module",
			Files: map[string]*hcl.File{"file1.hcl": {}},
		}
		store.Put(content)

		// Get twice
		copy1 := store.Get("/module")
		copy2 := store.Get("/module")

		require.NotNil(t, copy1)
		require.NotNil(t, copy2)

		// They should be equal but not the same object
		assert.Equal(t, copy1, copy2)
		assert.NotSame(t, copy1, copy2)

		// Modifying one shouldn't affect the other
		copy1.Files["new.hcl"] = &hcl.File{}
		assert.Len(t, copy2.Files, 1)
	})
}

// TestStoreExists tests the Exists method.
func TestStoreExists(t *testing.T) {
	t.Run("returns true for existing module", func(t *testing.T) {
		store := New()
		content := &Content{Path: "/module"}
		store.Put(content)

		assert.True(t, store.Exists("/module"))
	})

	t.Run("returns false for non-existent module", func(t *testing.T) {
		store := New()
		assert.False(t, store.Exists("/non/existent"))
	})

	t.Run("returns false after removal", func(t *testing.T) {
		store := New()
		content := &Content{Path: "/module"}
		store.Put(content)
		store.Remove("/module")

		assert.False(t, store.Exists("/module"))
	})
}

// TestStoreRemove tests the Remove method.
func TestStoreRemove(t *testing.T) {
	t.Run("removes existing module", func(t *testing.T) {
		store := New()
		content := &Content{Path: "/module"}
		store.Put(content)

		assert.True(t, store.Exists("/module"))
		store.Remove("/module")
		assert.False(t, store.Exists("/module"))
		assert.Nil(t, store.Get("/module"))
	})

	t.Run("remove non-existent module is no-op", func(t *testing.T) {
		store := New()

		// Should not panic
		store.Remove("/non/existent")
		assert.False(t, store.Exists("/non/existent"))
	})

	t.Run("remove doesn't affect other modules", func(t *testing.T) {
		store := New()
		store.Put(&Content{Path: "/module1"})
		store.Put(&Content{Path: "/module2"})

		store.Remove("/module1")

		assert.False(t, store.Exists("/module1"))
		assert.True(t, store.Exists("/module2"))
	})
}

// TestStoreListDirs tests the ListDirs method.
func TestStoreListDirs(t *testing.T) {
	t.Run("returns empty list for new store", func(t *testing.T) {
		store := New()
		dirs := store.ListDirs()

		assert.Empty(t, dirs)
	})

	t.Run("returns all module directories", func(t *testing.T) {
		store := New()
		store.Put(&Content{Path: "/module1"})
		store.Put(&Content{Path: "/module2"})
		store.Put(&Content{Path: "/module3"})

		dirs := store.ListDirs()

		assert.Len(t, dirs, 3)
		assert.Contains(t, dirs, "/module1")
		assert.Contains(t, dirs, "/module2")
		assert.Contains(t, dirs, "/module3")
	})

	t.Run("reflects removed modules", func(t *testing.T) {
		store := New()
		store.Put(&Content{Path: "/module1"})
		store.Put(&Content{Path: "/module2"})
		store.Remove("/module1")

		dirs := store.ListDirs()

		assert.Len(t, dirs, 1)
		assert.Contains(t, dirs, "/module2")
		assert.NotContains(t, dirs, "/module1")
	})

	t.Run("reflects updated modules", func(t *testing.T) {
		store := New()
		store.Put(&Content{Path: "/module1"})
		store.Put(&Content{Path: "/module1"}) // update

		dirs := store.ListDirs()

		assert.Len(t, dirs, 1)
		assert.Contains(t, dirs, "/module1")
	})
}

// TestStoreConcurrency tests concurrent access to the store.
func TestStoreConcurrency(t *testing.T) {
	t.Run("concurrent Put operations", func(t *testing.T) {
		store := New()
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				content := &Content{
					Path:  "/module",
					Files: map[string]*hcl.File{},
				}
				store.Put(content)
			}(i)
		}

		wg.Wait()
		assert.True(t, store.Exists("/module"))
	})

	t.Run("concurrent Put and Get operations", func(t *testing.T) {
		store := New()
		var wg sync.WaitGroup

		// Pre-populate
		for i := 0; i < 10; i++ {
			store.Put(&Content{
				Path:  "/module" + string(rune('0'+i)),
				Files: map[string]*hcl.File{},
			})
		}

		// Concurrent reads and writes
		for i := 0; i < 50; i++ {
			wg.Add(2)

			// Reader
			go func(id int) {
				defer wg.Done()
				_ = store.Get("/module" + string(rune('0'+(id%10))))
			}(i)

			// Writer
			go func(id int) {
				defer wg.Done()
				store.Put(&Content{
					Path:  "/module" + string(rune('0'+(id%10))),
					Files: map[string]*hcl.File{},
				})
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent ListDirs and Put operations", func(t *testing.T) {
		store := New()
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				store.Put(&Content{
					Path:  "/module" + string(rune('0'+(id%10))),
					Files: map[string]*hcl.File{},
				})
			}(i)
		}

		// Readers
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = store.ListDirs()
			}()
		}

		wg.Wait()

		// Should have modules
		assert.NotEmpty(t, store.ListDirs())
	})

	t.Run("concurrent Exists and Remove operations", func(t *testing.T) {
		store := New()

		// Pre-populate
		for i := 0; i < 10; i++ {
			store.Put(&Content{Path: "/module" + string(rune('0'+i))})
		}

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(2)

			go func(id int) {
				defer wg.Done()
				_ = store.Exists("/module" + string(rune('0'+(id%10))))
			}(i)

			go func(id int) {
				defer wg.Done()
				store.Remove("/module" + string(rune('0'+(id%10))))
			}(i)
		}

		wg.Wait()
	})
}

// TestRoundTrip tests converting Content to module and back.
func TestRoundTrip(t *testing.T) {
	file := &hcl.File{}
	diags := hcl.Diagnostics{&hcl.Diagnostic{Summary: "test"}}
	targets := &target.Targets{}
	refMap := &target.ReferenceMap{
		DefToRefs: target.DefToRefs{},
		RefsToDef: target.RefsToDef{},
	}
	xrd := &XRD{APIVersion: "v1", Kind: "Test"}

	original := &Content{
		Path:    "/test/module",
		Files:   map[string]*hcl.File{"test.hcl": file},
		Diags:   map[string]hcl.Diagnostics{"test.hcl": diags},
		Targets: targets,
		RefMap:  refMap,
		XRD:     xrd,
	}

	// Convert to module and back
	mod := original.toModule()
	roundTripped := mod.Content()

	assert.Equal(t, original.Path, roundTripped.Path)
	assert.Equal(t, len(original.Files), len(roundTripped.Files))
	assert.Same(t, original.Files["test.hcl"], roundTripped.Files["test.hcl"])
	assert.Equal(t, len(original.Diags), len(roundTripped.Diags))
	assert.Equal(t, original.Diags["test.hcl"], roundTripped.Diags["test.hcl"])
	assert.Same(t, original.Targets, roundTripped.Targets)
	assert.Same(t, original.RefMap, roundTripped.RefMap)
	assert.Same(t, original.XRD, roundTripped.XRD)
}
