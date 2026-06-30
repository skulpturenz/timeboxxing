package com.timeboxxing.data.mock

import com.timeboxxing.domain.model.CalendarDate
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

    val usageEvents = listOf(
        UsageEvent(
            id = "usage-proposal",
            title = "Morgan Ltd. proposal draft for the April rollout",
            sourceName = "Word",
            sourceType = UsageSourceType.Document,
            startMinute = 12 * 60,
            durationMinutes = 30,
            projectHintId = null,
        ),
        UsageEvent(
            id = "usage-intro-email",
            title = "Re: Intro Axion Ltd.",
            sourceName = "Gmail",
            sourceType = UsageSourceType.Email,
            startMinute = 12 * 60 + 30,
            durationMinutes = 20,
            projectHintId = null,
        ),
        UsageEvent(
            id = "usage-harper-meeting",
            title = "Teams meeting with James Harper",
            sourceName = "Teams",
            sourceType = UsageSourceType.Meeting,
            startMinute = 12 * 60 + 50,
            durationMinutes = 15,
            projectHintId = null,
        ),
        UsageEvent(
            id = "usage-acme-sheet",
            title = "AcmeCorp Q1 overview spreadsheet",
            sourceName = "Excel",
            sourceType = UsageSourceType.Spreadsheet,
            startMinute = 13 * 60 + 5,
            durationMinutes = 20,
            projectHintId = null,
        ),
        UsageEvent(
            id = "usage-daven-chat",
            title = "John (DM) - Daven Ltd.",
            sourceName = "Slack",
            sourceType = UsageSourceType.Messaging,
            startMinute = 13 * 60 + 25,
            durationMinutes = 15,
            projectHintId = null,
        ),
        UsageEvent(
            id = "usage-daven-brief",
            title = "Daven performance brief",
            sourceName = "PowerPoint",
            sourceType = UsageSourceType.Presentation,
            startMinute = 13 * 60 + 40,
            durationMinutes = 20,
            projectHintId = null,
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

    return TimeboxxingMockData(
        usageDays = usageDays,
        projects = emptyList(),
        usageEvents = usageEvents,
        initialEntries = emptyList(),
    )
}
