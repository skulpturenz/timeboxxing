package com.timeboxxing.app.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Close
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingScreenState

/**
 * App-wide modal shown when an update is available. On non-stable (prerelease/dev) builds it cannot
 * be dismissed — the only way out is to install. On stable builds it can be closed, and offers a
 * "Don't notify on startup" opt-out. Visibility is driven by [TimeboxxingScreenState.showUpdateDialog].
 */
@Composable
fun UpdateAvailableDialog(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    val update = state.update
    val available = update.available ?: return
    val forced = !state.isStableBuild
    val progress = update.downloadProgress

    Dialog(
        onDismissRequest = { if (!forced) onAction(TimeboxxingAction.DismissUpdateDialog) },
        // On forced (prerelease) builds, block Esc / scrim-tap dismissal.
        properties = DialogProperties(
            dismissOnBackPress = !forced,
            dismissOnClickOutside = !forced,
        ),
    ) {
        TbSurface(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 440.dp),
            shape = RoundedCornerShape(TbTheme.radii.panel),
            color = TbTheme.colors.surface,
            border = BorderStroke(Dp.Hairline, TbTheme.colors.separator),
            shadowElevation = 18.dp,
        ) {
            Column(
                modifier = Modifier.padding(18.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    TbText(
                        modifier = Modifier.weight(1f),
                        text = "Update available",
                        style = TbTheme.typography.title,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    if (!forced) {
                        TbIconButton(
                            icon = Icons.Rounded.Close,
                            contentDescription = "Close",
                            onClick = { onAction(TimeboxxingAction.DismissUpdateDialog) },
                            variant = TbButtonVariant.Ghost,
                        )
                    }
                }

                Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    TbText(
                        text = "Version ${available.version} is available.",
                        style = TbTheme.typography.body,
                    )
                    update.currentVersion.takeIf { it.isNotBlank() }?.let { current ->
                        TbText(
                            text = "You're on version $current.",
                            style = TbTheme.typography.bodySmall,
                            color = TbTheme.colors.secondaryText,
                        )
                    }
                    if (forced) {
                        TbText(
                            text = "This is a prerelease build — updating is required to continue.",
                            style = TbTheme.typography.bodySmall,
                            color = TbTheme.colors.secondaryText,
                        )
                    }
                }

                update.error?.let { message ->
                    TbText(
                        text = message,
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.destructive,
                    )
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.End),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (!forced) {
                        TbButton(
                            onClick = { onAction(TimeboxxingAction.SetNotifyUpdatesOnStartup(false)) },
                            variant = TbButtonVariant.Secondary,
                            enabled = !update.busy,
                        ) {
                            TbText("Don't notify on startup", style = TbTheme.typography.button)
                        }
                    }
                    TbButton(
                        onClick = { onAction(TimeboxxingAction.StartUpdateInstall) },
                        enabled = progress == null && !update.installing,
                    ) {
                        TbText(
                            text = when {
                                update.installing -> "Restarting to apply update"
                                progress != null -> "Downloading ${(progress * 100).toInt()}%"
                                else -> "Download and install ${available.version}"
                            },
                            style = TbTheme.typography.button,
                        )
                    }
                }
            }
        }
    }
}
