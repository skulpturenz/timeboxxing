package com.timeboxxing.app.data

import com.timeboxxing.app.model.AiModelOption
import com.timeboxxing.app.model.AiModelOptions
import com.timeboxxing.app.model.AiProvider
import com.timeboxxing.app.model.AiSettings

interface SettingsRepository {
    suspend fun listModelOptions(): AiModelOptions
    suspend fun getAiSettings(): AiSettings
    suspend fun saveAiSettings(settings: AiSettings): AiSettings
}

class StaticSettingsRepository : SettingsRepository {
    override suspend fun listModelOptions(): AiModelOptions =
        AiModelOptions(
            embeddingModels = listOf(
                AiModelOption(1, "qwen/qwen3-embedding-8b", "qwen3-embedding:8b", "Qwen3 Embedding 8B"),
                AiModelOption(2, "qwen/qwen3-embedding-4b", "qwen3-embedding:4b", "Qwen3 Embedding 4B"),
                AiModelOption(3, "google/gemini-embedding-2-preview", "", "Gemini Embedding 2 Preview"),
            ),
            semanticModels = listOf(
                AiModelOption(1, "minimax/minimax-m3", "", "MiniMax M3"),
                AiModelOption(2, "deepseek/deepseek-v4-pro", "", "DeepSeek V4 Pro"),
                AiModelOption(3, "google/gemini-3-flash-preview", "", "Gemini 3 Flash Preview"),
                AiModelOption(4, "google/gemma-4-26b-a4b-it", "gemma4:26b", "Gemma 4 26B A4B IT"),
            ),
        )

    override suspend fun getAiSettings(): AiSettings =
        AiSettings(provider = AiProvider.OpenRouter, openRouterSecretExists = true)

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings = settings
}

class UnavailableSettingsRepository(
    private val message: String,
) : SettingsRepository {
    override suspend fun listModelOptions(): AiModelOptions {
        throw IllegalStateException(message)
    }

    override suspend fun getAiSettings(): AiSettings {
        throw IllegalStateException(message)
    }

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        throw IllegalStateException(message)
    }
}
