package com.timeboxxing.data.time

import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.UsageDay

expect fun usageDayForCalendarDate(date: CalendarDate): UsageDay

expect fun currentCalendarDate(): CalendarDate

expect fun calendarDateForEpochMillis(epochMillis: Long): CalendarDate
