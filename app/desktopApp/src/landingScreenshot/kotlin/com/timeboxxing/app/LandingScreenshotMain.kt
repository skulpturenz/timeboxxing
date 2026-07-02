package com.timeboxxing.app

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.awt.ComposeWindow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application
import androidx.compose.ui.window.rememberWindowState
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.createInitialTimeboxxingState
import com.timeboxxing.app.ui.TbDarkColors
import com.timeboxxing.app.ui.fonts.InterFontLoader
import com.timeboxxing.data.mock.mockTimeboxxingData
import com.timeboxxing.domain.model.AppearanceMode
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntryDraft
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimeboxxingMockData
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import java.awt.Dimension
import java.awt.Color as AwtColor

private const val LandingScreenshotTitle = "Timeboxxing Landing Capture"
private const val WindowWidth = 1280
private const val WindowHeight = 860

fun main() {
    configureLandingScreenshotSystemProperties()

    application {
        val windowState = rememberWindowState(width = WindowWidth.dp, height = WindowHeight.dp)
        Window(
            onCloseRequest = ::exitApplication,
            title = LandingScreenshotTitle,
            state = windowState,
        ) {
            val background = TbDarkColors.appBackground
            var fontFamily by remember { mutableStateOf<FontFamily?>(null) }
            LaunchedEffect(Unit) {
                window.minimumSize = Dimension(WindowWidth, WindowHeight)
                window.applyLandingScreenshotChrome(background)
                fontFamily = InterFontLoader.loadFontFamily()
            }

            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(background),
            ) {
                TimeboxxingApp(
                    state = landingScreenshotState(),
                    onAction = { _: TimeboxxingAction -> },
                    fontFamily = fontFamily,
                    darkTheme = true,
                )
            }
        }
    }
}

private fun landingScreenshotState() =
    createInitialTimeboxxingState(
        data = landingScreenshotData(),
        appearanceMode = AppearanceMode.Dark,
    ).copy(
        selectedUsageIds = setOf("usage-proposal", "usage-research"),
        draft = EntryDraft(
            projectId = "client-launch",
            title = "Launch analytics review",
            notes = "Captured from docs, browser research, and client follow-up.",
            startMinute = 12 * 60,
            durationMinutes = 95,
            billable = true,
        ),
    )

