package com.timeboxxing.app

import androidx.compose.ui.graphics.Color
import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaAppUsageBucket
import com.timeboxxing.domain.model.AmaAppUsageChart
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaMessageRole
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntryMode
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingSection
import com.timeboxxing.app.presentation.createInitialTimeboxxingState
import com.timeboxxing.app.presentation.reduceTimeboxxingState
import com.timeboxxing.app.ui.buildTimelineGrid
import com.timeboxxing.app.ui.timelineFirstEventScrollDp
import com.timeboxxing.app.ui.timelineDpPerMinuteForZoom
import com.timeboxxing.app.ui.timelineEntryScrollDp
import com.timeboxxing.app.ui.timelineMinuteOffsetDp
import com.timeboxxing.app.ui.timelineNowScrollDirection
import com.timeboxxing.app.ui.timelineRowHeightDpForZoom
import com.timeboxxing.app.ui.timelineScrollToMinuteDp
import com.timeboxxing.app.ui.timelineSegmentsForRow
import com.timeboxxing.app.ui.timelineIntervalMarkerOffsets
import com.timeboxxing.app.ui.SchedulePaneAutoScrollState
import com.timeboxxing.app.ui.TimelineScrollDirection
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.TbLightColors
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

class ThemeContrastTest {

    @Test
    fun lightAndDarkPalettesUseDistinctSurfaces() {
        assertEquals(Color.White, TbLightColors.appBackground)
        assertEquals(Color.Black, TbDarkColors.appBackground)
        assertEquals(Color(0xFF00FFEE), TbLightColors.accent)
        assertEquals(Color(0xFF00FFEE), TbDarkColors.accent)
        assertTrue(TbLightColors.appBackground != TbDarkColors.appBackground)
        assertTrue(TbLightColors.surface != TbDarkColors.surface)
        assertTrue(TbLightColors.elevatedSurface != TbDarkColors.elevatedSurface)
        assertTrue(TbLightColors.text != TbDarkColors.text)
    }

    @Test
    fun darkPaletteTextContrastsAgainstSurfaces() {
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.surface)
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.groupedSurface)
        assertContrastAtLeast(7.0, TbDarkColors.text, TbDarkColors.elevatedSurface)
        assertContrastAtLeast(4.5, TbDarkColors.secondaryText, TbDarkColors.surface)
    }

    @Test
    fun accentTextContrastsAgainstAccentBackgrounds() {
        assertContrastAtLeast(4.5, TbLightColors.accentText, TbLightColors.accent)
        assertContrastAtLeast(4.5, TbDarkColors.accentText, TbDarkColors.accent)
    }

    private fun assertContrastAtLeast(
        minimum: Double,
        foreground: Color,
        background: Color,
    ) {
        val actual = contrastRatio(foreground, background)
        assertTrue(
            actual >= minimum,
            "Expected contrast ratio $actual to be at least $minimum",
        )
    }

    private fun contrastRatio(
        foreground: Color,
        background: Color,
    ): Double {
        val foregroundLuminance = relativeLuminance(foreground)
        val backgroundLuminance = relativeLuminance(background)
        return (max(foregroundLuminance, backgroundLuminance) + 0.05) /
            (min(foregroundLuminance, backgroundLuminance) + 0.05)
    }

    private fun relativeLuminance(color: Color): Double =
        0.2126 * linearizedColorComponent(color.red) +
            0.7152 * linearizedColorComponent(color.green) +
            0.0722 * linearizedColorComponent(color.blue)

    private fun linearizedColorComponent(component: Float): Double {
        val value = component.toDouble()
        return if (value <= 0.03928) {
            value / 12.92
        } else {
            ((value + 0.055) / 1.055).pow(2.4)
        }
    }
}
