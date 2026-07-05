package com.timeboxxing.app.presentation

import com.timeboxxing.domain.model.UpdateChannel

/** A release that is newer than the running build and can be installed. */
data class AvailableUpdate(
    val version: String,
    val downloadUrl: String,
    val notes: String?,
)

sealed interface UpdateCheckResult {
    /** The running build is the newest available release. */
    data object UpToDate : UpdateCheckResult

    /** A newer release is available. */
    data class Available(val update: AvailableUpdate) : UpdateCheckResult

    /** Update checks are disabled for this build (e.g. local/dev builds). */
    data object Unsupported : UpdateCheckResult
}

/**
 * Platform capability for self-updating the desktop app. The desktop implementation checks GitHub
 * releases, downloads the matching installer, applies it in place and relaunches. Mirrors the
 * [TimesheetExportFileWriter] capability: defined in commonMain, implemented per platform and
 * exposed through [TimeboxxingRuntime].
 */
interface AppUpdater {
    /** The version string of the running build (e.g. "0.0.5"). */
    val currentVersion: String

    /** Queries the release source on [channel] for a newer version. */
    suspend fun check(channel: UpdateChannel): UpdateCheckResult

    /**
     * Downloads [update] and applies it in place, then relaunches the app. On success this
     * terminates the current process, so it normally does not return; [onProgress] reports the
     * download fraction in the range 0f..1f.
     */
    suspend fun downloadAndInstall(update: AvailableUpdate, onProgress: (Float) -> Unit)
}

/** No-op updater for previews, tests and the mock runtime. Reports updates as unsupported. */
class StaticAppUpdater(
    override val currentVersion: String = "",
) : AppUpdater {
    override suspend fun check(channel: UpdateChannel): UpdateCheckResult = UpdateCheckResult.Unsupported

    override suspend fun downloadAndInstall(update: AvailableUpdate, onProgress: (Float) -> Unit) {
        onProgress(1f)
    }
}
