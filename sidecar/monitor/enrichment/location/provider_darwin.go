//go:build darwin

package location

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreLocation

#import <Foundation/Foundation.h>
#import <CoreLocation/CoreLocation.h>
#import <os/lock.h>

// CoreLocation delivers delegate callbacks on the run loop of the thread that
// created the manager, so it runs on a dedicated thread with its own CFRunLoop.
// The latest fix is cached under a lock; the Go reader never blocks on a fix.
//
// Authorization requires a bundled app with NSLocationWhenInUseUsageDescription;
// unbundled/CLI runs simply never get a fix, leaving the cache empty.
//
// All globals are `static` (internal linkage) and the exported symbols are
// uniquely prefixed so this coexists with sidecar/monitor/platform/darwin.go's
// own CoreLocation provider in the same binary without symbol collisions.

static os_unfair_lock g_locLock = OS_UNFAIR_LOCK_INIT;
static double g_lat = 0;
static double g_lon = 0;
static int g_hasLocation = 0;

// Kept in statics so ARC does not release them (the manager holds its delegate
// weakly).
static CLLocationManager *g_locationManager = nil;

@interface TBXEnrichLocationDelegate : NSObject <CLLocationManagerDelegate>
@end

@implementation TBXEnrichLocationDelegate
- (void)locationManager:(CLLocationManager *)manager
     didUpdateLocations:(NSArray<CLLocation *> *)locations {
    CLLocation *loc = [locations lastObject];
    if (!loc) return;
    os_unfair_lock_lock(&g_locLock);
    g_lat = loc.coordinate.latitude;
    g_lon = loc.coordinate.longitude;
    g_hasLocation = 1;
    os_unfair_lock_unlock(&g_locLock);
}
- (void)locationManager:(CLLocationManager *)manager
       didFailWithError:(NSError *)error {
    // Keep the previous cached value on transient failures.
}
@end

static TBXEnrichLocationDelegate *g_locationDelegate = nil;

// tbxStartLocationUpdates spins up the CLLocationManager once, on a dedicated
// run-loop thread. Safe to call repeatedly.
void tbxStartLocationUpdates(void) {
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        NSThread *thread = [[NSThread alloc] initWithBlock:^{
            @autoreleasepool {
                g_locationDelegate = [[TBXEnrichLocationDelegate alloc] init];
                g_locationManager = [[CLLocationManager alloc] init];
                g_locationManager.delegate = g_locationDelegate;
                g_locationManager.desiredAccuracy = kCLLocationAccuracyKilometer;
                [g_locationManager startUpdatingLocation];
                CFRunLoopRun();
            }
        }];
        [thread start];
    });
}

// tbxRequestLocationAuthorization raises the macOS location-permission prompt.
// It is separate from tbxStartLocationUpdates so the prompt fires only when the
// caller explicitly requests it. Safe to call before or after startup; a no-op
// until the manager exists.
void tbxRequestLocationAuthorization(void) {
    if (g_locationManager != nil &&
        [g_locationManager respondsToSelector:@selector(requestWhenInUseAuthorization)]) {
        [g_locationManager requestWhenInUseAuthorization];
    }
}

// tbxGetCachedLocation writes the latest fix into lat/lon and returns 1 when a
// fix is available, 0 otherwise. Never blocks on the network/GPS.
int tbxGetCachedLocation(double *lat, double *lon) {
    int has = 0;
    os_unfair_lock_lock(&g_locLock);
    if (g_hasLocation) {
        *lat = g_lat;
        *lon = g_lon;
        has = 1;
    }
    os_unfair_lock_unlock(&g_locLock);
    return has;
}

// tbxLocationAuthStatus returns the raw CLAuthorizationStatus (notDetermined=0,
// restricted=1, denied=2, authorizedAlways=3, authorizedWhenInUse=4).
int tbxLocationAuthStatus(void) {
    if (g_locationManager != nil) {
        if (@available(macOS 11.0, *)) {
            return (int)g_locationManager.authorizationStatus;
        }
    }
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
    return (int)[CLLocationManager authorizationStatus];
#pragma clang diagnostic pop
}
*/
import "C"

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// DefaultLocationProvider starts the CoreLocation background provider (once) and
// returns a provider that reads its cache non-blockingly.
func DefaultLocationProvider() LocationProvider {
	C.tbxStartLocationUpdates()
	return coreLocationProvider{}
}

type coreLocationProvider struct{}

func (coreLocationProvider) Location() (float64, float64, bool) {
	var lat, lon C.double
	if C.tbxGetCachedLocation(&lat, &lon) == 1 {
		return float64(lat), float64(lon), true
	}
	return 0, 0, false
}

func (coreLocationProvider) Permission() (permission.Status, bool) {
	// CLAuthorizationStatus: authorizedAlways=3, authorizedWhenInUse=4.
	status := int(C.tbxLocationAuthStatus())
	return permission.Status{
		Name:       "Location Services",
		Granted:    status == 3 || status == 4,
		HowToGrant: "System Settings → Privacy & Security → Location Services → enable this app",
	}, true
}

// RequestPermission raises the macOS location-authorization prompt. Authorization
// only actually succeeds for a bundled app with NSLocationWhenInUseUsageDescription;
// unbundled/CLI runs get no fix and the cache stays empty.
func (coreLocationProvider) RequestPermission(context.Context) {
	C.tbxRequestLocationAuthorization()
}
