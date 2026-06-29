package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.AiModelOption
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiProvider
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.repository.SettingsRepository

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
