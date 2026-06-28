package com.timeboxxing.app.data

import com.timeboxxing.app.model.CalendarDate
import com.timeboxxing.app.model.UsageDay

expect fun usageDayForCalendarDate(date: CalendarDate): UsageDay

expect fun currentCalendarDate(): CalendarDate
