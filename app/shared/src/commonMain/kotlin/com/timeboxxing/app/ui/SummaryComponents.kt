package com.timeboxxing.app.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.model.TimesheetExportFormat
import com.timeboxxing.domain.model.formatDuration
import com.timeboxxing.app.presentation.TimeboxxingAction
import com.timeboxxing.app.presentation.TimeboxxingScreenState

internal const val DefaultProjectColorArgb = 0xFF00FFEE
internal val ProjectPaletteColorOptions = listOf(
    0xFF4F7CFF,
    0xFF33B679,
    0xFFFFB020,
    0xFFE25563,
    0xFF9B6DFF,
    0xFFFF7A45,
    0xFF2FA7B8,
    0xFF6E7F80,
)

@Composable
fun ProjectSummaryRail(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
    headerAction: @Composable (() -> Unit)? = null,
) {
    var showingCreateProject by remember { mutableStateOf(false) }
    var projectPendingDelete by remember { mutableStateOf<Project?>(null) }

    if (showingCreateProject) {
        CreateProjectDialog(
            projects = state.projects,
            onDismiss = { showingCreateProject = false },
            onCreate = { name, colorArgb ->
                onAction(TimeboxxingAction.CreateProject(name, colorArgb))
                showingCreateProject = false
            },
        )
    }
    projectPendingDelete?.let { project ->
        DeleteProjectDialog(
            project = project,
            onDismiss = { projectPendingDelete = null },
            onDelete = {
                onAction(TimeboxxingAction.DeleteProject(project.id))
                projectPendingDelete = null
            },
        )
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(TbTheme.colors.groupedSurface)
            .padding(22.dp),
        verticalArrangement = Arrangement.spacedBy(18.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbText(
                modifier = Modifier.weight(1f),
                text = "Projects",
                style = TbTheme.typography.title,
            )
            TbIconButton(
                icon = Icons.Rounded.Add,
                contentDescription = "New project",
                onClick = { showingCreateProject = true },
                variant = TbButtonVariant.Ghost,
            )
            headerAction?.invoke()
        }

        ProjectTotals(
            state = state,
            onDeleteProject = { projectPendingDelete = it },
            onCreateProjectClick = { showingCreateProject = true },
            framedEmptyState = true,
        )

        TbHorizontalDivider()

        TotalsCard(state)

        TimesheetPrimaryActionButton(
            state = state,
            onAction = onAction,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
fun InlineSummary(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
) {
    TbCard {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                TbText(
                    text = "Summary",
                    style = TbTheme.typography.title2,
                )
                TimesheetPrimaryActionButton(state = state, onAction = onAction)
            }
            TotalsRow(state)
            ProjectTotals(
                state = state,
                framedEmptyState = false,
            )
        }
    }
}

@Composable
fun TimesheetPrimaryActionButton(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    val enabled = state.entries.isNotEmpty() && !state.timesheetExporting
    var expanded by remember { mutableStateOf(false) }

    TbMenu(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        anchor = {
            TbButton(
                modifier = modifier,
                enabled = enabled,
                onClick = { expanded = true },
            ) {
                TbText("Export as", style = TbTheme.typography.button)
                TbIcon(
                    imageVector = Icons.Rounded.ExpandMore,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(start = 6.dp)
                        .size(17.dp),
                )
            }
        },
        panelContent = {
            TbMenuItem(
                onClick = {
                    expanded = false
                    onAction(TimeboxxingAction.ExportTimesheet(TimesheetExportFormat.Json))
                },
            ) {
                TbText("JSON", style = TbTheme.typography.body)
            }
            TbMenuItem(
                onClick = {
                    expanded = false
                    onAction(TimeboxxingAction.ExportTimesheet(TimesheetExportFormat.Csv))
                },
            ) {
                TbText("CSV", style = TbTheme.typography.body)
            }
        },
    )
}

@Composable
private fun ProjectTotals(
    state: TimeboxxingScreenState,
    onDeleteProject: ((Project) -> Unit)? = null,
    onCreateProjectClick: (() -> Unit)? = null,
    framedEmptyState: Boolean = true,
) {
    if (state.projects.isEmpty()) {
        EmptyProjectState(
            onCreateProjectClick = onCreateProjectClick,
            framed = framedEmptyState,
        )
        return
    }

    val maxMinutes = state.projects
        .maxOfOrNull { project -> state.minutesForProject(project.id) }
        ?.coerceAtLeast(1)
        ?: 1

    Column(
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        state.projects.forEach { project ->
            ProjectTotalRow(
                project = project,
                minutes = state.minutesForProject(project.id),
                maxMinutes = maxMinutes,
                onDelete = onDeleteProject?.let { deleteProject -> { deleteProject(project) } },
            )
        }
    }
}

@Composable
private fun ProjectTotalRow(
    project: Project,
    minutes: Int,
    maxMinutes: Int,
    onDelete: (() -> Unit)?,
) {
    val fraction = if (minutes <= 0) {
        0f
    } else {
        (minutes.toFloat() / maxMinutes.toFloat()).coerceIn(0.08f, 1f)
    }

    TbSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(TbTheme.radii.card),
        color = TbTheme.colors.elevatedSurface,
        shadowElevation = 1.dp,
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                ProjectColorDot(project)
                TbText(
                    modifier = Modifier.weight(1f),
                    text = project.name,
                    style = TbTheme.typography.headline,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = formatDuration(minutes),
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                )
                if (onDelete != null) {
                    TbIconButton(
                        icon = Icons.Rounded.Delete,
                        contentDescription = "Delete project",
                        onClick = onDelete,
                        variant = TbButtonVariant.Destructive,
                    )
                }
            }

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(6.dp)
                    .background(TbTheme.colors.groupedSurface, RoundedCornerShape(999.dp)),
            ) {
                if (fraction > 0f) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth(fraction)
                            .height(6.dp)
                            .background(Color(project.colorArgb), RoundedCornerShape(999.dp)),
                    )
                }
            }
        }
    }
}

