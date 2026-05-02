package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func seedFuzz(f *testing.F, seedPath string) []byte {
	f.Helper()
	data, err := os.ReadFile(seedPath)
	if err != nil {
		f.Fatalf("read seed %s: %v", seedPath, err)
	}
	f.Add(data)
	return data
}

func runExtractFuzz(t *testing.T, name string, src []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ExtractFile(%s) panicked: %v", name, r)
		}
	}()
	_, _ = ExtractFile(name, src)
}

func FuzzExtractGo(f *testing.F) {
	for _, p := range []string{"testdata/goproj/main.go", "testdata/goproj/service.go"} {
		seedFuzz(f, filepath.Clean(p))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		runExtractFuzz(t, "fuzz.go", src)
	})
}

func FuzzExtractPython(f *testing.F) {
	for _, p := range []string{"testdata/pyproj/main.py", "testdata/pyproj/service.py"} {
		seedFuzz(f, filepath.Clean(p))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		runExtractFuzz(t, "fuzz.py", src)
	})
}

func FuzzExtractJS(f *testing.F) {
	for _, p := range []string{"testdata/jsproj/main.js", "testdata/jsproj/service.js"} {
		seedFuzz(f, filepath.Clean(p))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		runExtractFuzz(t, "fuzz.js", src)
	})
}

func FuzzExtractTS(f *testing.F) {
	for _, p := range []string{"testdata/tsproj/main.ts", "testdata/tsproj/service.ts"} {
		seedFuzz(f, filepath.Clean(p))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		runExtractFuzz(t, "fuzz.ts", src)
	})
}
