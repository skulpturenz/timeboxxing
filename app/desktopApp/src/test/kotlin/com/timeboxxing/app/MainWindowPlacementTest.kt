package com.timeboxxing.app

import androidx.compose.ui.window.WindowPlacement
import kotlin.test.Test
import kotlin.test.assertEquals

class MainWindowPlacementTest {
    @Test
    fun titleBarDoubleClickMaximizesFloatingWindow() {
        assertEquals(
            WindowPlacement.Maximized,
            nextTitleBarDoubleClickPlacement(WindowPlacement.Floating),
        )
    }

    @Test
    fun titleBarDoubleClickRestoresMaximizedWindow() {
        assertEquals(
            WindowPlacement.Floating,
            nextTitleBarDoubleClickPlacement(WindowPlacement.Maximized),
        )
    }

    @Test
    fun titleBarDoubleClickMaximizesFullscreenWindow() {
        assertEquals(
            WindowPlacement.Maximized,
            nextTitleBarDoubleClickPlacement(WindowPlacement.Fullscreen),
        )
    }

    @Test
    fun macOsWindowAppearanceFollowsAppTheme() {
        assertEquals("NSAppearanceNameDarkAqua", macOsWindowAppearanceName(darkTheme = true))
        assertEquals("NSAppearanceNameAqua", macOsWindowAppearanceName(darkTheme = false))
    }
}
