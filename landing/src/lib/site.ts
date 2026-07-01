export const releaseUrl = "https://github.com/skulpturenz/timeboxxing/releases";

export const downloads = {
  macos: {
    label: "Download for macOS",
    platform: "macOS",
    href: releaseUrl,
    note: "DMG installer",
  },
  windows: {
    label: "Download for Windows",
    platform: "Windows",
    href: releaseUrl,
    note: "EXE installer",
  },
} as const;

export const signupEndpoint = import.meta.env.PUBLIC_SIGNUP_ENDPOINT ?? "";
