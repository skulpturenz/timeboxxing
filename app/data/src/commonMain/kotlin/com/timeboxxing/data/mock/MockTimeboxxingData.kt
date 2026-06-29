package com.timeboxxing.data.mock

import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.TimeboxxingMockData
import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType

fun mockTimeboxxingData(): TimeboxxingMockData {
    val usageDays = listOf(
        UsageDay(
            label = "Wednesday, April 30, 2025",
            startedAtEpochMillis = 1_746_000_000_000,
            endedAtEpochMillis = 1_746_086_400_000,
            calendarDate = CalendarDate(2025, 4, 30),
        ),
        UsageDay(
            label = "Thursday, May 1, 2025",
            startedAtEpochMillis = 1_746_086_400_000,
            endedAtEpochMillis = 1_746_172_800_000,
            calendarDate = CalendarDate(2025, 5, 1),
        ),
        UsageDay(
            label = "Friday, May 2, 2025",
            startedAtEpochMillis = 1_746_172_800_000,
            endedAtEpochMillis = 1_746_259_200_000,
            calendarDate = CalendarDate(2025, 5, 2),
        ),
    )

    val projects = listOf(
        Project(
            id = "morgan",
            name = "Morgan Project",
            client = "Morgan Ltd.",
            colorArgb = 0xFF16B981,
            hourlyRateCents = 18_000,
        ),
        Project(
            id = "axion",
            name = "Axion Ltd. Project",
            client = "Axion Ltd.",
            colorArgb = 0xFFF59E0B,
            hourlyRateCents = 16_500,
        ),
        Project(
            id = "harper",
            name = "Harper Project",
            client = "James Harper",
            colorArgb = 0xFFDB2777,
            hourlyRateCents = 14_000,
        ),
        Project(
            id = "acme",
            name = "AcmeCorp Project",
            client = "AcmeCorp",
            colorArgb = 0xFF8B5CF6,
            hourlyRateCents = 17_500,
        ),
        Project(
            id = "daven",
            name = "Daven Retainer",
            client = "Daven Ltd.",
            colorArgb = 0xFF2563EB,
            hourlyRateCents = 15_000,
        ),
    )

    val usageEvents = listOf(
        UsageEvent(
            id = "usage-proposal",
            title = "Morgan Ltd. proposal draft for the April rollout",
            sourceName = "Word",
            sourceType = UsageSourceType.Document,
            startMinute = 12 * 60,
            durationMinutes = 30,
            projectHintId = "morgan",
        ),
        UsageEvent(
            id = "usage-intro-email",
            title = "Re: Intro Axion Ltd.",
            sourceName = "Gmail",
            sourceType = UsageSourceType.Email,
            startMinute = 12 * 60 + 30,
            durationMinutes = 20,
            projectHintId = "axion",
        ),
        UsageEvent(
            id = "usage-harper-meeting",
            title = "Teams meeting with James Harper",
            sourceName = "Teams",
            sourceType = UsageSourceType.Meeting,
            startMinute = 12 * 60 + 50,
            durationMinutes = 15,
            projectHintId = "harper",
        ),
        UsageEvent(
            id = "usage-acme-sheet",
            title = "AcmeCorp Q1 overview spreadsheet",
            sourceName = "Excel",
            sourceType = UsageSourceType.Spreadsheet,
            startMinute = 13 * 60 + 5,
            durationMinutes = 20,
            projectHintId = "acme",
        ),
        UsageEvent(
            id = "usage-daven-chat",
            title = "John (DM) - Daven Ltd.",
            sourceName = "Slack",
            sourceType = UsageSourceType.Messaging,
            startMinute = 13 * 60 + 25,
            durationMinutes = 15,
            projectHintId = "daven",
        ),
        UsageEvent(
            id = "usage-daven-brief",
            title = "Daven performance brief",
            sourceName = "PowerPoint",
            sourceType = UsageSourceType.Presentation,
            startMinute = 13 * 60 + 40,
            durationMinutes = 20,
            projectHintId = "daven",
        ),
        UsageEvent(
            id = "usage-research",
            title = "AD Group research and competitive notes",
            sourceName = "Browser",
            sourceType = UsageSourceType.Browser,
            startMinute = 14 * 60,
            durationMinutes = 30,
            projectHintId = null,
        ),
    )

    val initialEntries = listOf(
        TimeEntry(
            id = "entry-morgan",
            projectId = "morgan",
            title = "Proposal draft review",
            notes = "Drafted proposal language from captured document activity.",
            startMinute = 12 * 60,
            durationMinutes = 30,
            billable = true,
            sourceUsageIds = setOf("usage-proposal"),
        ),
        TimeEntry(
            id = "entry-axion",
            projectId = "axion",
            title = "Intro follow-up",
            notes = "Prepared response and next-step notes.",
            startMinute = 12 * 60 + 30,
            durationMinutes = 20,
            billable = true,
            sourceUsageIds = setOf("usage-intro-email"),
        ),
        TimeEntry(
            id = "entry-harper",
            projectId = "harper",
            title = "Client meeting",
            notes = "Meeting time captured from calendar and call activity.",
            startMinute = 12 * 60 + 50,
            durationMinutes = 15,
            billable = true,
            sourceUsageIds = setOf("usage-harper-meeting"),
        ),
        TimeEntry(
            id = "entry-acme",
            projectId = "acme",
            title = "Q1 reporting pass",
            notes = "Worked through spreadsheet review and cleanup.",
            startMinute = 13 * 60 + 5,
            durationMinutes = 20,
            billable = true,
            sourceUsageIds = setOf("usage-acme-sheet"),
        ),
    )

    return TimeboxxingMockData(
        usageDays = usageDays,
        projects = projects,
        usageEvents = usageEvents,
        initialEntries = initialEntries,
    )
}
