package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.DatabaseMaintenanceStatus
import com.timeboxxing.domain.model.DatabasePruneRange
import com.timeboxxing.domain.model.DatabasePruneResult
import com.timeboxxing.domain.model.DatabaseVacuumResult

interface SettingsRepository {
    suspend fun listModelOptions(): AiModelOptions
    suspend fun getAiSettings(): AiSettings
    suspend fun saveAiSettings(settings: AiSettings): AiSettings
    suspend fun getDatabaseMaintenanceStatus(): DatabaseMaintenanceStatus
    suspend fun pruneDatabaseRange(range: DatabasePruneRange): DatabasePruneResult
    suspend fun vacuumDatabase(): DatabaseVacuumResult
}
