package com.timeboxxing.data.grpc

import com.timeboxxing.sidecar.settings.v1.DatabaseMaintenanceStatus
import com.timeboxxing.sidecar.settings.v1.PruneDatabaseRangeResponse
import com.timeboxxing.sidecar.settings.v1.VacuumDatabaseResponse
import kotlin.test.Test
import kotlin.test.assertEquals

class DatabaseMaintenanceMapperTest {
    @Test
    fun mapsDatabaseMaintenanceStatus() {
        val status = DatabaseMaintenanceStatus.newBuilder()
            .setSizeBytes(42)
            .build()

        assertEquals(42L, status.toDatabaseMaintenanceStatus().sizeBytes)
    }

    @Test
    fun mapsPruneDatabaseRangeResponse() {
        val response = PruneDatabaseRangeResponse.newBuilder()
            .setSizeBytes(2048)
            .setLedgerItemsDeleted(1)
            .setLedgerItemTimelineEntriesDeleted(2)
            .setTimelineDeleted(3)
            .setForegroundProcessesDeleted(4)
            .setForegroundProcessMetadataDeleted(5)
            .setTimelineSemanticDocumentsDeleted(6)
            .setTimelineEmbeddingsDeleted(7)
            .setApplicationsDeleted(8)
            .build()

        val result = response.toDatabasePruneResult()

        assertEquals(2048L, result.status.sizeBytes)
        assertEquals(1L, result.counts.ledgerItemsDeleted)
        assertEquals(2L, result.counts.ledgerItemTimelineEntriesDeleted)
        assertEquals(3L, result.counts.timelineDeleted)
        assertEquals(4L, result.counts.foregroundProcessesDeleted)
        assertEquals(5L, result.counts.foregroundProcessMetadataDeleted)
        assertEquals(6L, result.counts.timelineSemanticDocumentsDeleted)
        assertEquals(7L, result.counts.timelineEmbeddingsDeleted)
        assertEquals(8L, result.counts.applicationsDeleted)
        assertEquals(36L, result.counts.totalDeletedRows)
    }

    @Test
    fun mapsVacuumDatabaseResponse() {
        val response = VacuumDatabaseResponse.newBuilder()
            .setSizeBeforeBytes(4096)
            .setSizeAfterBytes(2048)
            .build()

        val result = response.toDatabaseVacuumResult()

        assertEquals(4096L, result.sizeBeforeBytes)
        assertEquals(2048L, result.sizeAfterBytes)
    }
}
