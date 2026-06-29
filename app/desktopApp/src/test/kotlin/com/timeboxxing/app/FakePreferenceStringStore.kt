package com.timeboxxing.app

internal class FakePreferenceStringStore(
    initialValues: Map<String, String> = emptyMap(),
) : PreferenceStringStore {
    val values = initialValues.toMutableMap()

    override fun get(key: String, defaultValue: String): String =
        values[key] ?: defaultValue

    override fun put(key: String, value: String) {
        values[key] = value
    }
}
