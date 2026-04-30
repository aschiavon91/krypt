//go:build windows

package editor

func applyUmask(_ int) (restore func()) {
	return func() {}
}
