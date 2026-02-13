//go:build darwin

package permissions

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#include <dispatch/dispatch.h>
#include <objc/runtime.h>

// Forward declarations - defined in permissions_darwin.go's CGo preamble,
// linked at the package level within the same Go package.
extern int checkAccessibilityPermission(void);
extern int checkScreenRecordingPermission(void);
extern int requestAccessibilityPermission(void);
extern int requestScreenRecordingPermission(void);
extern int checkScreenRecordingLive(void);

typedef struct {
	int accessibility;
	int screenRecording;
} WizardResult;

// Tracks the wizard window so button handlers can adjust it.
static NSWindow *wizardWindow = nil;

// Semaphore signaled when the wizard closes (Done or X button).
static dispatch_semaphore_t wizardDoneSem = NULL;

// Set to 1 when the user clicks "Open Screen Recording Settings…" button.
// checkScreenRecordingLive() (SCShareableContent) is only called after this
// flag is set, because calling it without permission triggers TCC prompts.
static int _scrButtonClicked = 0;

// Footer label — referenced by doneButtonClicked to show a hint.
static NSTextField *wizardFooter = nil;

// Button action handlers called via Objective-C runtime.
static void openAccessibilitySettings(id self, SEL _cmd, id sender) {
	@autoreleasepool {
		NSURL *url = [NSURL URLWithString:
			@"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"];
		[[NSWorkspace sharedWorkspace] openURL:url];
	}
}

static void openScreenRecordingSettings(id self, SEL _cmd, id sender) {
	@autoreleasepool {
		_scrButtonClicked = 1;
		NSURL *url = [NSURL URLWithString:
			@"x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"];
		[[NSWorkspace sharedWorkspace] openURL:url];
	}
}

static void doneButtonClicked(id self, SEL _cmd, id sender) {
	@autoreleasepool {
		int acc = checkAccessibilityPermission();
		int scr = _scrButtonClicked ? checkScreenRecordingLive() : 0;
		if (!acc || !scr) {
			// Show hint instead of closing
			if (wizardFooter) {
				[wizardFooter setStringValue:
					@"Permission not detected \u2014 try: make cert && make build"];
				[wizardFooter setTextColor:[NSColor systemOrangeColor]];
			}
			return;
		}
		[wizardWindow close];
		wizardWindow = nil;
		wizardFooter = nil;
		if (wizardDoneSem) {
			dispatch_semaphore_signal(wizardDoneSem);
		}
	}
}

// Register a minimal Objective-C class at runtime to serve as button target.
// Returns a singleton instance.
static id getButtonTarget(void) {
	static id instance = nil;
	if (instance) return instance;

	Class cls = objc_allocateClassPair([NSObject class], "MCPWizardButtonTarget", 0);
	// "v@:@" = void return, self, _cmd, sender (id)
	class_addMethod(cls, sel_registerName("openAccessibility:"),
		(IMP)openAccessibilitySettings, "v@:@");
	class_addMethod(cls, sel_registerName("openScreenRecording:"),
		(IMP)openScreenRecordingSettings, "v@:@");
	class_addMethod(cls, sel_registerName("doneClicked:"),
		(IMP)doneButtonClicked, "v@:@");
	objc_registerClassPair(cls);

	instance = [[cls alloc] init];
	return instance;
}

