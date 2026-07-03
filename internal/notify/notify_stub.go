//go:build !darwin

package notify

// Notify is a no-op on non-macOS platforms.
func Notify(title, body string) error {
	return nil
}
