#import <Cocoa/Cocoa.h>
#include <CoreGraphics/CoreGraphics.h>

// playTinkSound plays the system "Tink" sound asynchronously on the main thread.
void playTinkSound(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSSound *sound = [NSSound soundNamed:@"Tink"];
        [sound play];
    });
}

// createBorderOverlay creates a borderless, click-through overlay window with
// the given border color on every active display. Returns an NSArray of windows.
static NSArray* createBorderOverlays(NSColor *color) {
    NSMutableArray *windows = [NSMutableArray array];

    uint32_t displayCount = 0;
    CGGetActiveDisplayList(0, NULL, &displayCount);
    if (displayCount == 0) return windows;

    CGDirectDisplayID displays[displayCount];
    CGGetActiveDisplayList(displayCount, displays, &displayCount);

    for (uint32_t i = 0; i < displayCount; i++) {
        CGRect bounds = CGDisplayBounds(displays[i]);
        NSRect frame = NSMakeRect(bounds.origin.x, bounds.origin.y,
                                  bounds.size.width, bounds.size.height);

        NSWindow *overlay = [[NSWindow alloc]
            initWithContentRect:frame
                      styleMask:NSWindowStyleMaskBorderless
                        backing:NSBackingStoreBuffered
                          defer:NO];

        [overlay setLevel:CGShieldingWindowLevel() - 1];
        [overlay setOpaque:NO];
        [overlay setBackgroundColor:[NSColor clearColor]];
        [overlay setIgnoresMouseEvents:YES];
        [overlay setHasShadow:NO];
        [overlay setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
                                       NSWindowCollectionBehaviorStationary];

        NSView *contentView = [overlay contentView];
        [contentView setWantsLayer:YES];
        CALayer *layer = [contentView layer];
        layer.borderColor = [color CGColor];
        layer.borderWidth = 8.0;
        layer.cornerRadius = 0.0;

        [windows addObject:overlay];
    }
    return windows;
}

// Warn flash state — managed on main thread only.
static NSTimer *warnTimer = nil;
static NSArray *warnOverlays = nil;
static BOOL warnVisible = NO;

// startWarnFlash shows a persistent red border that flashes on/off.
void startWarnFlash(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        // Stop any existing flash first
        if (warnTimer) {
            [warnTimer invalidate];
            warnTimer = nil;
        }
        if (warnOverlays) {
            for (NSWindow *w in warnOverlays) [w orderOut:nil];
            warnOverlays = nil;
        }

        warnOverlays = createBorderOverlays([NSColor redColor]);
        warnVisible = YES;
        for (NSWindow *w in warnOverlays) [w orderFrontRegardless];

        // Flash: toggle visibility every 600ms
        warnTimer = [NSTimer scheduledTimerWithTimeInterval:0.6
                                                    repeats:YES
                                                      block:^(NSTimer *timer) {
            warnVisible = !warnVisible;
            for (NSWindow *w in warnOverlays) {
                if (warnVisible) {
                    [w orderFrontRegardless];
                } else {
                    [w orderOut:nil];
                }
            }
        }];
    });
}

// stopWarnFlash stops the red flash and shows a brief green border.
void stopWarnFlash(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        // Stop red flash
        if (warnTimer) {
            [warnTimer invalidate];
            warnTimer = nil;
        }
        if (warnOverlays) {
            for (NSWindow *w in warnOverlays) [w orderOut:nil];
            warnOverlays = nil;
        }
        warnVisible = NO;

        // Brief green flash
        NSArray *greenOverlays = createBorderOverlays([NSColor systemGreenColor]);
        for (NSWindow *w in greenOverlays) [w orderFrontRegardless];

        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(800 * NSEC_PER_MSEC)),
                       dispatch_get_main_queue(), ^{
            for (NSWindow *w in greenOverlays) [w orderOut:nil];
        });
    });
}

// --- Popover notification (anchored to systray icon) ---

#import <objc/runtime.h>

// PopoverViewController provides the content view for the NSPopover.
// Two lines: project name (bold) on top, tool description below.
@interface PopoverViewController : NSViewController
@property (nonatomic, strong) NSTextField *titleLabel;
@property (nonatomic, strong) NSTextField *detailLabel;
@end

@implementation PopoverViewController

