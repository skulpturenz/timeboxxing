package com.timeboxxing.app

import com.timeboxxing.app.model.AppearanceMode
import kotlin.test.Test
import kotlin.test.assertEquals

class AppearancePreferencesTest {
    @Test
    fun preferencesRoundTripAppearanceMode() {
        val store = FakePreferenceStringStore()
        val preferences = AppearancePreferences(store)

        preferences.save(AppearanceMode.Dark)

        assertEquals(AppearanceMode.Dark, preferences.load())
    }

    @Test
    fun preferencesDefaultToSystemForUnknownValue() {
        val store = FakePreferenceStringStore(
            initialValues = mapOf("appearanceMode" to "unknown"),
        )
        val preferences = AppearancePreferences(store)

        assertEquals(AppearanceMode.System, preferences.load())
    }

    @Test
    fun preferencesDefaultToSystemWhenUnset() {
        val preferences = AppearancePreferences(FakePreferenceStringStore())

        assertEquals(AppearanceMode.System, preferences.load())
    }
}
