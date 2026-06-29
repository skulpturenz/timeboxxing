package com.timeboxxing.app

import com.timeboxxing.app.di.timeboxxingPresentationModule
import com.timeboxxing.app.presentation.TimeboxxingRuntime
import com.timeboxxing.app.sidecar.SidecarProcessManager
import com.timeboxxing.app.sidecar.SidecarSessionLog
import com.timeboxxing.app.ui.UsageIconLoader
import org.koin.dsl.module

internal val desktopTimeboxxingModule = module {
    includes(timeboxxingPresentationModule)
    single { SidecarSessionLog() }
    single<SecretStore> { DesktopSecretStore() }
    single { AppearancePreferences() }
    single { SidecarProcessManager(sessionLog = get()) }
    single {
        DesktopTimeboxxingRuntime(
            secretStore = get(),
            appearancePreferences = get(),
            sidecarManager = get(),
            sidecarSessionLog = get(),
            diagnosticsEnabled = DesktopBuildConfig.DiagnosticsEnabled,
        )
    }
    single<TimeboxxingRuntime> { get<DesktopTimeboxxingRuntime>() }
    single<UsageIconLoader> { NativeUsageIconLoader() }
}
