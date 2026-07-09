//go:build darwin

package notify

/*
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdlib.h>
void snitchInitNotifications(void);
void snitchDeliverNotification(const char *title, const char *body);
*/
import "C"
import "unsafe"

// Init requests Notification Center authorization and installs a foreground
// presentation delegate. Safe to call multiple times; call once at app start
// so the macOS permission prompt can appear before the first fail alert.
func Init() {
	C.snitchInitNotifications()
}

// Notify shows a macOS Notification Center alert attributed to the current
// process's app bundle (Snitch Bar.app), so the Snitch icon is used instead of
// the Script Editor glyph that osascript produces.
//
// Uses UNUserNotificationCenter (Apple's supported API). notify_darwin.m
// implements the ObjC side and must be linked on Darwin builds.
func Notify(title, body string) error {
	ct := C.CString(title)
	cb := C.CString(body)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	C.snitchDeliverNotification(ct, cb)
	return nil
}
