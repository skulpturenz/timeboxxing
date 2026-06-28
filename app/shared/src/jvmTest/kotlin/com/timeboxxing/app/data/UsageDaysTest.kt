package com.timeboxxing.app.data

import com.timeboxxing.app.model.CalendarDate
import java.time.Clock
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import kotlin.test.Test
import kotlin.test.assertEquals

class UsageDaysTest {

    @Test
    fun usageDayForCalendarDateUsesFixedZoneBoundaries() {
        val zoneId = ZoneId.of("Pacific/Auckland")
        val date = CalendarDate(2026, 6, 28)

        val day = usageDayForCalendarDate(date, zoneId)

        assertEquals("Sunday, June 28, 2026", day.label)
        assertEquals(date, day.calendarDate)
        assertEquals(
            LocalDate.of(2026, 6, 28).atStartOfDay(zoneId).toInstant().toEpochMilli(),
            day.startedAtEpochMillis,
        )
        assertEquals(
            LocalDate.of(2026, 6, 29).atStartOfDay(zoneId).toInstant().toEpochMilli(),
            day.endedAtEpochMillis,
        )
    }

    @Test
    fun currentCalendarDateUsesFixedClockAndZone() {
        val zoneId = ZoneId.of("Pacific/Auckland")
        val clock = Clock.fixed(Instant.parse("2026-06-27T12:30:00Z"), zoneId)

        assertEquals(
            CalendarDate(2026, 6, 28),
            currentCalendarDate(clock, zoneId),
        )
    }
}
