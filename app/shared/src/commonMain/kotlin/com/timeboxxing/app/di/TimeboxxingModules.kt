package com.timeboxxing.app.di

import com.timeboxxing.app.presentation.StaticTimeboxxingRuntime
import com.timeboxxing.app.presentation.TimeboxxingRuntime
import com.timeboxxing.app.presentation.TimeboxxingViewModel
import com.timeboxxing.app.ui.NoOpUsageIconLoader
import com.timeboxxing.app.ui.UsageIconLoader
import org.koin.dsl.module

val timeboxxingPresentationModule = module {
    single { TimeboxxingViewModel(get()) }
}

val timeboxxingPreviewModule = module {
    single<TimeboxxingRuntime> { StaticTimeboxxingRuntime() }
    single<UsageIconLoader> { NoOpUsageIconLoader }
}
