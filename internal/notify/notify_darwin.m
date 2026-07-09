#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>

void snitchDeliverNotification(const char *title, const char *body) {
	@autoreleasepool {
		NSUserNotification *n = [[NSUserNotification alloc] init];
		n.title = [NSString stringWithUTF8String:title ? title : ""];
		n.informativeText = [NSString stringWithUTF8String:body ? body : ""];
		n.soundName = NSUserNotificationDefaultSoundName;
		[[NSUserNotificationCenter defaultUserNotificationCenter] deliverNotification:n];
	}
}
