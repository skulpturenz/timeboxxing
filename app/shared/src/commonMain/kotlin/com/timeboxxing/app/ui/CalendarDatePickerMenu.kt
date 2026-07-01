package com.timeboxxing.app.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.timeboxxing.data.time.currentCalendarDate
import com.timeboxxing.domain.model.CalendarDate
import com.timeboxxing.domain.model.WeekdayShortLabels
import com.timeboxxing.domain.model.calendarMonthGrid
import com.timeboxxing.domain.model.monthYearLabel
import com.timeboxxing.domain.model.plusMonths
import com.timeboxxing.domain.model.startOfMonth

@Composable
fun CalendarDatePickerMenu(
    selectedDate: CalendarDate?,
    label: String,
    onDateSelected: (CalendarDate) -> Unit,
    modifier: Modifier = Modifier,
    textStyle: TextStyle = TbTheme.typography.button,
    enabled: Boolean = true,
) {
    val todayDate = currentCalendarDate()
    var expanded by remember { mutableStateOf(false) }
    var visibleMonth by remember(selectedDate, todayDate) {
        mutableStateOf((selectedDate ?: todayDate).startOfMonth())
    }

    TbMenu(
        expanded = expanded && enabled,
        onExpandedChange = { expanded = if (enabled) it else false },
        anchor = {
            TbButton(
                modifier = modifier,
                onClick = { expanded = true },
                enabled = enabled,
                variant = TbButtonVariant.Secondary,
                contentPadding = PaddingValues(horizontal = 14.dp, vertical = 9.dp),
            ) {
                TbText(
                    modifier = Modifier.weight(1f),
                    text = label,
                    style = textStyle,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbIcon(
                    imageVector = Icons.Rounded.ExpandMore,
                    contentDescription = null,
                    modifier = Modifier
                        .padding(start = 8.dp)
                        .size(16.dp),
                )
            }
        },
        panelContent = {
            CalendarPanel(
                visibleMonth = visibleMonth,
                selectedDate = selectedDate,
                todayDate = todayDate,
                onPreviousMonth = { visibleMonth = visibleMonth.plusMonths(-1) },
                onNextMonth = { visibleMonth = visibleMonth.plusMonths(1) },
                onDateSelected = { date ->
                    expanded = false
                    onDateSelected(date)
                },
            )
        },
    )
}

@Composable
private fun CalendarPanel(
    visibleMonth: CalendarDate,
    selectedDate: CalendarDate?,
    todayDate: CalendarDate,
    onPreviousMonth: () -> Unit,
    onNextMonth: () -> Unit,
    onDateSelected: (CalendarDate) -> Unit,
) {
    Column(
        modifier = Modifier
            .width(286.dp)
            .padding(6.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TbIconButton(
                icon = Icons.AutoMirrored.Rounded.KeyboardArrowLeft,
                contentDescription = "Previous month",
                onClick = onPreviousMonth,
                variant = TbButtonVariant.Ghost,
            )
            TbText(
                text = visibleMonth.monthYearLabel(),
                style = TbTheme.typography.headline,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            TbIconButton(
                icon = Icons.AutoMirrored.Rounded.KeyboardArrowRight,
                contentDescription = "Next month",
                onClick = onNextMonth,
                variant = TbButtonVariant.Ghost,
            )
        }

        CalendarWeekdayRow()

        calendarMonthGrid(visibleMonth).chunked(7).forEach { week ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                week.forEach { date ->
                    CalendarDayButton(
                        date = date,
                        visibleMonth = visibleMonth,
                        selected = date == selectedDate,
                        today = date == todayDate,
                        onClick = { onDateSelected(date) },
                    )
                }
            }
        }
    }
}

@Composable
private fun CalendarWeekdayRow() {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        WeekdayShortLabels.forEach { label ->
            Box(
                modifier = Modifier.size(34.dp),
                contentAlignment = Alignment.Center,
            ) {
                TbText(
                    text = label,
                    style = TbTheme.typography.caption.copy(textAlign = TextAlign.Center),
                    color = TbTheme.colors.tertiaryText,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun CalendarDayButton(
    date: CalendarDate,
    visibleMonth: CalendarDate,
    selected: Boolean,
    today: Boolean,
    onClick: () -> Unit,
) {
    val colors = TbTheme.colors
    val inVisibleMonth = date.year == visibleMonth.year && date.month == visibleMonth.month
    val interactionSource = remember { MutableInteractionSource() }
    val hovered by interactionSource.collectIsHoveredAsState()
    val pressed by interactionSource.collectIsPressedAsState()
    val shape = RoundedCornerShape(TbTheme.radii.control)
    val backgroundColor = when {
        selected -> colors.accent
        today -> colors.accentSubtle
        pressed -> colors.controlFillHover
        hovered -> colors.controlFill
        else -> Color.Transparent
    }
    val border = if (today && !selected) BorderStroke(1.dp, colors.accent) else null
    val textColor = when {
        selected -> colors.accentText
        today -> colors.text
        inVisibleMonth -> colors.text
        else -> colors.tertiaryText
    }

    TbSurface(
        modifier = Modifier
            .size(34.dp)
            .clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick,
            ),
        color = backgroundColor,
        shape = shape,
        border = border,
        contentColor = textColor,
        contentAlignment = Alignment.Center,
    ) {
        TbText(
            text = date.dayOfMonth.toString(),
            style = TbTheme.typography.label.copy(
                textAlign = TextAlign.Center,
                fontWeight = if (today || selected) FontWeight.SemiBold else FontWeight.Medium,
            ),
            color = textColor,
            maxLines = 1,
        )
    }
}
