//go:build !darwin

package notify

// Init is a no-op on non-macOS platforms.
func Init() {}

// Notify is a no-op on non-macOS platforms.
func Notify(title, body string) error {
	return nil
}
