#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

// UNUserNotificationCenter is Apple's supported API.

@interface SnitchNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation SnitchNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	// Menubar apps stay "foreground"; without this, banners are suppressed.
	completionHandler(UNNotificationPresentationOptionBanner |
	                  UNNotificationPresentationOptionList |
	                  UNNotificationPresentationOptionSound);
}
@end

static SnitchNotificationDelegate *gSnitchNotifyDelegate;

static void snitchInitNotificationsMain(void) {
	if (gSnitchNotifyDelegate != nil) {
		return;
	}
	gSnitchNotifyDelegate = [[SnitchNotificationDelegate alloc] init];
	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	center.delegate = gSnitchNotifyDelegate;
	UNAuthorizationOptions opts = UNAuthorizationOptionAlert | UNAuthorizationOptionSound;
	[center requestAuthorizationWithOptions:opts
		completionHandler:^(BOOL granted, NSError *_Nullable error) {
			if (error != nil) {
				NSLog(@"snitch notify auth error: %@", error);
			} else if (!granted) {
				NSLog(@"snitch notify auth denied");
			}
		}];
}

void snitchInitNotifications(void) {
	if ([NSThread isMainThread]) {
		snitchInitNotificationsMain();
	} else {
		dispatch_sync(dispatch_get_main_queue(), ^{
			snitchInitNotificationsMain();
		});
	}
}

void snitchDeliverNotification(const char *title, const char *body) {
	// Copy into NSStrings on the calling thread BEFORE any async hop —
	// the Go caller frees the C strings as soon as this function returns.
	NSString *titleStr = [NSString stringWithUTF8String:title ? title : ""];
	NSString *bodyStr = [NSString stringWithUTF8String:body ? body : ""];

	void (^deliver)(void) = ^{
		// Init is expected at app start; keep a lazy fallback for tests.
		snitchInitNotificationsMain();

		UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
		content.title = titleStr;
		content.body = bodyStr;
		content.sound = [UNNotificationSound defaultSound];

		NSString *ident = [[NSUUID UUID] UUIDString];
		UNNotificationRequest *req =
			[UNNotificationRequest requestWithIdentifier:ident content:content trigger:nil];
		[[UNUserNotificationCenter currentNotificationCenter]
			addNotificationRequest:req
			withCompletionHandler:^(NSError *_Nullable error) {
				if (error != nil) {
					NSLog(@"snitch notify add error: %@", error);
				}
			}];
	};

	if ([NSThread isMainThread]) {
		deliver();
	} else {
		dispatch_async(dispatch_get_main_queue(), deliver);
	}
}
