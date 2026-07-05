package com.timeboxxing.app

import java.nio.file.Files
import kotlin.io.path.exists
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull

class UpdateFailureMarkerTest {
    @Test
    fun noMarkerReturnsNull() {
        val dir = Files.createTempDirectory("timeboxxing-update-test")
        assertNull(readAndClearUpdateFailure(dir))
    }

    @Test
    fun readsMessageAndDeletesMarker() {
        val dir = Files.createTempDirectory("timeboxxing-update-test")
        val marker = dir.resolve(UpdateFailureMarkerName)
        Files.writeString(marker, """{"version":"0.0.1-11","log":"C:\\timeboxxing-update.log"}""")

        val message = readAndClearUpdateFailure(dir)

        assertEquals(
            "The update to v0.0.1-11 couldn't be installed. See C:\\timeboxxing-update.log for details.",
            message,
        )
        // Shown only once: the marker is cleared after reading.
        assertFalse(marker.exists())
    }

    @Test
    fun formatOmitsMissingFields() {
        assertEquals(
            "The update to v1.2.3 couldn't be installed.",
            formatUpdateFailureMessage("""{"version":"1.2.3"}"""),
        )
        assertEquals(
            "The update couldn't be installed.",
            formatUpdateFailureMessage("{}"),
        )
    }

    @Test
    fun malformedMarkerYieldsDefaultMessageAndIsCleared() {
        val dir = Files.createTempDirectory("timeboxxing-update-test")
        val marker = dir.resolve(UpdateFailureMarkerName)
        Files.writeString(marker, "not json")

        val message = readAndClearUpdateFailure(dir)

        assertEquals(DefaultUpdateFailureMessage, message)
        assertFalse(marker.exists())
    }
}
