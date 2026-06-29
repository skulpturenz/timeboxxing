package com.timeboxxing.app

import com.timeboxxing.domain.repository.SettingsRepository
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiSettings
import kotlinx.coroutines.runBlocking
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse

class DesktopSettingsRepositoryTest {
    @Test
    fun getAiSettingsMergesStoredSecrets() = runBlocking {
        val delegate = FakeSettingsRepository(
            aiSettings = AiSettings(openRouterSecretExists = false, ollamaSecretExists = false),
        )
        val secrets = FakeSecretStore(
            initialSecrets = mapOf(
                SecretKey.OpenRouter to "openrouter-secret",
                SecretKey.Ollama to "ollama-secret",
            ),
        )
        val repository = DesktopSettingsRepository(delegate, secrets) {}

        val settings = repository.getAiSettings()

        assertEquals("openrouter-secret", settings.openRouterApiKey)
        assertEquals("ollama-secret", settings.ollamaApiKey)
        assertEquals(true, settings.openRouterSecretExists)
        assertEquals(true, settings.ollamaSecretExists)
    }

    @Test
    fun saveAiSettingsStoresAndDeletesSecrets() = runBlocking {
        var restartCount = 0
        val delegate = FakeSettingsRepository()
        val secrets = FakeSecretStore(
            initialSecrets = mapOf(
                SecretKey.OpenRouter to "old-openrouter-secret",
                SecretKey.Ollama to "old-ollama-secret",
            ),
        )
        val repository = DesktopSettingsRepository(delegate, secrets) { restartCount++ }

        val saved = repository.saveAiSettings(
            AiSettings(
                openRouterApiKey = "",
                ollamaApiKey = "new-ollama-secret",
            ),
        )

        assertFalse(SecretKey.OpenRouter in secrets.values)
        assertEquals("new-ollama-secret", secrets.values[SecretKey.Ollama])
        assertEquals(1, restartCount)
        assertEquals("", delegate.savedSettings?.openRouterApiKey)
        assertEquals("new-ollama-secret", delegate.savedSettings?.ollamaApiKey)
        assertEquals("", saved.openRouterApiKey)
        assertEquals("new-ollama-secret", saved.ollamaApiKey)
        assertEquals(false, saved.openRouterSecretExists)
        assertEquals(true, saved.ollamaSecretExists)
    }
}

private class FakeSettingsRepository(
    private val aiSettings: AiSettings = AiSettings(),
) : SettingsRepository {
    var savedSettings: AiSettings? = null

    override suspend fun listModelOptions(): AiModelOptions = AiModelOptions()

    override suspend fun getAiSettings(): AiSettings = aiSettings

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        savedSettings = settings
        return settings.copy(openRouterApiKey = "", ollamaApiKey = "")
    }
}

private class FakeSecretStore(
    initialSecrets: Map<SecretKey, String> = emptyMap(),
) : SecretStore {
    val values = initialSecrets.toMutableMap()

    override suspend fun read(key: SecretKey): String =
        values[key].orEmpty()

    override suspend fun write(key: SecretKey, value: String) {
        values[key] = value
    }

    override suspend fun delete(key: SecretKey) {
        values.remove(key)
    }
}