// showPermissionWizard creates a native NSWindow on the main thread and
// blocks the calling thread until the user clicks Done or closes the window.
//
// Modeled after AltTab's permission flow:
// - Normal window level (not floating) — standard macOS z-ordering
// - hidesOnDeactivate=NO — window stays visible when app loses focus
// - Accessibility: polled passively via AXIsProcessTrustedWithOptions (no prompt)
// - Screen Recording: polled via SCShareableContent ONLY after the user clicks
//   the button. CGPreflightScreenCaptureAccess is never used in the wizard
//   because it returns stale results after re-signing.
// - The wizard never auto-closes. The user must click Done.
static WizardResult showPermissionWizard(void) {
	__block WizardResult result = {0, 0};
	dispatch_semaphore_t sem = dispatch_semaphore_create(0);
	wizardDoneSem = sem;

	// Register the current binary in TCC before showing the wizard.
	// This ensures System Settings shows an entry matching our CDHash.
	// Without this, the user may toggle an outdated entry (stale CDHash
	// from a previous build) and AXIsProcessTrustedWithOptions won't
	// detect the grant.
	int preAcc = checkAccessibilityPermission();
	if (!preAcc) {
		requestAccessibilityPermission();
	}

	dispatch_async(dispatch_get_main_queue(), ^{
		@autoreleasepool {
			// --- Window ---
			NSRect frame = NSMakeRect(0, 0, 420, 320);
			NSUInteger styleMask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable;
			NSWindow *window = [[NSWindow alloc]
				initWithContentRect:frame
				styleMask:styleMask
				backing:NSBackingStoreBuffered
				defer:NO];

			[window setTitle:@"MCPMacControl \u2014 Permissions"];
			// Floating level so the wizard stays visible — our app is an
			// LSUIElement (no Dock icon) so a normal-level window would
			// disappear behind other apps with no way to find it.
			[window setLevel:NSFloatingWindowLevel];
			[window setHidesOnDeactivate:NO];
			[window setReleasedWhenClosed:NO];
			[window center];
			wizardWindow = window;

			NSView *content = [window contentView];

			// --- Header ---
			NSTextField *header = [NSTextField labelWithString:
				@"MCPMacControl needs permissions\nto control your Mac."];
			[header setFont:[NSFont systemFontOfSize:14 weight:NSFontWeightMedium]];
			[header setMaximumNumberOfLines:2];
			[header setFrame:NSMakeRect(20, 260, 380, 40)];
			[content addSubview:header];

			// --- Accessibility section ---
			NSTextField *accDot = [NSTextField labelWithString:@"\U0001F534"];
			[accDot setFont:[NSFont systemFontOfSize:16]];
			[accDot setFrame:NSMakeRect(20, 220, 30, 24)];
			[content addSubview:accDot];

			NSTextField *accLabel = [NSTextField labelWithString:@"Accessibility"];
			[accLabel setFont:[NSFont systemFontOfSize:13 weight:NSFontWeightSemibold]];
			[accLabel setFrame:NSMakeRect(50, 220, 200, 20)];
			[content addSubview:accLabel];

			NSTextField *accDesc = [NSTextField labelWithString:
				@"Control mouse, keyboard, and windows"];
			[accDesc setFont:[NSFont systemFontOfSize:11]];
			[accDesc setTextColor:[NSColor secondaryLabelColor]];
			[accDesc setFrame:NSMakeRect(50, 200, 300, 16)];
			[content addSubview:accDesc];

			id target = getButtonTarget();

			NSButton *accBtn = [NSButton buttonWithTitle:@"Open Accessibility Settings\u2026"
				target:target
				action:sel_registerName("openAccessibility:")];
			[accBtn setFrame:NSMakeRect(50, 172, 230, 24)];
			[content addSubview:accBtn];

			// --- Screen Recording section ---
			NSTextField *scrDot = [NSTextField labelWithString:@"\U0001F534"];
			[scrDot setFont:[NSFont systemFontOfSize:16]];
			[scrDot setFrame:NSMakeRect(20, 132, 30, 24)];
			[content addSubview:scrDot];

			NSTextField *scrLabel = [NSTextField labelWithString:@"Screen Recording"];
			[scrLabel setFont:[NSFont systemFontOfSize:13 weight:NSFontWeightSemibold]];
			[scrLabel setFrame:NSMakeRect(50, 132, 200, 20)];
			[content addSubview:scrLabel];

			NSTextField *scrDesc = [NSTextField labelWithString:
				@"Capture screenshots and window list"];
			[scrDesc setFont:[NSFont systemFontOfSize:11]];
			[scrDesc setTextColor:[NSColor secondaryLabelColor]];
			[scrDesc setFrame:NSMakeRect(50, 112, 300, 16)];
			[content addSubview:scrDesc];

			NSButton *scrBtn = [NSButton buttonWithTitle:@"Open Screen Recording Settings\u2026"
				target:target
				action:sel_registerName("openScreenRecording:")];
			[scrBtn setFrame:NSMakeRect(50, 84, 250, 24)];
			[content addSubview:scrBtn];

			// --- Footer ---
			NSTextField *footer = [NSTextField labelWithString:
				@"Grant both permissions, then click Done.\n"
				@"Screen Recording may require an app restart."];
			[footer setFont:[NSFont systemFontOfSize:11]];
			[footer setTextColor:[NSColor secondaryLabelColor]];
			[footer setMaximumNumberOfLines:2];
			[footer setFrame:NSMakeRect(20, 44, 380, 30)];
			[content addSubview:footer];
			wizardFooter = footer;

			// --- Done button ---
			NSButton *doneBtn = [NSButton buttonWithTitle:@"Done"
				target:target
				action:sel_registerName("doneClicked:")];
			[doneBtn setBezelStyle:NSBezelStyleRounded];
			[doneBtn setFrame:NSMakeRect(310, 10, 90, 28)];
			[content addSubview:doneBtn];

			// Reset button-clicked flag for this wizard session.
			_scrButtonClicked = 0;

			// --- Permission polling timer (0.5s) ---
			// Accessibility: passive check, safe to poll always.
			// Screen Recording: only probed via SCShareableContent after
			// the user clicks the button. Never uses CGPreflightScreenCaptureAccess
			// because it returns stale results after re-signing.
			[NSTimer scheduledTimerWithTimeInterval:0.5
				repeats:YES
				block:^(NSTimer *t) {
					// Passive — never triggers a prompt.
					int acc = checkAccessibilityPermission();

					// Only probe screen recording after the user clicked the
					// button. Before that, dot stays red.
					int scr = _scrButtonClicked ? checkScreenRecordingLive() : 0;

					[accDot setStringValue:(acc ? @"\U0001F7E2" : @"\U0001F534")];
					[scrDot setStringValue:(scr ? @"\U0001F7E2" : @"\U0001F534")];

					// Hide buttons once granted
					if (acc) {
						[accBtn setHidden:YES];
						[accDesc setStringValue:@"Granted"];
					}
					if (scr) {
						[scrBtn setHidden:YES];
						[scrDesc setStringValue:@"Granted"];
					}

					result.accessibility = acc;
					result.screenRecording = scr;

					// Update header when both are granted
					if (acc && scr) {
						[header setStringValue:@"\u2705 All permissions granted!"];
						[header setFont:[NSFont systemFontOfSize:14 weight:NSFontWeightBold]];
					}

					// Window was closed by user (X button or Done)
					if (![window isVisible]) {
						[t invalidate];
						wizardWindow = nil;
						wizardFooter = nil;
						wizardDoneSem = NULL;
						dispatch_semaphore_signal(sem);
					}
				}];

			// Show and activate
			[window makeKeyAndOrderFront:nil];
			[NSApp activateIgnoringOtherApps:YES];
		}
	});

	// Block calling goroutine until wizard closes
	dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

	// Re-check permissions after wizard closes.
	result.accessibility = checkAccessibilityPermission();
	result.screenRecording = checkScreenRecordingLive();
	wizardDoneSem = NULL;
	return result;
}
*/
import "C"

import (
	"sync"
)

var (
	wizardMu     sync.Mutex
	wizardActive bool
	wizardResult chan Status
)

// ShowPermissionWizard displays the native macOS permission wizard window.
// It blocks until the user grants all permissions or closes the window.
// Concurrent callers wait on the same result.
func ShowPermissionWizard() Status {
	wizardMu.Lock()

	if wizardActive {
		// Another goroutine is already showing the wizard - wait for its result.
		ch := wizardResult
		wizardMu.Unlock()
		return <-ch
	}

	// First caller - set up result channel and show the wizard.
	wizardActive = true
	wizardResult = make(chan Status)
	wizardMu.Unlock()

	r := C.showPermissionWizard()
	status := Status{
		Accessibility:   r.accessibility != 0,
		ScreenRecording: r.screenRecording != 0,
	}

	wizardMu.Lock()
	ch := wizardResult
	wizardActive = false
	wizardResult = nil
	wizardMu.Unlock()

	// Broadcast result to all waiting goroutines, then close the channel.
	go func() {
		for {
			select {
			case ch <- status:
			default:
				close(ch)
				return
			}
		}
	}()

	return status
}
