package com.timeboxxing.app

import com.timeboxxing.domain.model.UpdateChannel

internal class UpdateChannelPreferences(
    private val store: PreferenceStringStore = JavaPreferenceStringStore(preferencesNode()),
) {
    fun load(): UpdateChannel =
        updateChannelFromPreferenceToken(store.get(UpdateChannelKey, "")) ?: UpdateChannel.Stable

    fun save(channel: UpdateChannel) {
        store.put(UpdateChannelKey, channel.preferenceToken)
    }
}

private const val UpdateChannelKey = "updateChannel"

private val UpdateChannel.preferenceToken: String
    get() = when (this) {
        UpdateChannel.Stable -> "stable"
        UpdateChannel.Beta -> "beta"
        UpdateChannel.Alpha -> "alpha"
    }

private fun updateChannelFromPreferenceToken(token: String): UpdateChannel? =
    when (token.trim()) {
        UpdateChannel.Stable.preferenceToken -> UpdateChannel.Stable
        UpdateChannel.Beta.preferenceToken -> UpdateChannel.Beta
        UpdateChannel.Alpha.preferenceToken -> UpdateChannel.Alpha
        else -> null
    }
