package com.timeboxxing.app

internal class UpdateNotificationPreferences(
    private val store: PreferenceStringStore = JavaPreferenceStringStore(preferencesNode()),
) {
    fun loadNotifyOnStartup(): Boolean =
        when (store.get(NotifyOnStartupKey, "").trim().lowercase()) {
            "false" -> false
            else -> true // default on; unrecognized/empty means the user hasn't opted out
        }

    fun saveNotifyOnStartup(enabled: Boolean) {
        store.put(NotifyOnStartupKey, enabled.toString())
    }
}

private const val NotifyOnStartupKey = "notifyUpdatesOnStartup"
