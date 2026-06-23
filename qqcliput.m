#import <Foundation/Foundation.h>
#import <CoreGraphics/CoreGraphics.h>
#import <CoreImage/CoreImage.h>
#import <Vision/Vision.h>
#import <objc/message.h>
#import "qqcliput.h"

static uint32_t findQQWindow(void) {
    while (1) {
        @autoreleasepool {
            CFArrayRef windowList = CGWindowListCopyWindowInfo(
                kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
            if (!windowList) {
                [NSThread sleepForTimeInterval:5.0];
                continue;
            }

            uint32_t bestWID = 0;
            CFIndex count = CFArrayGetCount(windowList);

            for (CFIndex i = 0; i < count; i++) {
                NSDictionary *info = (__bridge NSDictionary *)CFArrayGetValueAtIndex(windowList, i);

                NSString *owner = info[(__bridge NSString *)kCGWindowOwnerName];
                if (![owner isEqualToString:@"QQ"]) continue;

                NSNumber *layer = info[(__bridge NSString *)kCGWindowLayer];
                if ([layer integerValue] != 0) continue;

                NSDictionary *boundsDict = info[(__bridge NSString *)kCGWindowBounds];
                CGRect bounds;
                CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)boundsDict, &bounds);
                if (bounds.size.width < 300 || bounds.size.height < 200) continue;

                NSNumber *widNum = info[(__bridge NSString *)kCGWindowNumber];
                bestWID = [widNum unsignedIntValue];
            }

            CFRelease(windowList);
            if (bestWID != 0) return bestWID;
        }
        [NSThread sleepForTimeInterval:5.0];
    }
}

uint32_t find_qq_window(void) {
    return findQQWindow();
}

int is_qq_frontmost(void) {
    @autoreleasepool {
        CFArrayRef windowList = CGWindowListCopyWindowInfo(
            kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
        if (!windowList) return 0;

        CFIndex count = CFArrayGetCount(windowList);
        for (CFIndex i = 0; i < count; i++) {
            NSDictionary *info = (__bridge NSDictionary *)CFArrayGetValueAtIndex(windowList, i);
            NSNumber *layer = info[(__bridge NSString *)kCGWindowLayer];
            if ([layer integerValue] != 0) continue;
            NSString *owner = info[(__bridge NSString *)kCGWindowOwnerName];
            if (owner && [owner isEqualToString:@"QQ"]) {
                CFRelease(windowList);
                return 1;
            }
        }

        CFRelease(windowList);
        return 0;
    }
}

int window_exists(uint32_t window_id) {
    @autoreleasepool {
        CFArrayRef windowList = CGWindowListCopyWindowInfo(
            kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
        if (!windowList) return 0;

        CFIndex count = CFArrayGetCount(windowList);
        int found = 0;

        for (CFIndex i = 0; i < count; i++) {
            NSDictionary *info = (__bridge NSDictionary *)CFArrayGetValueAtIndex(windowList, i);
            NSNumber *widNum = info[(__bridge NSString *)kCGWindowNumber];
            if ([widNum unsignedIntValue] == window_id) {
                found = 1;
                break;
            }
        }

        CFRelease(windowList);
        return found;
    }
}

static CGImageRef captureWindow(uint32_t window_id) {
    CGImageRef image = CGWindowListCreateImage(
        CGRectNull,
        kCGWindowListOptionIncludingWindow,
        (CGWindowID)window_id,
        kCGWindowImageBoundsIgnoreFraming);
    return image;
}

static NSString *ocrImage(CGImageRef image) {
    if (!image) return nil;

    CIImage *ciImage = [[CIImage alloc] initWithCGImage:image];

    VNImageRequestHandler *handler = [[VNImageRequestHandler alloc]
        initWithCIImage:ciImage options:@{}];

    VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] init];
    request.recognitionLevel = 0;
    request.recognitionLanguages = @[@"zh-Hans", @"zh-Hant", @"en-US"];
    request.usesLanguageCorrection = YES;
    SEL autoLangSel = NSSelectorFromString(@"setAutomaticallyDetectsLanguage:");
    if ([request respondsToSelector:autoLangSel]) {
        ((void (*)(id, SEL, BOOL))objc_msgSend)(request, autoLangSel, YES);
    }

    NSError *error = nil;
    [handler performRequests:@[request] error:&error];
    if (error) return nil;

    NSMutableArray *results = [NSMutableArray array];
    for (VNRecognizedTextObservation *observation in request.results) {
        VNRecognizedText *top = [observation topCandidates:1].firstObject;
        if (top.string) {
            [results addObject:top.string];
        }
    }

    if (results.count == 0) return nil;
    return [results componentsJoinedByString:@"\n"];
}

