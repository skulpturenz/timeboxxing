package com.timeboxxing.domain.model

data class CalendarDate(
    val year: Int,
    val month: Int,
    val dayOfMonth: Int,
) : Comparable<CalendarDate> {
    init {
        require(year >= 1) { "year must be positive" }
        require(month in 1..12) { "month must be between 1 and 12" }
        require(dayOfMonth in 1..daysInMonth(year, month)) { "dayOfMonth is invalid for month" }
    }

    override fun compareTo(other: CalendarDate): Int =
        compareValuesBy(this, other, CalendarDate::year, CalendarDate::month, CalendarDate::dayOfMonth)
}

fun CalendarDate.isoLabel(): String =
    "${year.toString().padStart(4, '0')}-${month.toString().padStart(2, '0')}-${dayOfMonth.toString().padStart(2, '0')}"

fun minCalendarDate(first: CalendarDate, second: CalendarDate): CalendarDate =
    if (first <= second) first else second

fun maxCalendarDate(first: CalendarDate, second: CalendarDate): CalendarDate =
    if (first >= second) first else second

/** Inclusive day count between [start] and [end]; assumes [start] <= [end]. */
fun daysBetweenInclusive(start: CalendarDate, end: CalendarDate): Int {
    var cursor = start
    var count = 1
    while (cursor < end) {
        cursor = cursor.plusDays(1)
        count += 1
    }
    return count
}

fun CalendarDate.plusDays(delta: Int): CalendarDate {
    var date = this
    when {
        delta > 0 -> repeat(delta) { date = date.nextDay() }
        delta < 0 -> repeat(-delta) { date = date.previousDay() }
    }
    return date
}

fun CalendarDate.plusMonths(delta: Int): CalendarDate {
    val monthIndex = (year - 1) * 12 + (month - 1) + delta
    val nextYear = floorDiv(monthIndex, 12) + 1
    val nextMonth = floorMod(monthIndex, 12) + 1
    val nextDay = dayOfMonth.coerceAtMost(daysInMonth(nextYear, nextMonth))
    return CalendarDate(nextYear, nextMonth, nextDay)
}

fun CalendarDate.startOfMonth(): CalendarDate =
    copy(dayOfMonth = 1)

fun CalendarDate.isoDayOfWeek(): Int {
    val monthOffsets = intArrayOf(0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4)
    val adjustedYear = if (month < 3) year - 1 else year
    val sundayBased = (
        adjustedYear +
            adjustedYear / 4 -
            adjustedYear / 100 +
            adjustedYear / 400 +
            monthOffsets[month - 1] +
            dayOfMonth
        ) % 7
    return if (sundayBased == 0) 7 else sundayBased
}

fun CalendarDate.monthYearLabel(): String =
    "${MonthNames[month - 1]} $year"

fun calendarMonthGrid(month: CalendarDate): List<CalendarDate> {
    val firstOfMonth = month.startOfMonth()
    val firstGridDate = firstOfMonth.plusDays(-(firstOfMonth.isoDayOfWeek() - 1))
    return (0 until 42).map { firstGridDate.plusDays(it) }
}

fun daysInMonth(year: Int, month: Int): Int =
    when (month) {
        1, 3, 5, 7, 8, 10, 12 -> 31
        4, 6, 9, 11 -> 30
        2 -> if (isLeapYear(year)) 29 else 28
        else -> error("Invalid month: $month")
    }

val WeekdayShortLabels = listOf("Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun")

private val MonthNames = listOf(
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
)

private fun CalendarDate.nextDay(): CalendarDate =
    if (dayOfMonth < daysInMonth(year, month)) {
        copy(dayOfMonth = dayOfMonth + 1)
    } else if (month < 12) {
        CalendarDate(year, month + 1, 1)
    } else {
        CalendarDate(year + 1, 1, 1)
    }

private fun CalendarDate.previousDay(): CalendarDate =
    if (dayOfMonth > 1) {
        copy(dayOfMonth = dayOfMonth - 1)
    } else if (month > 1) {
        val previousMonth = month - 1
        CalendarDate(year, previousMonth, daysInMonth(year, previousMonth))
    } else {
        CalendarDate(year - 1, 12, 31)
    }

private fun isLeapYear(year: Int): Boolean =
    year % 4 == 0 && (year % 100 != 0 || year % 400 == 0)

private fun floorDiv(value: Int, divisor: Int): Int {
    val quotient = value / divisor
    val remainder = value % divisor
    return if (remainder != 0 && (value xor divisor) < 0) quotient - 1 else quotient
}

private fun floorMod(value: Int, divisor: Int): Int =
    value - floorDiv(value, divisor) * divisor
