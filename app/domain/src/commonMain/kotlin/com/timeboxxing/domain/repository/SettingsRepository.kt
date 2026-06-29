package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiSettings

interface SettingsRepository {
    suspend fun listModelOptions(): AiModelOptions
    suspend fun getAiSettings(): AiSettings
    suspend fun saveAiSettings(settings: AiSettings): AiSettings
}