@Composable
private fun EmptyProjectState(
    onCreateProjectClick: (() -> Unit)?,
    framed: Boolean,
) {
    val content: @Composable () -> Unit = {
        Column(
            modifier = if (framed) Modifier.padding(16.dp) else Modifier,
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            TbText(
                text = "No projects yet",
                style = TbTheme.typography.title2,
            )
            TbText(
                text = "Create a project to assign time.",
                style = TbTheme.typography.body,
                color = TbTheme.colors.secondaryText,
            )
            if (onCreateProjectClick != null) {
                TbButton(
                    modifier = Modifier.fillMaxWidth(),
                    onClick = onCreateProjectClick,
                    variant = TbButtonVariant.Secondary,
                ) {
                    TbIcon(
                        imageVector = Icons.Rounded.Add,
                        contentDescription = null,
                        modifier = Modifier
                            .padding(end = 6.dp)
                            .size(17.dp),
                    )
                    TbText("New project", style = TbTheme.typography.button)
                }
            }
        }
    }

    if (framed) {
        TbCard { content() }
    } else {
        content()
    }
}

@Composable
private fun DeleteProjectDialog(
    project: Project,
    onDismiss: () -> Unit,
    onDelete: () -> Unit,
) {
    Dialog(onDismissRequest = onDismiss) {
        TbSurface(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 420.dp),
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
                        text = "Delete ${project.name}?",
                        style = TbTheme.typography.title,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TbIconButton(
                        icon = Icons.Rounded.Close,
                        contentDescription = "Close",
                        onClick = onDismiss,
                        variant = TbButtonVariant.Ghost,
                    )
                }

                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    TbText(
                        text = "Any linked timesheets or invoices will be unlinked.",
                        style = TbTheme.typography.body,
                    )
                    TbText(
                        text = "This will not delete existing time entries in this pass.",
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.secondaryText,
                    )
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.End),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    TbButton(
                        onClick = onDismiss,
                        variant = TbButtonVariant.Secondary,
                    ) {
                        TbText("Cancel", style = TbTheme.typography.button)
                    }
                    TbButton(
                        onClick = onDelete,
                        variant = TbButtonVariant.Destructive,
                    ) {
                        TbIcon(
                            imageVector = Icons.Rounded.Delete,
                            contentDescription = null,
                            modifier = Modifier
                                .padding(end = 6.dp)
                                .size(17.dp),
                        )
                        TbText("Delete project", style = TbTheme.typography.button)
                    }
                }
            }
        }
    }
}

