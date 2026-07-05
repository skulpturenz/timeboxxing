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
 * App-wide modal shown when an update is available. It can always be dismissed (closed for the
 * session); on stable builds it additionally offers a persistent "Don't notify on startup" opt-out.
 * Visibility is driven by [TimeboxxingScreenState.showUpdateDialog].
 */
@Composable
fun UpdateAvailableDialog(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    val update = state.update
    val available = update.available ?: return
    // Only stable builds may permanently silence the startup notification; prerelease/dev builds can
    // still close the dialog for now, but it returns on the next launch.
    val canDisableStartupNotify = state.isStableBuild
    val progress = update.downloadProgress

    Dialog(
        onDismissRequest = { onAction(TimeboxxingAction.DismissUpdateDialog) },
        properties = DialogProperties(
            dismissOnBackPress = true,
            dismissOnClickOutside = true,
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
                    TbIconButton(
                        icon = Icons.Rounded.Close,
                        contentDescription = "Close",
                        onClick = { onAction(TimeboxxingAction.DismissUpdateDialog) },
                        variant = TbButtonVariant.Ghost,
                    )
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
                    if (!canDisableStartupNotify) {
                        TbText(
                            text = "This is a prerelease build — please keep it up to date.",
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
                    if (canDisableStartupNotify) {
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
