package com.timeboxxing.app.sidecar

import com.timeboxxing.app.model.DiagnosticsLogLine
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.time.Clock
import java.time.format.DateTimeFormatter

class SidecarSessionLog(
    private val maxLines: Int = DefaultMaxLines,
    private val clock: Clock = Clock.systemDefaultZone(),
) {
    private val entries = ArrayDeque<DiagnosticsLogLine>()
    private val _lines = MutableStateFlow<List<DiagnosticsLogLine>>(emptyList())
    private var nextSequence = 1L

    val lines: StateFlow<List<DiagnosticsLogLine>> = _lines.asStateFlow()

    @Synchronized
    fun append(message: String) {
        entries.addLast(
            DiagnosticsLogLine(
                sequence = nextSequence++,
                timestamp = TimestampFormatter.format(clock.instant().atZone(clock.zone)),
                message = message,
            ),
        )
        while (entries.size > maxLines.coerceAtLeast(1)) {
            entries.removeFirst()
        }
        _lines.value = entries.toList()
    }

    fun snapshot(): List<DiagnosticsLogLine> = lines.value

    private companion object {
        const val DefaultMaxLines = 5_000
        val TimestampFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")
    }
}
