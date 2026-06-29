package com.timeboxxing.app

import com.timeboxxing.app.state.WorkspacePane
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class WorkspacePanePreferencesTest {
    @Test
    fun preferencesRoundTripCollapsedPanes() {
        val store = FakePreferenceStringStore()
        val preferences = WorkspacePanePreferences(store)

        preferences.save(setOf(WorkspacePane.Projects, WorkspacePane.TimeEntries))

        assertEquals(setOf(WorkspacePane.TimeEntries, WorkspacePane.Projects), preferences.load())
    }

    @Test
    fun preferencesIgnoreUnknownPaneIds() {
        val store = FakePreferenceStringStore(
            initialValues = mapOf("collapsedWorkspacePanes" to "projects,unknown,timeEntries"),
        )
        val preferences = WorkspacePanePreferences(store)

        assertEquals(setOf(WorkspacePane.TimeEntries, WorkspacePane.Projects), preferences.load())
    }

    @Test
    fun preferencesSanitizeAllCollapsedStoredState() {
        val store = FakePreferenceStringStore(
            initialValues = mapOf("collapsedWorkspacePanes" to "usageSchedule,timeEntries,projects"),
        )
        val preferences = WorkspacePanePreferences(store)

        assertEquals(setOf(WorkspacePane.TimeEntries, WorkspacePane.Projects), preferences.load())
    }

    @Test
    fun preferencesSaveOnlyPaneIdentifiers() {
        val store = FakePreferenceStringStore()
        val preferences = WorkspacePanePreferences(store)

        preferences.save(setOf(WorkspacePane.UsageSchedule))

        assertEquals("usageSchedule", store.values["collapsedWorkspacePanes"])
        assertTrue(store.values.values.none { it.contains("Morgan", ignoreCase = true) })
    }
}
