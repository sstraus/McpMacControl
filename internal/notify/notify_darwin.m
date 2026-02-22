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

// --- Balloon notification (floating HUD near menu bar) ---

static NSWindow *balloonWindow = nil;
static NSTextField *balloonLabel = nil;
static NSTimer *balloonTimer = nil;

// showBalloon displays a dark floating HUD just below the menu bar on the
// primary display. Auto-dismisses after 3 seconds. Calling again resets the
// timer and updates the text.
void showBalloon(const char* text) {
    // Copy the string before dispatch_async — the caller frees the original
    // immediately after this function returns (Go defer C.free).
    char *copy = strdup(text);
    dispatch_async(dispatch_get_main_queue(), ^{
        NSString *str = [NSString stringWithUTF8String:copy];
        free(copy);

        if (!balloonWindow) {
            NSScreen *screen = [NSScreen mainScreen];
            CGFloat screenWidth = screen.frame.size.width;
            CGFloat screenTop = NSMaxY(screen.visibleFrame);
            CGFloat bw = 320;
            CGFloat bh = 32;

            NSRect frame = NSMakeRect(
                screenWidth - bw - 8,   // right-aligned with margin
                screenTop - bh - 4,     // just below menu bar
                bw, bh
            );

            balloonWindow = [[NSWindow alloc]
                initWithContentRect:frame
                          styleMask:NSWindowStyleMaskBorderless
                            backing:NSBackingStoreBuffered
                              defer:NO];

            [balloonWindow setLevel:NSStatusWindowLevel + 1];
            [balloonWindow setOpaque:NO];
            [balloonWindow setBackgroundColor:[NSColor colorWithWhite:0.12 alpha:0.92]];
            [balloonWindow setIgnoresMouseEvents:YES];
            [balloonWindow setHasShadow:YES];
            [balloonWindow setCollectionBehavior:
                NSWindowCollectionBehaviorCanJoinAllSpaces |
                NSWindowCollectionBehaviorStationary];

            [balloonWindow.contentView setWantsLayer:YES];
            balloonWindow.contentView.layer.cornerRadius = 8.0;
            balloonWindow.contentView.layer.masksToBounds = YES;

            balloonLabel = [[NSTextField alloc] initWithFrame:NSMakeRect(10, 4, bw - 20, 24)];
            [balloonLabel setBezeled:NO];
            [balloonLabel setDrawsBackground:NO];
            [balloonLabel setEditable:NO];
            [balloonLabel setSelectable:NO];
            [balloonLabel setTextColor:[NSColor whiteColor]];
            [balloonLabel setFont:[NSFont monospacedSystemFontOfSize:12 weight:NSFontWeightMedium]];
            [balloonLabel setLineBreakMode:NSLineBreakByTruncatingTail];
            [balloonWindow.contentView addSubview:balloonLabel];
        }

        [balloonLabel setStringValue:str];
        [balloonWindow orderFrontRegardless];

        // Reset auto-dismiss timer (3 seconds)
        if (balloonTimer) {
            [balloonTimer invalidate];
        }
        balloonTimer = [NSTimer scheduledTimerWithTimeInterval:3.0
                                                       repeats:NO
                                                         block:^(NSTimer *timer) {
            [balloonWindow orderOut:nil];
            balloonTimer = nil;
        }];
    });
}

// hideBalloon hides the balloon immediately.
void hideBalloon(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (balloonTimer) {
            [balloonTimer invalidate];
            balloonTimer = nil;
        }
        if (balloonWindow) {
            [balloonWindow orderOut:nil];
        }
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
