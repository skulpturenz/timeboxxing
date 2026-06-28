package com.timeboxxing.app.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import com.timeboxxing.app.model.Project
import com.timeboxxing.app.model.formatDuration
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingScreenState

@Composable
fun ProjectSummaryRail(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
    headerAction: @Composable (() -> Unit)? = null,
) {
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
            headerAction?.invoke()
        }

        ProjectTotals(state)

        TbHorizontalDivider()

        TotalsCard(state)

        TbButton(
            modifier = Modifier.fillMaxWidth(),
            onClick = { onAction(TimeboxxingAction.ConfirmPrimaryAction) },
        ) {
            TbIcon(
                imageVector = Icons.Rounded.Check,
                contentDescription = null,
                modifier = Modifier
                    .padding(end = 6.dp)
                    .height(17.dp),
            )
            TbText(state.primaryActionLabel, style = TbTheme.typography.button)
        }
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
                TbButton(onClick = { onAction(TimeboxxingAction.ConfirmPrimaryAction) }) {
                    TbIcon(
                        imageVector = Icons.Rounded.Check,
                        contentDescription = null,
                        modifier = Modifier
                            .padding(end = 6.dp)
                            .height(17.dp),
                    )
                    TbText(state.primaryActionLabel, style = TbTheme.typography.button)
                }
            }
            TotalsRow(state)
            ProjectTotals(state)
        }
    }
}

@Composable
private fun ProjectTotals(state: TimeboxxingScreenState) {
    Column(
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        state.projects.forEach { project ->
            ProjectTotalRow(
                project = project,
                minutes = state.minutesForProject(project.id),
                maxMinutes = state.entries.maxOfOrNull { it.durationMinutes }?.coerceAtLeast(1) ?: 1,
            )
        }
    }
}

@Composable
private fun ProjectTotalRow(
    project: Project,
    minutes: Int,
    maxMinutes: Int,
) {
    val fraction = (minutes.toFloat() / maxMinutes.toFloat()).coerceIn(0.08f, 1f)

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
            }

            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(6.dp)
                    .background(TbTheme.colors.groupedSurface, RoundedCornerShape(999.dp)),
            ) {
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
