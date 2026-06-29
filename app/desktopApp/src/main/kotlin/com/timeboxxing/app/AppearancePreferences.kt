package com.timeboxxing.app

import com.timeboxxing.domain.model.AppearanceMode

internal class AppearancePreferences(
    private val store: PreferenceStringStore = JavaPreferenceStringStore(preferencesNode()),
) {
    fun load(): AppearanceMode =
        appearanceModeFromPreferenceToken(store.get(AppearanceModeKey, "")) ?: AppearanceMode.System

    fun save(mode: AppearanceMode) {
        store.put(AppearanceModeKey, mode.preferenceToken)
    }
}

private const val AppearanceModeKey = "appearanceMode"

private val AppearanceMode.preferenceToken: String
    get() = when (this) {
        AppearanceMode.System -> "system"
        AppearanceMode.Light -> "light"
        AppearanceMode.Dark -> "dark"
    }

private fun appearanceModeFromPreferenceToken(token: String): AppearanceMode? =
    when (token.trim()) {
        AppearanceMode.System.preferenceToken -> AppearanceMode.System
        AppearanceMode.Light.preferenceToken -> AppearanceMode.Light
        AppearanceMode.Dark.preferenceToken -> AppearanceMode.Dark
        else -> null
    }
