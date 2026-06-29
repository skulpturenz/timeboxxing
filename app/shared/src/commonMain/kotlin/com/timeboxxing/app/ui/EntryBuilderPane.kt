package com.timeboxxing.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.ContentCopy
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.material.icons.rounded.Remove
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.timeboxxing.domain.model.EntryDraft
import com.timeboxxing.domain.model.EntryMode
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimeEntry
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.model.UsageSourceType
import com.timeboxxing.domain.model.formatClockTime
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.domain.model.formatTimeRange
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingScreenState

@Composable
fun EntryBuilderPane(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    showInlineSummary: Boolean,
    modifier: Modifier = Modifier,
    headerAction: @Composable (() -> Unit)? = null,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(TbTheme.colors.appBackground)
            .padding(20.dp)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "Time entries",
                    style = TbTheme.typography.largeTitle,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = "Create billable entries from usage history",
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            ModeToggle(
                mode = state.mode,
                onModeChange = { onAction(TimeboxxingAction.ChangeMode(it)) },
            )
            headerAction?.invoke()
        }

        if (showInlineSummary) {
            InlineSummary(state = state, onAction = onAction)
        }

        DraftEditor(
            state = state,
            onAction = onAction,
        )

        EntryList(
            state = state,
            onAction = onAction,
        )
    }
}

@Composable
private fun DraftEditor(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    val draft = state.draft
    val selectedCount = state.selectedUsageIds.size

    TbCard {
        Column(
            modifier = Modifier.padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    TbText(
                        text = "Create entry",
                        style = TbTheme.typography.title,
                    )
                    TbText(
                        text = if (selectedCount == 0) {
                            "Start manually or select captured usage on the left."
                        } else {
                            "$selectedCount captured item${if (selectedCount == 1) "" else "s"} selected"
                        },
                        style = TbTheme.typography.body,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }

            if (state.selectedUsageEvents.isNotEmpty()) {
                SelectedUsagePreview(events = state.selectedUsageEvents)
            }

            ProjectPicker(
                projects = state.projects,
                selectedProjectId = draft.projectId,
                onProjectSelected = { onAction(TimeboxxingAction.UpdateDraftProject(it)) },
            )

            TbTextField(
                modifier = Modifier.fillMaxWidth(),
                value = draft.title,
                onValueChange = { onAction(TimeboxxingAction.UpdateDraftTitle(it)) },
                label = "Entry title",
                singleLine = true,
            )

            TbTextField(
                modifier = Modifier.fillMaxWidth(),
                value = draft.notes,
                onValueChange = { onAction(TimeboxxingAction.UpdateDraftNotes(it)) },
                label = "Notes",
                minLines = 2,
                maxLines = 4,
            )

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TimeStepper(
                    modifier = Modifier.weight(1f),
                    label = "Start",
                    value = formatClockTime(draft.startMinute),
                    onDecrease = {
                        onAction(TimeboxxingAction.UpdateDraftStart(draft.startMinute - state.zoomMinutes))
                    },
                    onIncrease = {
                        onAction(TimeboxxingAction.UpdateDraftStart(draft.startMinute + state.zoomMinutes))
                    },
                )

                DurationEditor(
                    modifier = Modifier.weight(1f),
                    draft = draft,
                    onAction = onAction,
                )
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { onAction(TimeboxxingAction.UpdateDraftBillable(!draft.billable)) },
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbCheckbox(
                    checked = draft.billable,
                    onCheckedChange = { onAction(TimeboxxingAction.UpdateDraftBillable(it)) },
                    accessibilityLabel = "Billable",
                )
                Column {
                    TbText(
                        text = "Billable",
                        style = TbTheme.typography.headline,
                    )
                    TbText(
                        text = "Include this entry in invoice totals",
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.secondaryText,
                    )
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbButton(
                    modifier = Modifier.weight(1f),
                    onClick = { onAction(TimeboxxingAction.NewBlankDraft) },
                    variant = TbButtonVariant.Secondary,
                ) {
                    TbText("New blank", style = TbTheme.typography.button)
                }
                TbButton(
                    modifier = Modifier.weight(1f),
                    onClick = { onAction(TimeboxxingAction.AddDraftEntry) },
                ) {
                    TbIcon(
                        imageVector = Icons.Rounded.Add,
                        contentDescription = null,
                        modifier = Modifier
                            .padding(end = 6.dp)
                            .size(17.dp),
                    )
                    TbText("Add entry", style = TbTheme.typography.button)
                }
            }
        }
    }
}

