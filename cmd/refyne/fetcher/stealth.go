package fetcher

import (
	"context"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// StealthScript contains JavaScript to evade common bot detection techniques.
// Based on puppeteer-extra-plugin-stealth evasions.
const StealthScript = `
(function() {
    'use strict';

    // 1. Remove navigator.webdriver
    // Chrome 89+ already sets this to false/undefined in some cases, but we ensure it
    Object.defineProperty(navigator, 'webdriver', {
        get: () => undefined,
        configurable: true
    });
    // Also delete from prototype for older detection methods
    delete Object.getPrototypeOf(navigator).webdriver;

    // 2. Mock navigator.plugins with realistic values
    // Headless Chrome has an empty plugins array which is a dead giveaway
    const mockPlugins = [
        {
            name: 'Chrome PDF Plugin',
            description: 'Portable Document Format',
            filename: 'internal-pdf-viewer',
            length: 1
        },
        {
            name: 'Chrome PDF Viewer',
            description: '',
            filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
            length: 1
        },
        {
            name: 'Native Client',
            description: '',
            filename: 'internal-nacl-plugin',
            length: 2
        }
    ];

    const pluginArray = Object.create(PluginArray.prototype);
    mockPlugins.forEach((p, i) => {
        const plugin = Object.create(Plugin.prototype);
        Object.defineProperties(plugin, {
            name: { value: p.name, enumerable: true },
            description: { value: p.description, enumerable: true },
            filename: { value: p.filename, enumerable: true },
            length: { value: p.length, enumerable: true }
        });
        pluginArray[i] = plugin;
        pluginArray[p.name] = plugin;
    });
    Object.defineProperty(pluginArray, 'length', { value: mockPlugins.length });
    Object.defineProperty(pluginArray, 'item', { value: (i) => pluginArray[i] || null });
    Object.defineProperty(pluginArray, 'namedItem', { value: (n) => pluginArray[n] || null });
    Object.defineProperty(pluginArray, 'refresh', { value: () => {} });

    Object.defineProperty(navigator, 'plugins', {
        get: () => pluginArray,
        configurable: true
    });

    // 3. Mock navigator.mimeTypes
    const mockMimeTypes = [
        { type: 'application/pdf', description: 'Portable Document Format', suffixes: 'pdf' },
        { type: 'text/pdf', description: 'Portable Document Format', suffixes: 'pdf' }
    ];

    const mimeTypeArray = Object.create(MimeTypeArray.prototype);
    mockMimeTypes.forEach((m, i) => {
        const mimeType = Object.create(MimeType.prototype);
        Object.defineProperties(mimeType, {
            type: { value: m.type, enumerable: true },
            description: { value: m.description, enumerable: true },
            suffixes: { value: m.suffixes, enumerable: true },
            enabledPlugin: { value: pluginArray[0], enumerable: true }
        });
        mimeTypeArray[i] = mimeType;
        mimeTypeArray[m.type] = mimeType;
    });
    Object.defineProperty(mimeTypeArray, 'length', { value: mockMimeTypes.length });
    Object.defineProperty(mimeTypeArray, 'item', { value: (i) => mimeTypeArray[i] || null });
    Object.defineProperty(mimeTypeArray, 'namedItem', { value: (n) => mimeTypeArray[n] || null });

    Object.defineProperty(navigator, 'mimeTypes', {
        get: () => mimeTypeArray,
        configurable: true
    });

    // 4. Set navigator.languages
    Object.defineProperty(navigator, 'languages', {
        get: () => Object.freeze(['en-US', 'en']),
        configurable: true
    });

    // 5. Mock chrome.runtime
    // Headless Chrome doesn't have window.chrome in some contexts
    if (!window.chrome) {
        Object.defineProperty(window, 'chrome', {
            value: {},
            writable: true,
            enumerable: true,
            configurable: false
        });
    }

    if (!window.chrome.runtime) {
        window.chrome.runtime = {
            OnInstalledReason: {
                CHROME_UPDATE: 'chrome_update',
                INSTALL: 'install',
                SHARED_MODULE_UPDATE: 'shared_module_update',
                UPDATE: 'update'
            },
            OnRestartRequiredReason: {
                APP_UPDATE: 'app_update',
                OS_UPDATE: 'os_update',
                PERIODIC: 'periodic'
            },
            PlatformArch: {
                ARM: 'arm',
                ARM64: 'arm64',
                MIPS: 'mips',
                MIPS64: 'mips64',
                X86_32: 'x86-32',
                X86_64: 'x86-64'
            },
            PlatformNaclArch: {
                ARM: 'arm',
                MIPS: 'mips',
                MIPS64: 'mips64',
                X86_32: 'x86-32',
                X86_64: 'x86-64'
            },
            PlatformOs: {
                ANDROID: 'android',
                CROS: 'cros',
                LINUX: 'linux',
                MAC: 'mac',
                OPENBSD: 'openbsd',
                WIN: 'win'
            },
            RequestUpdateCheckStatus: {
                NO_UPDATE: 'no_update',
                THROTTLED: 'throttled',
                UPDATE_AVAILABLE: 'update_available'
            },
            get id() { return undefined; },
            connect: function() {},
            sendMessage: function() {}
        };
    }

    // 6. Fix permissions query for notifications
    const originalQuery = Permissions.prototype.query;
    Permissions.prototype.query = function(parameters) {
        if (parameters.name === 'notifications') {
            return Promise.resolve({ state: Notification.permission });
        }
        return originalQuery.call(this, parameters);
    };

    // 7. Override WebGL vendor and renderer
    const getParameterProxyHandler = {
        apply: function(target, ctx, args) {
            const param = args[0];
            const result = Reflect.apply(target, ctx, args);
            // UNMASKED_VENDOR_WEBGL
            if (param === 37445) {
                return 'Intel Inc.';
            }
            // UNMASKED_RENDERER_WEBGL
            if (param === 37446) {
                return 'Intel Iris OpenGL Engine';
            }
            return result;
        }
    };

    // Patch WebGL contexts
    try {
        const webglGetParameter = WebGLRenderingContext.prototype.getParameter;
        WebGLRenderingContext.prototype.getParameter = new Proxy(webglGetParameter, getParameterProxyHandler);
    } catch (e) {}

    try {
        const webgl2GetParameter = WebGL2RenderingContext.prototype.getParameter;
        WebGL2RenderingContext.prototype.getParameter = new Proxy(webgl2GetParameter, getParameterProxyHandler);
    } catch (e) {}

    // 8. Fix iframe contentWindow access
    try {
        Object.defineProperty(HTMLIFrameElement.prototype, 'contentWindow', {
            get: function() {
                return this.contentDocument?.defaultView || null;
            }
        });
    } catch (e) {}

    // 9. Make toString() for native functions look native
    const nativeToStringFunc = Function.prototype.toString;
    const customToString = function() {
        if (this === Permissions.prototype.query) {
            return 'function query() { [native code] }';
        }
        return nativeToStringFunc.call(this);
    };
    Function.prototype.toString = customToString;

    // 10. Override navigator.hardwareConcurrency if it's 0 (suspicious in headless)
    if (navigator.hardwareConcurrency === 0) {
        Object.defineProperty(navigator, 'hardwareConcurrency', {
            get: () => 4,
            configurable: true
        });
    }

    // 11. Override navigator.deviceMemory if missing
    if (navigator.deviceMemory === undefined || navigator.deviceMemory === 0) {
        Object.defineProperty(navigator, 'deviceMemory', {
            get: () => 8,
            configurable: true
        });
    }

    console.debug('[stealth] Anti-detection patches applied');
})();
`

// StealthExecAllocatorOptions returns Chrome flags optimized for stealth.
// These should be used when creating the browser allocator.
func StealthExecAllocatorOptions() []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		// Basic headless setup
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),

		// Anti-detection flags
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
		chromedp.Flag("useAutomationExtension", false),

		// Disable infobars and other automation indicators
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-extensions", false), // Some sites check for extension support
		chromedp.Flag("disable-plugins-discovery", true),
		chromedp.Flag("disable-default-apps", true),

		// Realistic browser behavior
		chromedp.Flag("disable-background-networking", false),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),

		// Media and WebRTC
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		chromedp.Flag("use-fake-device-for-media-stream", true),

		// Window size to look realistic
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("start-maximized", true),

		// Language settings
		chromedp.Flag("lang", "en-US,en"),
		chromedp.Flag("accept-lang", "en-US,en;q=0.9"),

		// Ignore certificate errors (useful for some sites)
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("allow-running-insecure-content", true),
	}
}

// InjectStealthScript returns a chromedp.Action that injects the stealth script
// before any page scripts run. This should be called before navigation.
func InjectStealthScript() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(StealthScript).Do(ctx)
		return err
	})
}

// CaptureScreenshotOnError captures a screenshot for debugging purposes.
// Returns the screenshot as bytes, or nil if capture fails.
func CaptureScreenshotOnError(ctx context.Context) []byte {
	var screenshot []byte
	// Use a short timeout for screenshot capture (5 seconds)
	captureCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := chromedp.Run(captureCtx, chromedp.CaptureScreenshot(&screenshot)); err != nil {
		// Screenshot capture failed - browser might be in a bad state
		return nil
	}
	return screenshot
}
