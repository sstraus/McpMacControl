#import <Cocoa/Cocoa.h>
#include <CoreGraphics/CoreGraphics.h>

// playTinkSound plays the system "Tink" sound asynchronously on the main thread.
void playTinkSound(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSSound *sound = [NSSound soundNamed:@"Tink"];
        [sound play];
    });
}

// showOverlay creates a borderless, click-through overlay window with an orange
// border on every active display. Each overlay auto-dismisses after 400ms.
void showOverlay(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        uint32_t displayCount = 0;
        CGGetActiveDisplayList(0, NULL, &displayCount);
        if (displayCount == 0) return;

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

            // Content view with orange border via CALayer
            NSView *contentView = [overlay contentView];
            [contentView setWantsLayer:YES];
            CALayer *layer = [contentView layer];
            layer.borderColor = [[NSColor orangeColor] CGColor];
            layer.borderWidth = 8.0;
            layer.cornerRadius = 0.0;

            [overlay orderFrontRegardless];

            // Auto-dismiss after 800ms
            dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(800 * NSEC_PER_MSEC)),
                           dispatch_get_main_queue(), ^{
                [overlay orderOut:nil];
            });
        }
    });
}