@Composable
private fun SelectedUsagePreview(events: List<UsageEvent>) {
    TbSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.accentSubtle,
        border = androidx.compose.foundation.BorderStroke(
            androidx.compose.ui.unit.Dp.Hairline,
            TbTheme.colors.separator,
        ),
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            TbText(
                text = "Selected usage",
                style = TbTheme.typography.label,
                color = TbTheme.colors.text,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            events.take(3).forEach { event ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    ProjectColorDot(
                        Project(
                            id = event.id,
                            name = event.sourceName,
                            client = event.sourceName,
                            colorArgb = sourcePreviewColor(event.sourceType),
                            hourlyRateCents = 0,
                        ),
                    )
                    TbText(
                        modifier = Modifier.weight(1f),
                        text = event.title,
                        style = TbTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TbText(
                        text = formatDuration(event.durationMinutes),
                        style = TbTheme.typography.label,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 1,
                    )
                }
            }
            if (events.size > 3) {
                TbText(
                    text = "+${events.size - 3} more",
                    style = TbTheme.typography.label,
                    color = TbTheme.colors.secondaryText,
                )
            }
        }
    }
}

@Composable
private fun ProjectPicker(
    projects: List<Project>,
    selectedProjectId: String,
    onProjectSelected: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val selected = projects.firstOrNull { it.id == selectedProjectId } ?: projects.first()

    TbMenu(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        anchor = {
            TbButton(
                modifier = Modifier.fillMaxWidth(),
                onClick = { expanded = true },
                variant = TbButtonVariant.Secondary,
            ) {
                ProjectColorDot(selected)
                TbText(
                    modifier = Modifier
                        .padding(start = 8.dp)
                        .weight(1f),
                    text = selected.name,
                    style = TbTheme.typography.body,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbIcon(
                    imageVector = Icons.Rounded.ExpandMore,
                    contentDescription = null,
                    modifier = Modifier.size(17.dp),
                    tint = TbTheme.colors.accent,
                )
            }
        },
        panelContent = {
            projects.forEach { project ->
                TbMenuItem(
                    onClick = {
                        expanded = false
                        onProjectSelected(project.id)
                    },
                ) {
                    ProjectColorDot(project)
                    Column {
                        TbText(
                            project.name,
                            style = TbTheme.typography.body,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        TbText(
                            project.client,
                            style = TbTheme.typography.bodySmall,
                            color = TbTheme.colors.secondaryText,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
        },
    )
}

@Composable
private fun TimeStepper(
    label: String,
    value: String,
    onDecrease: () -> Unit,
    onIncrease: () -> Unit,
    modifier: Modifier = Modifier,
) {
    TbSurface(
        modifier = modifier,
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.groupedSurface,
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            TbText(
                text = label,
                style = TbTheme.typography.label,
                color = TbTheme.colors.secondaryText,
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbIconButton(
                    icon = Icons.Rounded.Remove,
                    contentDescription = "Move start earlier",
                    onClick = onDecrease,
                )
                TbText(
                    modifier = Modifier.weight(1f),
                    text = value,
                    style = TbTheme.typography.title2,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbIconButton(
                    icon = Icons.Rounded.Add,
                    contentDescription = "Move start later",
                    onClick = onIncrease,
                )
            }
        }
    }
}

@Composable
private fun DurationEditor(
    draft: EntryDraft,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    TbSurface(
        modifier = modifier,
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.groupedSurface,
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            TbText(
                text = "Duration",
                style = TbTheme.typography.label,
                color = TbTheme.colors.secondaryText,
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbIconButton(
                    icon = Icons.Rounded.Remove,
                    contentDescription = "Shorten duration",
                    onClick = {
                        onAction(TimeboxxingAction.UpdateDraftDuration(draft.durationMinutes - 5))
                    },
                )
                TbTextField(
                    modifier = Modifier.weight(1f),
                    value = draft.durationMinutes.toString(),
                    onValueChange = { input ->
                        input.filter { it.isDigit() }.toIntOrNull()?.let {
                            onAction(TimeboxxingAction.UpdateDraftDuration(it))
                        }
                    },
                    singleLine = true,
                    suffix = { TbText("m", style = TbTheme.typography.bodySmall, color = TbTheme.colors.secondaryText) },
                    keyboardType = KeyboardType.Number,
                )
                TbIconButton(
                    icon = Icons.Rounded.Add,
                    contentDescription = "Lengthen duration",
                    onClick = {
                        onAction(TimeboxxingAction.UpdateDraftDuration(draft.durationMinutes + 5))
                    },
                )
            }
        }
    }
}

@Composable
private fun EntryList(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    val usageById = remember(state.usageEvents) { state.usageEvents.associateBy { it.id } }

    Column(
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                TbText(
                    text = "Draft entries",
                    style = TbTheme.typography.title2,
                )
                TbText(
                    text = "${state.entries.size} entries, ${formatDuration(state.billableMinutes)} billable",
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                )
            }

            TbButton(onClick = { onAction(TimeboxxingAction.ConfirmPrimaryAction) }) {
                TbIcon(
                    imageVector = Icons.Rounded.Check,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(end = 6.dp)
                        .size(17.dp),
                )
                TbText(state.primaryActionLabel, style = TbTheme.typography.button)
            }
        }

        state.entries.forEach { entry ->
            EntryCard(
                entry = entry,
                project = state.projectFor(entry.projectId),
                sourceEvents = entry.sourceUsageIds.mapNotNull { usageById[it] },
                showRate = state.mode == EntryMode.Invoice,
                onDuplicate = { onAction(TimeboxxingAction.DuplicateEntry(entry.id)) },
                onDelete = { onAction(TimeboxxingAction.DeleteEntry(entry.id)) },
            )
        }
    }
}

@Composable
private fun EntryCard(
    entry: TimeEntry,
    project: Project,
    sourceEvents: List<UsageEvent>,
    showRate: Boolean,
    onDuplicate: () -> Unit,
    onDelete: () -> Unit,
) {
    TbCard {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = 86.dp),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxHeight()
                    .width(4.dp)
                    .background(Color(project.colorArgb)),
            )
            Column(
                modifier = Modifier.padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.Top,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        TbText(
                            text = entry.title,
                            style = TbTheme.typography.headline,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                        TbText(
                            text = project.name,
                            style = TbTheme.typography.bodySmall,
                            color = TbTheme.colors.secondaryText,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    TbText(
                        text = formatDuration(entry.durationMinutes),
                        style = TbTheme.typography.headline,
                    )
                }

                TbText(
                    text = formatTimeRange(entry.startMinute, entry.durationMinutes),
                    style = TbTheme.typography.caption,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )

                if (sourceEvents.isNotEmpty()) {
                    EntrySourceSummary(sourceEvents = sourceEvents)
                }

                if (entry.notes.isNotBlank()) {
                    TbText(
                        text = entry.notes,
                        style = TbTheme.typography.caption,
                        color = TbTheme.colors.secondaryText,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (entry.billable) {
                        TbBadge(
                            label = if (showRate) {
                                formatMoney(project.hourlyRateCents * entry.durationMinutes / 60)
                            } else {
                                "Billable"
                            },
                        )
                    }

                    TbIconButton(
                        icon = Icons.Rounded.ContentCopy,
                        contentDescription = "Duplicate entry",
                        onClick = onDuplicate,
                        variant = TbButtonVariant.Ghost,
                    )
                    TbIconButton(
                        icon = Icons.Rounded.Delete,
                        contentDescription = "Delete entry",
                        onClick = onDelete,
                        variant = TbButtonVariant.Destructive,
                    )
                }
            }
        }
    }
}

@Composable
private fun EntrySourceSummary(sourceEvents: List<UsageEvent>) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbText(
            text = "From",
            style = TbTheme.typography.label,
            color = TbTheme.colors.tertiaryText,
            maxLines = 1,
        )
        sourceEvents.take(2).forEach { event ->
            TbBadge(
                label = event.sourceName,
                color = TbTheme.colors.secondaryText,
                background = TbTheme.colors.controlFill,
            )
        }
        if (sourceEvents.size > 2) {
            TbBadge(
                label = "+${sourceEvents.size - 2}",
                color = TbTheme.colors.secondaryText,
                background = TbTheme.colors.controlFill,
            )
        }
    }
}

@Composable
fun ProjectColorDot(project: Project) {
    Box(
        modifier = Modifier
            .size(10.dp)
            .background(Color(project.colorArgb), RoundedCornerShape(999.dp)),
    )
}

fun formatMoney(cents: Int): String {
    val dollars = cents / 100
    val remainder = cents % 100
    return "$${dollars}.${remainder.toString().padStart(2, '0')}"
}

private fun sourcePreviewColor(sourceType: UsageSourceType): Long =
    when (sourceType) {
        UsageSourceType.Browser -> 0xFFFF9F0A
        UsageSourceType.Idle -> 0xFF6B7280
        else -> 0xFF64748B
    }