static NSString *ocrImageJSON(CGImageRef image) {
    if (!image) return nil;

    CIImage *ciImage = [[CIImage alloc] initWithCGImage:image];

    VNImageRequestHandler *handler = [[VNImageRequestHandler alloc]
        initWithCIImage:ciImage options:@{}];

    VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] init];
    request.recognitionLevel = 0;
    request.recognitionLanguages = @[@"zh-Hans", @"zh-Hant", @"en-US"];
    request.usesLanguageCorrection = YES;
    SEL autoLangSel = NSSelectorFromString(@"setAutomaticallyDetectsLanguage:");
    if ([request respondsToSelector:autoLangSel]) {
        ((void (*)(id, SEL, BOOL))objc_msgSend)(request, autoLangSel, YES);
    }

    NSError *error = nil;
    [handler performRequests:@[request] error:&error];
    if (error) return @"[]";

    NSMutableArray *jsonArr = [NSMutableArray array];
    for (VNRecognizedTextObservation *obs in request.results) {
        CGRect box = obs.boundingBox;
        VNRecognizedText *top = [obs topCandidates:1].firstObject;
        if (!top.string || top.string.length == 0) continue;
        NSDictionary *item = @{
            @"text": top.string,
            @"x": @(box.origin.x),
            @"y": @(1.0 - box.origin.y),
            @"w": @(box.size.width),
            @"h": @(box.size.height),
        };
        [jsonArr addObject:item];
    }

    if (jsonArr.count == 0) return @"[]";
    NSError *jsonErr = nil;
    NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonArr options:0 error:&jsonErr];
    if (jsonErr) return @"[]";
    return [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
}

char *ocr_window_json(uint32_t window_id) {
    @autoreleasepool {
        CGImageRef image = captureWindow(window_id);
        if (!image) return strdup("[]");

        NSString *json = ocrImageJSON(image);
        CGImageRelease(image);

        if (!json) return strdup("[]");
        return strdup([json UTF8String]);
    }
}

char *ocr_window(uint32_t window_id) {
    @autoreleasepool {
        CGImageRef image = captureWindow(window_id);
        if (!image) return strdup("");

        NSString *text = ocrImage(image);
        CGImageRelease(image);

        if (!text || text.length == 0) return strdup("");
        return strdup([text UTF8String]);
    }
}

char *get_window_bounds(uint32_t window_id) {
    @autoreleasepool {
        CFArrayRef windows = CGWindowListCopyWindowInfo(
            kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
        if (!windows) return strdup("{}");

        CFIndex count = CFArrayGetCount(windows);
        for (CFIndex i = 0; i < count; i++) {
            NSDictionary *info = (__bridge NSDictionary *)CFArrayGetValueAtIndex(windows, i);
            NSNumber *widNum = info[(__bridge NSString *)kCGWindowNumber];
            if ([widNum unsignedIntValue] == window_id) {
                NSDictionary *boundsDict = info[(__bridge NSString *)kCGWindowBounds];
                CGRect bounds;
                CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)boundsDict, &bounds);
                CFRelease(windows);
                NSString *json = [NSString stringWithFormat:@"{\"w\":%.0f,\"h\":%.0f}",
                    bounds.size.width, bounds.size.height];
                return strdup([json UTF8String]);
            }
        }

        CFRelease(windows);
        return strdup("{}");
    }
}

static NSString *ocrImageRegionJSON(CGImageRef fullImage, CGFloat nx, CGFloat ny, CGFloat nw, CGFloat nh) {
    CGFloat fullW = CGImageGetWidth(fullImage);
    CGFloat fullH = CGImageGetHeight(fullImage);

    CGRect cropRect = CGRectMake(nx * fullW, ny * fullH, nw * fullW, nh * fullH);
    CGImageRef cropped = CGImageCreateWithImageInRect(fullImage, cropRect);
    if (!cropped) return @"[]";

    CIImage *ciImage = [[CIImage alloc] initWithCGImage:cropped];
    VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithCIImage:ciImage options:@{}];

    VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] init];
    request.recognitionLevel = 0;
    request.recognitionLanguages = @[@"zh-Hans", @"zh-Hant", @"en-US"];
    request.usesLanguageCorrection = YES;
    SEL autoLangSel = NSSelectorFromString(@"setAutomaticallyDetectsLanguage:");
    if ([request respondsToSelector:autoLangSel]) {
        ((void (*)(id, SEL, BOOL))objc_msgSend)(request, autoLangSel, YES);
    }

    NSError *error = nil;
    [handler performRequests:@[request] error:&error];
    if (error) { CGImageRelease(cropped); return @"[]"; }

    NSMutableArray *jsonArr = [NSMutableArray array];
    for (VNRecognizedTextObservation *obs in request.results) {
        CGRect box = obs.boundingBox;
        VNRecognizedText *top = [obs topCandidates:1].firstObject;
        if (!top.string || top.string.length == 0) continue;

        double bx = box.origin.x;
        double by = 1.0 - box.origin.y;

        NSDictionary *item = @{
            @"text": top.string,
            @"x": @(nx + bx * nw),
            @"y": @(ny + by * nh),
            @"w": @(box.size.width * nw),
            @"h": @(box.size.height * nh),
        };
        [jsonArr addObject:item];
    }

    CGImageRelease(cropped);

    if (jsonArr.count == 0) return @"[]";
    NSError *jsonErr = nil;
    NSData *jsonData = [NSJSONSerialization dataWithJSONObject:jsonArr options:0 error:&jsonErr];
    if (jsonErr) return @"[]";
    return [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
}

char *ocr_window_region_json(uint32_t window_id, double nx, double ny, double nw, double nh) {
    @autoreleasepool {
        CGImageRef image = captureWindow(window_id);
        if (!image) return strdup("[]");

        NSString *json = ocrImageRegionJSON(image, (CGFloat)nx, (CGFloat)ny, (CGFloat)nw, (CGFloat)nh);
        CGImageRelease(image);

        if (!json) return strdup("[]");
        return strdup([json UTF8String]);
    }
}
