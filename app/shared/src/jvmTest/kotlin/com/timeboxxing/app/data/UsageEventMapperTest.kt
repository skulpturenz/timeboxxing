package com.timeboxxing.app.data

import com.timeboxxing.app.model.UsageDay
import com.timeboxxing.app.model.UsageSourceType
import com.timeboxxing.sidecar.usage.v1.UsageEvent
import com.timeboxxing.sidecar.usage.v1.UsageSource
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class UsageEventMapperTest {
    @Test
    fun mapsBrowserUsageAndClipsToSelectedDay() {
        val day = UsageDay(
            label = "Saturday, June 13, 2026",
            startedAtEpochMillis = 1_781_308_800_000,
            endedAtEpochMillis = 1_781_395_200_000,
        )
        val event = UsageEvent.newBuilder()
            .setId(42)
            .setTitle("Client dashboard")
            .setSourceName("Google Chrome")
            .setSource(UsageSource.USAGE_SOURCE_BROWSER)
            .setApplicationIdentifier("com.google.Chrome")
            .setApplicationPath("/Applications/Google Chrome.app")
            .setStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis - 15 * 60_000))
            .setEndedAt(timestampFromEpochMillis(day.startedAtEpochMillis + 45 * 60_000))
            .build()

        val mapped = event.toUsageEvent(day)

        assertEquals("sidecar-42", mapped?.id)
        assertEquals("Client dashboard", mapped?.title)
        assertEquals("Google Chrome", mapped?.sourceName)
        assertEquals(UsageSourceType.Browser, mapped?.sourceType)
        assertEquals(0, mapped?.startMinute)
        assertEquals(45, mapped?.durationMinutes)
        assertEquals("Google Chrome", mapped?.applicationIdentity?.name)
        assertEquals("com.google.Chrome", mapped?.applicationIdentity?.identifier)
        assertEquals("/Applications/Google Chrome.app", mapped?.applicationIdentity?.path)
    }

    @Test
    fun mapsIdleUsage() {
        val day = UsageDay("Today", 1_000_000, 1_000_000 + 24 * 60 * 60_000)
        val event = UsageEvent.newBuilder()
            .setId(7)
            .setSource(UsageSource.USAGE_SOURCE_IDLE)
            .setStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis + 10 * 60_000))
            .setEndedAt(timestampFromEpochMillis(day.startedAtEpochMillis + 12 * 60_000))
            .build()

        val mapped = event.toUsageEvent(day)

        assertEquals("Idle", mapped?.title)
        assertEquals(UsageSourceType.Idle, mapped?.sourceType)
        assertEquals(10, mapped?.startMinute)
        assertEquals(2, mapped?.durationMinutes)
        assertNull(mapped?.applicationIdentity)
    }

    @Test
    fun mapsActiveUsageToStableDisplayId() {
        val day = UsageDay("Today", 1_000_000, 1_000_000 + 24 * 60 * 60_000)
        val event = UsageEvent.newBuilder()
            .setId(-1)
            .setActive(true)
            .setTitle("Client dashboard")
            .setSourceName("Google Chrome")
            .setSource(UsageSource.USAGE_SOURCE_BROWSER)
            .setApplicationIdentifier("com.google.Chrome")
            .setApplicationPath("/Applications/Google Chrome.app")
            .setStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis + 10 * 60_000))
            .setEndedAt(timestampFromEpochMillis(day.startedAtEpochMillis + 42 * 60_000))
            .build()

        val mapped = event.toUsageEvent(day)

        assertEquals("sidecar-active", mapped?.id)
        assertEquals(true, mapped?.isActive)
        assertEquals(10, mapped?.startMinute)
        assertEquals(32, mapped?.durationMinutes)
        assertEquals("com.google.Chrome", mapped?.applicationIdentity?.identifier)
    }

    @Test
    fun dropsEventsOutsideSelectedDay() {
        val day = UsageDay("Today", 1_000_000, 2_000_000)
        val event = UsageEvent.newBuilder()
            .setId(1)
            .setStartedAt(timestampFromEpochMillis(2_000_000))
            .setEndedAt(timestampFromEpochMillis(3_000_000))
            .build()

        assertNull(event.toUsageEvent(day))
    }
}
