package com.timeboxxing.app

import androidx.compose.ui.awt.ComposeWindow
import androidx.compose.ui.graphics.Color
import com.sun.jna.Library
import com.sun.jna.Memory
import com.sun.jna.Native
import com.sun.jna.Pointer

// Windows title-bar theming via the Desktop Window Manager (DWM). The macOS title bar is
// themed through apple.awt.* rootPane properties in main.kt; this is the Windows counterpart.
private interface Dwmapi : Library {
    fun DwmSetWindowAttribute(
        hwnd: Pointer,
        dwAttribute: Int,
        pvAttribute: Pointer,
        cbAttribute: Int,
    ): Int

    companion object {
        val INSTANCE: Dwmapi by lazy { Native.load("dwmapi", Dwmapi::class.java) }
    }
}

// Win10 1809+: toggles the caption between dark and light.
private const val DWMWA_USE_IMMERSIVE_DARK_MODE = 20

// Win11 22000+: sets an exact caption color. No-op (non-zero HRESULT) on older Windows.
private const val DWMWA_CAPTION_COLOR = 35

/**
 * Colors the native Windows title bar to match [backgroundColor]/[darkTheme]. Best-effort:
 * the exact caption color only applies on Windows 11; on Windows 10 the immersive dark/light
 * toggle still gives a themed caption. Any failure (missing dwmapi, non-displayable window,
 * older OS) is swallowed so it can never break startup. Safe to call repeatedly — it is
 * re-invoked whenever the app theme changes.
 */
internal fun ComposeWindow.applyWindowsTitleBar(backgroundColor: Color, darkTheme: Boolean) {
    runCatching {
        // Null until the window is displayable; the driving LaunchedEffect re-runs on theme
        // changes, so the caption gets themed once the handle is available.
        val hwnd = Native.getWindowPointer(this) ?: return
        val dwm = Dwmapi.INSTANCE

        Memory(4).use { dark ->
            dark.setInt(0, if (darkTheme) 1 else 0)
            dwm.DwmSetWindowAttribute(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, dark, 4)
        }

        Memory(4).use { caption ->
            caption.setInt(0, backgroundColor.toColorRef())
            dwm.DwmSetWindowAttribute(hwnd, DWMWA_CAPTION_COLOR, caption, 4)
        }
    }
}

// Windows COLORREF is 0x00BBGGRR.
private fun Color.toColorRef(): Int {
    val r = (red * 255f).toInt() and 0xFF
    val g = (green * 255f).toInt() and 0xFF
    val b = (blue * 255f).toInt() and 0xFF
    return (b shl 16) or (g shl 8) or r
}
