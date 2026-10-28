// Package testassets separates opt-in real-database tests from hermetic tests.
package testassets

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Path enables a real-database test only when IPINFO_INTEGRATION=1.
// Missing files are failures in this mode, so a green run cannot hide missing coverage.
func Path(t testing.TB, name string) string {
	t.Helper()
	if os.Getenv("IPINFO_INTEGRATION") != "1" {
		t.Skip("真实数据库集成测试：设置 IPINFO_INTEGRATION=1 并先运行 python scripts/fetch_test_assets.py")
	}
	dir := os.Getenv("IPINFO_ASSETS_DIR")
	if dir == "" {
		_, file, _, _ := runtime.Caller(0)
		dir = filepath.Join(filepath.Dir(file), "..", "..", "assets")
	}
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		t.Fatalf("集成测试数据库不可用：%s；运行 python scripts/fetch_test_assets.py（%v）", path, err)
	}
	return path
}
