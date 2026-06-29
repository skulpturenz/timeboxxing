package com.timeboxxing.app.ui

import androidx.compose.ui.graphics.ImageBitmap
import com.timeboxxing.domain.model.UsageApplicationIdentity

interface UsageIconLoader {
    suspend fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap?
}

object NoOpUsageIconLoader : UsageIconLoader {
    override suspend fun loadIcon(identity: UsageApplicationIdentity): ImageBitmap? = null
}
