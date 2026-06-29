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

class FormatterTest {

    @Test
    fun formatsDurationAndClockTimes() {
        assertEquals("45m", formatDuration(45))
        assertEquals("2h 5m", formatDuration(125))
        assertEquals("12:00 PM", formatClockTime(12 * 60))
        assertEquals("2:05 PM", formatClockTime(14 * 60 + 5))
    }
}
