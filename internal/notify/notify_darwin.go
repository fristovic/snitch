//go:build darwin

package notify

/*
#cgo LDFLAGS: -framework Foundation -framework AppKit
#include <stdlib.h>
void snitchDeliverNotification(const char *title, const char *body);
*/
import "C"
import "unsafe"

// Notify shows a macOS Notification Center alert attributed to the current
// process's app bundle (Snitch Bar.app), so the Snitch icon is used instead of
// the Script Editor glyph that osascript produces.
//
// notify_darwin.m implements snitchDeliverNotification and must be linked on
// Darwin builds. NSUserNotification is deprecated but intentional for now.
func Notify(title, body string) error {
	ct := C.CString(title)
	cb := C.CString(body)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	C.snitchDeliverNotification(ct, cb)
	return nil
}