private fun landingScreenshotData(): TimeboxxingMockData {
    val base = mockTimeboxxingData()
    val projects = listOf(
        Project(
            id = "client-launch",
            name = "Client launch",
            client = "Morgan Ltd.",
            colorArgb = 0xFF00FFEE,
            hourlyRateCents = 18_000,
        ),
        Project(
            id = "strategy-retainer",
            name = "Strategy retainer",
            client = "Daven Studio",
            colorArgb = 0xFF4F7CFF,
            hourlyRateCents = 16_500,
        ),
        Project(
            id = "operations",
            name = "Operations",
            client = "Internal",
            colorArgb = 0xFFFFB020,
            hourlyRateCents = 0,
        ),
    )
    val usageDay = UsageDay(
        label = "Thursday, May 1, 2025",
        startedAtEpochMillis = 1_746_086_400_000,
        endedAtEpochMillis = 1_746_172_800_000,
        calendarDate = CalendarDate(2025, 5, 1),
    )
    val usageEvents = listOf(
        UsageEvent(
            id = "usage-proposal",
            title = "Launch proposal final pass",
            sourceName = "Writer",
            sourceType = UsageSourceType.Document,
            startMinute = 9 * 60 + 10,
            durationMinutes = 35,
            projectHintId = "client-launch",
        ),
        UsageEvent(
            id = "usage-email",
            title = "Client follow-up thread",
            sourceName = "Mail",
            sourceType = UsageSourceType.Email,
            startMinute = 9 * 60 + 45,
            durationMinutes = 20,
            projectHintId = "client-launch",
        ),
        UsageEvent(
            id = "usage-meeting",
            title = "Launch checkpoint call",
            sourceName = "Meet",
            sourceType = UsageSourceType.Meeting,
            startMinute = 10 * 60 + 20,
            durationMinutes = 45,
            projectHintId = "client-launch",
        ),
        UsageEvent(
            id = "usage-model",
            title = "Forecast model cleanup",
            sourceName = "Sheets",
            sourceType = UsageSourceType.Spreadsheet,
            startMinute = 11 * 60 + 25,
            durationMinutes = 30,
            projectHintId = "strategy-retainer",
        ),
        UsageEvent(
            id = "usage-chat",
            title = "Team planning notes",
            sourceName = "Chat",
            sourceType = UsageSourceType.Messaging,
            startMinute = 12 * 60 + 10,
            durationMinutes = 18,
            projectHintId = "strategy-retainer",
        ),
        UsageEvent(
            id = "usage-deck",
            title = "Roadmap deck edits",
            sourceName = "Slides",
            sourceType = UsageSourceType.Presentation,
            startMinute = 13 * 60 + 5,
            durationMinutes = 42,
            projectHintId = "strategy-retainer",
        ),
        UsageEvent(
            id = "usage-research",
            title = "Competitive research pass",
            sourceName = "Browser",
            sourceType = UsageSourceType.Browser,
            startMinute = 14 * 60 + 5,
            durationMinutes = 55,
            projectHintId = "client-launch",
        ),
    )
    val entries = listOf(
        TimeEntry(
            id = "entry-client-launch",
            projectId = "client-launch",
            title = "Launch review and client follow-up",
            notes = "Merged proposal, email, and research activity.",
            startMinute = 9 * 60 + 10,
            durationMinutes = 100,
            billable = true,
            sourceUsageIds = setOf("usage-proposal", "usage-email", "usage-research"),
        ),
        TimeEntry(
            id = "entry-strategy",
            projectId = "strategy-retainer",
            title = "Strategy model and roadmap deck",
            notes = "Forecast updates and deck edits for weekly planning.",
            startMinute = 11 * 60 + 25,
            durationMinutes = 72,
            billable = true,
            sourceUsageIds = setOf("usage-model", "usage-deck"),
        ),
        TimeEntry(
            id = "entry-ops",
            projectId = "operations",
            title = "Inbox triage and planning",
            notes = "Non-billable operational cleanup.",
            startMinute = 15 * 60 + 20,
            durationMinutes = 28,
            billable = false,
            sourceUsageIds = setOf("usage-chat"),
        ),
    )

    return base.copy(
        usageDays = listOf(usageDay),
        projects = projects,
        usageEvents = usageEvents,
        initialEntries = entries,
    )
}

private fun configureLandingScreenshotSystemProperties() {
    if (!isMacOs()) return

    System.setProperty("apple.awt.application.appearance", "system")
    System.setProperty("apple.awt.application.name", LandingScreenshotTitle)
    System.setProperty("apple.laf.useScreenMenuBar", "true")
    System.setProperty("com.apple.mrj.application.apple.menu.about.name", LandingScreenshotTitle)
}

private fun ComposeWindow.applyLandingScreenshotChrome(backgroundColor: Color) {
    val awtBackground = backgroundColor.toAwtColor()
    background = awtBackground
    contentPane.background = awtBackground
    rootPane.background = awtBackground

    if (!isMacOs()) return

    rootPane.putClientProperty("apple.awt.fullWindowContent", true)
    rootPane.putClientProperty("apple.awt.transparentTitleBar", true)
    rootPane.putClientProperty("apple.awt.windowTitleVisible", false)
    rootPane.putClientProperty("apple.awt.windowAppearance", "NSAppearanceNameDarkAqua")
}

private fun isMacOs(): Boolean =
    System.getProperty("os.name").startsWith("Mac", ignoreCase = true)

private fun Color.toAwtColor(): AwtColor =
    AwtColor(red, green, blue, alpha)