- (void)loadView {
    CGFloat w = 300;
    CGFloat h = 52;
    NSView *container = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, w, h)];

    // Project name (top line)
    self.titleLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(12, 28, w - 24, 18)];
    [self.titleLabel setBezeled:NO];
    [self.titleLabel setDrawsBackground:NO];
    [self.titleLabel setEditable:NO];
    [self.titleLabel setSelectable:NO];
    [self.titleLabel setFont:[NSFont monospacedSystemFontOfSize:12 weight:NSFontWeightBold]];
    [self.titleLabel setLineBreakMode:NSLineBreakByTruncatingTail];
    [self.titleLabel setAlignment:NSTextAlignmentCenter];
    [container addSubview:self.titleLabel];

    // Tool description (bottom line)
    self.detailLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(12, 6, w - 24, 18)];
    [self.detailLabel setBezeled:NO];
    [self.detailLabel setDrawsBackground:NO];
    [self.detailLabel setEditable:NO];
    [self.detailLabel setSelectable:NO];
    [self.detailLabel setFont:[NSFont monospacedSystemFontOfSize:11 weight:NSFontWeightRegular]];
    [self.detailLabel setLineBreakMode:NSLineBreakByTruncatingTail];
    [self.detailLabel setAlignment:NSTextAlignmentCenter];
    [container addSubview:self.detailLabel];

    self.view = container;
}

@end

static NSPopover *statusPopover = nil;
static PopoverViewController *popoverVC = nil;
static NSTimer *popoverTimer = nil;

// Minimum time the popover stays visible (seconds).
static const NSTimeInterval kPopoverMinDuration = 5.0;

// getStatusItemButton returns the NSStatusItem's button from the systray
// library's AppDelegate, accessed via the Objective-C runtime.
// Returns nil if the app delegate or status item is not available.
static NSStatusBarButton* getStatusItemButton(void) {
    id delegate = [NSApp delegate];
    if (!delegate) return nil;

    Ivar ivar = class_getInstanceVariable([delegate class], "statusItem");
    if (!ivar) return nil;

    NSStatusItem *si = object_getIvar(delegate, ivar);
    if (!si) return nil;

    return si.button;
}

// fadeAndClosePopover fades the popover content to transparent over 0.5s,
// then closes the popover and resets alpha for the next show.
static void fadeAndClosePopover(void) {
    if (!statusPopover || !statusPopover.shown) return;

    NSView *contentView = popoverVC.view;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *ctx) {
        ctx.duration = 0.5;
        contentView.animator.alphaValue = 0.0;
    } completionHandler:^{
        [statusPopover performClose:nil];
        contentView.alphaValue = 1.0;
    }];
}

// showBalloon displays a native popover anchored to the systray icon.
// Text format: "title\ndetail" — first line is project name, second is tool info.
// If no newline, the entire text goes to the detail line.
void showBalloon(const char* text) {
    char *copy = strdup(text);
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString *str = [NSString stringWithUTF8String:copy];
        free(copy);

        NSStatusBarButton *button = getStatusItemButton();
        if (!button) return;

        if (!statusPopover) {
            popoverVC = [[PopoverViewController alloc] init];
            statusPopover = [[NSPopover alloc] init];
            statusPopover.contentViewController = popoverVC;
            statusPopover.contentSize = NSMakeSize(300, 52);
            statusPopover.behavior = NSPopoverBehaviorApplicationDefined;
            statusPopover.animates = YES;
        }

        // Ensure full opacity (may have been mid-fade)
        popoverVC.view.alphaValue = 1.0;

        if (!statusPopover.shown) {
            [statusPopover showRelativeToRect:button.bounds
                                       ofView:button
                                preferredEdge:NSMinYEdge];
        }

        // Split on first newline: title (project) + detail (tool)
        NSRange nl = [str rangeOfString:@"\n"];
        if (nl.location != NSNotFound) {
            [popoverVC.titleLabel setStringValue:[str substringToIndex:nl.location]];
            [popoverVC.detailLabel setStringValue:[str substringFromIndex:nl.location + 1]];
        } else {
            [popoverVC.titleLabel setStringValue:@""];
            [popoverVC.detailLabel setStringValue:str];
        }

        // Reset auto-dismiss timer
        if (popoverTimer) {
            [popoverTimer invalidate];
        }
        popoverTimer = [NSTimer scheduledTimerWithTimeInterval:kPopoverMinDuration
                                                       repeats:NO
                                                         block:^(NSTimer *timer) {
            fadeAndClosePopover();
            popoverTimer = nil;
        }];
    });
}

// hideBalloon fades out the popover. If the minimum display time hasn't
// elapsed yet, the timer handles the fade when it fires.
void hideBalloon(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (!statusPopover || !statusPopover.shown) return;

        // If a timer is still running, let it handle the fade.
        if (popoverTimer) return;

        fadeAndClosePopover();
    });
}

// showOverlay creates a borderless, click-through overlay window with an orange
// border on every active display. Each overlay auto-dismisses after 800ms.
void showOverlay(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSArray *overlays = createBorderOverlays([NSColor orangeColor]);
        for (NSWindow *w in overlays) [w orderFrontRegardless];

        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(800 * NSEC_PER_MSEC)),
                       dispatch_get_main_queue(), ^{
            for (NSWindow *w in overlays) [w orderOut:nil];
        });
    });
}
