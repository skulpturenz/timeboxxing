package com.timeboxxing.app

import java.util.prefs.Preferences

internal interface PreferenceStringStore {
    fun get(key: String, defaultValue: String): String
    fun put(key: String, value: String)
}

internal class JavaPreferenceStringStore(
    private val preferences: Preferences,
) : PreferenceStringStore {
    override fun get(key: String, defaultValue: String): String =
        preferences.get(key, defaultValue)

    override fun put(key: String, value: String) {
        preferences.put(key, value)
        runCatching { preferences.flush() }
    }
}

internal fun preferencesNode(): Preferences =
    Preferences.userRoot().node(PreferencesNode)

private const val PreferencesNode = "com.timeboxxing.app.ui"
