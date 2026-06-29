package com.timeboxxing.app

import com.timeboxxing.app.data.SettingsRepository
import com.timeboxxing.app.model.AiModelOptions
import com.timeboxxing.app.model.AiSettings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

internal class DesktopSettingsRepository(
    private val delegate: SettingsRepository,
    private val secretStore: SecretStore,
    private val onSettingsSaved: () -> Unit,
) : SettingsRepository {
    override suspend fun listModelOptions(): AiModelOptions =
        delegate.listModelOptions()

    override suspend fun getAiSettings(): AiSettings {
        val settings = delegate.getAiSettings()
        val openRouterApiKey = withContext(Dispatchers.IO) { secretStore.read(SecretKey.OpenRouter) }
        val ollamaApiKey = withContext(Dispatchers.IO) { secretStore.read(SecretKey.Ollama) }
        return settings.copy(
            openRouterApiKey = openRouterApiKey,
            ollamaApiKey = ollamaApiKey,
            openRouterSecretExists = settings.openRouterSecretExists || openRouterApiKey.isNotBlank(),
            ollamaSecretExists = settings.ollamaSecretExists || ollamaApiKey.isNotBlank(),
        )
    }

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        withContext(Dispatchers.IO) {
            secretStore.saveOrDelete(SecretKey.OpenRouter, settings.openRouterApiKey)
            secretStore.saveOrDelete(SecretKey.Ollama, settings.ollamaApiKey)
        }
        val saved = delegate.saveAiSettings(
            settings.copy(
                openRouterSecretExists = settings.openRouterApiKey.isNotBlank(),
                ollamaSecretExists = settings.ollamaApiKey.isNotBlank(),
            ),
        )
        onSettingsSaved()
        return saved.copy(
            openRouterApiKey = settings.openRouterApiKey,
            ollamaApiKey = settings.ollamaApiKey,
            openRouterSecretExists = settings.openRouterApiKey.isNotBlank(),
            ollamaSecretExists = settings.ollamaApiKey.isNotBlank(),
        )
    }

    private suspend fun SecretStore.saveOrDelete(key: SecretKey, value: String) {
        if (value.isBlank()) {
            delete(key)
        } else {
            write(key, value)
        }
    }
}
