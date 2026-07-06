package com.timeboxxing.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.FileDownload
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingScreenState
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.EntriesExportFormat
import com.timeboxxing.domain.model.isoLabel

@Composable
fun ExportPane(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp, vertical = 24.dp),
        verticalArrangement = Arrangement.spacedBy(18.dp),
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            TbText("Export", style = TbTheme.typography.largeTitle)
            TbText(
                text = "Export your time entries for a date range of up to a month.",
                style = TbTheme.typography.body,
                color = TbTheme.colors.secondaryText,
            )
        }

        ExportRangeCard(state = state, onAction = onAction)
        ExportFormatCard(state = state, onAction = onAction)

        Column(
            modifier = Modifier.widthIn(max = 760.dp).fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbButton(
                    onClick = { onAction(TimeboxxingAction.ExportEntries) },
                    enabled = state.canExportEntries,
                ) {
                    if (state.entriesExporting) {
                        TbSpinner(
                            modifier = Modifier.padding(end = 8.dp),
                            size = 15.dp,
                            color = TbTheme.colors.accentText,
                            trackColor = TbTheme.colors.accentText.copy(alpha = 0.3f),
                        )
                    } else {
                        TbIcon(
                            imageVector = Icons.Rounded.FileDownload,
                            contentDescription = null,
                            modifier = Modifier.padding(end = 8.dp).size(16.dp),
                        )
                    }
                    TbText(
                        text = if (state.entriesExporting) "Exporting" else "Export",
                        style = TbTheme.typography.button,
                    )
                }
                state.entriesExportStatus?.takeIf { state.entriesExporting }?.let { status ->
                    TbText(
                        text = status,
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.secondaryText,
                    )
                }
            }

            if (state.entriesExporting) {
                TbProgressBar(
                    progress = state.entriesExportProgress,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }

        state.entriesExportMessage?.let { message ->
            ExportNotice(message = message)
        }
    }
}

@Composable
private fun ExportRangeCard(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    TbCard(modifier = Modifier.widthIn(max = 760.dp)) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            TbText("Date range", style = TbTheme.typography.headline)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                ExportDateField(
                    modifier = Modifier.weight(1f),
                    label = "From",
                    date = state.exportStartDate,
                    enabled = !state.entriesExporting,
                    onDateSelected = { onAction(TimeboxxingAction.UpdateExportStartDate(it)) },
                )
                ExportDateField(
                    modifier = Modifier.weight(1f),
                    label = "Through",
                    date = state.exportEndDate,
                    enabled = !state.entriesExporting,
                    onDateSelected = { onAction(TimeboxxingAction.UpdateExportEndDate(it)) },
                )
            }
            state.exportRangeError?.let { error ->
                TbText(
                    text = error,
                    style = TbTheme.typography.bodySmall,
                    color = TbTheme.colors.destructive,
                )
            }
        }
    }
}

@Composable
private fun ExportFormatCard(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    val formats = EntriesExportFormat.entries
    TbCard(modifier = Modifier.widthIn(max = 760.dp)) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(18.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            TbText("Format", style = TbTheme.typography.headline)
            TbTabs(
                selectedIndex = state.exportFormat.ordinal,
                labels = formats.map { it.tabLabel() },
                onSelectedIndexChange = { index ->
                    onAction(TimeboxxingAction.SelectEntriesExportFormat(formats[index]))
                },
            )

            if (state.exportFormat == EntriesExportFormat.Pdf) {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    TbText(
                        text = state.exportTemplateName?.let { "Template: $it" } ?: "Using the default template.",
                        style = TbTheme.typography.body,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        TbButton(
                            onClick = { onAction(TimeboxxingAction.ChooseExportTemplate) },
                            enabled = !state.entriesExporting,
                            variant = TbButtonVariant.Secondary,
                        ) {
                            TbText("Upload template", style = TbTheme.typography.button)
                        }
                        if (state.exportTemplateName != null) {
                            TbButton(
                                onClick = { onAction(TimeboxxingAction.ClearExportTemplate) },
                                enabled = !state.entriesExporting,
                                variant = TbButtonVariant.Ghost,
                            ) {
                                TbText("Use default", style = TbTheme.typography.button)
                            }
                        }
                        TbButton(
                            onClick = { onAction(TimeboxxingAction.DownloadDefaultTemplate) },
                            variant = TbButtonVariant.Ghost,
                        ) {
                            TbText("Download default template", style = TbTheme.typography.button)
                        }
                    }
                    TbText(
                        text = "Templates are Handlebars files that use TailwindCSS and render to a self-contained PDF.",
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.secondaryText,
                    )
                }
            }
        }
    }
}

@Composable
private fun ExportDateField(
    label: String,
    date: CalendarDate?,
    onDateSelected: (CalendarDate) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        TbText(
            text = label,
            style = TbTheme.typography.label,
            color = TbTheme.colors.secondaryText,
        )
        CalendarDatePickerMenu(
            selectedDate = date,
            label = date?.isoLabel() ?: "Choose date",
            onDateSelected = onDateSelected,
            modifier = Modifier.fillMaxWidth(),
            enabled = enabled,
        )
    }
}

@Composable
private fun ExportNotice(message: String) {
    TbSurface(
        modifier = Modifier.widthIn(max = 760.dp),
        color = TbTheme.colors.groupedSurface,
        shape = RoundedCornerShape(TbTheme.radii.control),
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 9.dp),
            text = message,
            style = TbTheme.typography.body,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

private fun EntriesExportFormat.tabLabel(): String = when (this) {
    EntriesExportFormat.Json -> "JSON"
    EntriesExportFormat.Csv -> "CSV"
    EntriesExportFormat.Pdf -> "PDF"
}