@Composable
private fun CreateProjectDialog(
    projects: List<Project>,
    onDismiss: () -> Unit,
    onCreate: (String, Long) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var selectedColorArgb by remember { mutableStateOf(DefaultProjectColorArgb) }
    val trimmedName = name.trim()
    val duplicateName = projects.any { it.name.equals(trimmedName, ignoreCase = true) }
    val canCreate = trimmedName.isNotEmpty() && !duplicateName
    val paletteColorOptions = ProjectPaletteColorOptions

    Dialog(onDismissRequest = onDismiss) {
        TbSurface(
            modifier = Modifier
                .fillMaxWidth()
                .widthIn(max = 420.dp),
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
                        text = "New project",
                        style = TbTheme.typography.title,
                    )
                    TbIconButton(
                        icon = Icons.Rounded.Close,
                        contentDescription = "Close",
                        onClick = onDismiss,
                        variant = TbButtonVariant.Ghost,
                    )
                }

                TbTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = "Name",
                    singleLine = true,
                )

                if (duplicateName && trimmedName.isNotEmpty()) {
                    TbText(
                        text = "A project with this name already exists.",
                        style = TbTheme.typography.bodySmall,
                        color = TbTheme.colors.destructive,
                    )
                }

                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    TbText(
                        text = "Colour",
                        style = TbTheme.typography.label,
                        color = TbTheme.colors.secondaryText,
                    )
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(10.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            ProjectColorSwatch(
                                colorArgb = DefaultProjectColorArgb,
                                selected = selectedColorArgb == DefaultProjectColorArgb,
                                onClick = { selectedColorArgb = DefaultProjectColorArgb },
                            )
                            TbText(
                                text = "Default",
                                style = TbTheme.typography.bodySmall,
                                color = TbTheme.colors.secondaryText,
                            )
                            paletteColorOptions.take(4).forEach { colorArgb ->
                                ProjectColorSwatch(
                                    colorArgb = colorArgb,
                                    selected = colorArgb == selectedColorArgb,
                                    onClick = { selectedColorArgb = colorArgb },
                                )
                            }
                        }
                        paletteColorOptions.drop(4).chunked(6).forEach { rowColors ->
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(10.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                rowColors.forEach { colorArgb ->
                                    ProjectColorSwatch(
                                        colorArgb = colorArgb,
                                        selected = colorArgb == selectedColorArgb,
                                        onClick = { selectedColorArgb = colorArgb },
                                    )
                                }
                            }
                        }
                    }
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(10.dp, Alignment.End),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    TbButton(
                        onClick = onDismiss,
                        variant = TbButtonVariant.Secondary,
                    ) {
                        TbText("Cancel", style = TbTheme.typography.button)
                    }
                    TbButton(
                        onClick = { onCreate(trimmedName, selectedColorArgb) },
                        enabled = canCreate,
                    ) {
                        TbText("Create project", style = TbTheme.typography.button)
                    }
                }
            }
        }
    }
}

@Composable
private fun ProjectColorSwatch(
    colorArgb: Long,
    selected: Boolean,
    onClick: () -> Unit,
) {
    val shape = RoundedCornerShape(999.dp)
    val swatchColor = Color(colorArgb)
    val borderColor = if (selected) TbTheme.colors.text else TbTheme.colors.separator

    Box(
        modifier = Modifier
            .size(30.dp)
            .clip(shape)
            .background(swatchColor)
            .border(BorderStroke(if (selected) 2.dp else Dp.Hairline, borderColor), shape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) {
            TbIcon(
                imageVector = Icons.Rounded.Check,
                contentDescription = null,
                modifier = Modifier.size(16.dp),
                tint = swatchCheckColor(swatchColor),
            )
        }
    }
}

private fun swatchCheckColor(color: Color): Color {
    val luminance = 0.2126f * color.red + 0.7152f * color.green + 0.0722f * color.blue
    return if (luminance > 0.62f) Color.Black else Color.White
}

@Composable
private fun TotalsCard(state: TimeboxxingScreenState) {
    TbCard {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            TbText(
                text = "Today",
                style = TbTheme.typography.title2,
            )
            TotalsRow(state)
        }
    }
}

@Composable
private fun TotalsRow(state: TimeboxxingScreenState) {
    Column(
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        SummaryMetric(
            label = "Captured",
            value = formatDuration(state.capturedMinutes),
        )
        SummaryMetric(
            label = "Billable",
            value = formatDuration(state.billableMinutes),
        )
        SummaryMetric(
            label = "Unassigned",
            value = formatDuration(state.unassignedUsageMinutes),
        )
        SummaryMetric(
            label = "Invoice estimate",
            value = formatMoney(state.invoiceTotalCents()),
        )
    }
}

@Composable
private fun SummaryMetric(
    label: String,
    value: String,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TbText(
            text = label,
            style = TbTheme.typography.body,
            color = TbTheme.colors.secondaryText,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        TbText(
            modifier = Modifier.padding(start = 12.dp),
            text = value,
            style = TbTheme.typography.title2,
            maxLines = 1,
        )
    }
}
