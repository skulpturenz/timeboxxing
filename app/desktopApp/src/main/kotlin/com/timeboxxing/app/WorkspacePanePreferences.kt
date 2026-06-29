package com.timeboxxing.app

import com.timeboxxing.app.state.WorkspacePane
import com.timeboxxing.app.state.sanitizeCollapsedWorkspacePanes
import java.util.prefs.Preferences

internal class WorkspacePanePreferences(
    private val store: PreferenceStringStore = JavaPreferenceStringStore(
        preferencesNode(),
    ),
) {
    fun load(): Set<WorkspacePane> =
        sanitizeCollapsedWorkspacePanes(
            store.get(CollapsedWorkspacePanesKey, "")
                .split(PreferenceSeparator)
                .mapNotNull { token -> workspacePaneFromPreferenceToken(token) }
                .toSet(),
        )

    fun save(panes: Set<WorkspacePane>) {
        val serialized = sanitizeCollapsedWorkspacePanes(panes)
            .sortedBy { it.name }
            .joinToString(PreferenceSeparator) { it.preferenceToken }
        store.put(CollapsedWorkspacePanesKey, serialized)
    }
}

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
    Preferences.userRoot().node(WorkspacePanePreferencesNode)

private const val WorkspacePanePreferencesNode = "com.timeboxxing.app.ui"
private const val CollapsedWorkspacePanesKey = "collapsedWorkspacePanes"
private const val PreferenceSeparator = ","

private val WorkspacePane.preferenceToken: String
    get() = when (this) {
        WorkspacePane.UsageSchedule -> "usageSchedule"
        WorkspacePane.TimeEntries -> "timeEntries"
        WorkspacePane.Projects -> "projects"
    }

private fun workspacePaneFromPreferenceToken(token: String): WorkspacePane? =
    when (token.trim()) {
        WorkspacePane.UsageSchedule.preferenceToken -> WorkspacePane.UsageSchedule
        WorkspacePane.TimeEntries.preferenceToken -> WorkspacePane.TimeEntries
        WorkspacePane.Projects.preferenceToken -> WorkspacePane.Projects
        else -> null
    }
