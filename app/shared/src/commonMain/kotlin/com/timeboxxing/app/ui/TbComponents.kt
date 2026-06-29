package com.timeboxxing.app.ui

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicText
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.input.TextFieldLineLimits
import androidx.compose.foundation.text.input.rememberTextFieldState
import androidx.compose.foundation.text.input.setTextAndPlaceCursorAtEnd
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.composeunstyled.AnchorAlignment
import com.composeunstyled.AnchorSide
import com.composeunstyled.CheckedIndicator
import com.composeunstyled.DropdownMenuPanel
import com.composeunstyled.DropdownMenuPanelScope
import com.composeunstyled.Tab
import com.composeunstyled.TabList
import com.composeunstyled.TextInput
import com.composeunstyled.TooltipPanel
import com.composeunstyled.UnstyledButton
import com.composeunstyled.UnstyledCheckbox
import com.composeunstyled.UnstyledDropdownMenu
import com.composeunstyled.UnstyledDropdownMenuItem
import com.composeunstyled.UnstyledHorizontalSeparator
import com.composeunstyled.UnstyledTabGroup
import com.composeunstyled.UnstyledTextField
import com.composeunstyled.UnstyledTooltip
import kotlinx.coroutines.flow.collectLatest

enum class TbButtonVariant {
    Primary,
    Secondary,
    Ghost,
    Destructive,
}

@Composable
fun TbText(
    text: String,
    modifier: Modifier = Modifier,
    style: TextStyle = TbTheme.typography.body,
    color: Color = LocalTbContentColor.current,
    maxLines: Int = Int.MAX_VALUE,
    overflow: TextOverflow = TextOverflow.Clip,
    onTextLayout: ((TextLayoutResult) -> Unit)? = null,
) {
    BasicText(
        text = text,
        modifier = modifier,
        style = style.copy(color = color),
        maxLines = maxLines,
        overflow = overflow,
        onTextLayout = onTextLayout,
    )
}

@Composable
fun TbSurface(
    modifier: Modifier = Modifier,
    color: Color = TbTheme.colors.surface,
    shape: RoundedCornerShape = RoundedCornerShape(0.dp),
    border: BorderStroke? = null,
    shadowElevation: Dp = 0.dp,
    contentColor: Color = TbTheme.colors.text,
    contentAlignment: Alignment = Alignment.TopStart,
    content: @Composable BoxScope.() -> Unit,
) {
    val decorated = modifier
        .then(if (shadowElevation > 0.dp) Modifier.shadow(shadowElevation, shape) else Modifier)
        .clip(shape)
        .background(color)
        .then(if (border != null) Modifier.border(border, shape) else Modifier)

    CompositionLocalProvider(LocalTbContentColor provides contentColor) {
        Box(
            modifier = decorated,
            contentAlignment = contentAlignment,
            content = content,
        )
    }
}

@Composable
fun TbCard(
    modifier: Modifier = Modifier,
    color: Color = TbTheme.colors.elevatedSurface,
    borderColor: Color = TbTheme.colors.separator,
    shadowElevation: Dp = 0.dp,
    content: @Composable BoxScope.() -> Unit,
) {
    TbSurface(
        modifier = modifier,
        color = color,
        shape = RoundedCornerShape(TbTheme.radii.card),
        border = BorderStroke(Dp.Hairline, borderColor),
        shadowElevation = shadowElevation,
        content = content,
    )
}

@Composable
fun TbButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    variant: TbButtonVariant = TbButtonVariant.Primary,
    contentPadding: PaddingValues = PaddingValues(horizontal = 13.dp, vertical = 7.dp),
    content: @Composable RowScope.() -> Unit,
) {
    val colors = TbTheme.colors
    val shape = RoundedCornerShape(TbTheme.radii.control)
    val interactionSource = remember { MutableInteractionSource() }
    val hovered by interactionSource.collectIsHoveredAsState()
    val pressed by interactionSource.collectIsPressedAsState()
    val background = when (variant) {
        TbButtonVariant.Primary -> when {
            pressed -> colors.accentPressed
            hovered -> colors.accentPressed
            else -> colors.accent
        }
        TbButtonVariant.Secondary -> when {
            pressed -> colors.controlFillHover
            hovered -> colors.controlFillHover
            else -> colors.controlFill
        }
        TbButtonVariant.Ghost -> when {
            pressed -> colors.controlFillHover
            hovered -> colors.controlFill
            else -> Color.Transparent
        }
        TbButtonVariant.Destructive -> if (pressed) colors.destructive.copy(alpha = 0.84f) else colors.destructiveSubtle
    }
    val contentColor = when (variant) {
        TbButtonVariant.Primary -> colors.accentText
        TbButtonVariant.Secondary -> colors.text
        TbButtonVariant.Ghost -> if (hovered || pressed) colors.text else colors.secondaryText
        TbButtonVariant.Destructive -> if (pressed) colors.accentText else colors.destructive
    }
    val border = when (variant) {
        TbButtonVariant.Secondary -> BorderStroke(Dp.Hairline, colors.separator)
        TbButtonVariant.Ghost -> null
        TbButtonVariant.Primary, TbButtonVariant.Destructive -> null
    }

    UnstyledButton(
        onClick = onClick,
        enabled = enabled,
        interactionSource = interactionSource,
        modifier = modifier
            .heightIn(min = 32.dp, max = 48.dp)
            .clip(shape)
            .background(if (enabled) background else colors.controlFill.copy(alpha = 0.62f))
            .then(if (border != null) Modifier.border(border, shape) else Modifier),
        contentPadding = contentPadding,
    ) {
        CompositionLocalProvider(
            LocalTbContentColor provides if (enabled) contentColor else colors.tertiaryText,
        ) {
            Row(
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
                content = content,
            )
        }
    }
}

@Composable
fun TbIcon(
    imageVector: ImageVector,
    contentDescription: String?,
    modifier: Modifier = Modifier,
    tint: Color = LocalTbContentColor.current,
) {
    Image(
        painter = rememberVectorPainter(imageVector),
        contentDescription = contentDescription,
        modifier = modifier.size(18.dp),
        colorFilter = ColorFilter.tint(tint),
    )
}

@Composable
fun TbIconButton(
    icon: ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    variant: TbButtonVariant = TbButtonVariant.Secondary,
) {
    TbTooltip(text = contentDescription) {
        TbButton(
            modifier = modifier.size(32.dp),
            onClick = onClick,
            enabled = enabled,
            variant = variant,
            contentPadding = PaddingValues(0.dp),
        ) {
            TbIcon(
                imageVector = icon,
                contentDescription = contentDescription,
                modifier = Modifier.size(17.dp),
            )
        }
    }
}

@Composable
fun TbCheckbox(
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
    accessibilityLabel: String? = null,
) {
    val colors = TbTheme.colors
    val checkColor = colors.accentText
    UnstyledCheckbox(
        checked = checked,
        onCheckedChange = onCheckedChange,
        accessibilityLabel = accessibilityLabel,
        modifier = modifier
            .size(18.dp)
            .clip(RoundedCornerShape(5.dp))
            .background(if (checked) colors.accent else colors.surface)
            .border(
                BorderStroke(Dp.Hairline, if (checked) colors.accent else colors.separator),
                RoundedCornerShape(5.dp),
            ),
    ) {
        CheckedIndicator(modifier = Modifier.size(18.dp)) {
            Canvas(modifier = Modifier.size(18.dp)) {
                drawLine(
                    color = checkColor,
                    strokeWidth = 2.2f,
                    start = Offset(size.width * 0.28f, size.height * 0.52f),
                    end = Offset(size.width * 0.43f, size.height * 0.68f),
                )
                drawLine(
                    color = checkColor,
                    strokeWidth = 2.2f,
                    start = Offset(size.width * 0.43f, size.height * 0.68f),
                    end = Offset(size.width * 0.74f, size.height * 0.34f),
                )
            }
        }
    }
}

@Composable
fun TbTextField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    label: String? = null,
    singleLine: Boolean = false,
    minLines: Int = 1,
    maxLines: Int = Int.MAX_VALUE,
    suffix: (@Composable () -> Unit)? = null,
    keyboardType: KeyboardType = KeyboardType.Text,
    inputModifier: Modifier = Modifier,
) {
    val colors = TbTheme.colors
    val state = rememberTextFieldState(initialText = value)

    LaunchedEffect(value) {
        if (state.text.toString() != value) {
            state.setTextAndPlaceCursorAtEnd(value)
        }
    }
    LaunchedEffect(state, onValueChange) {
        snapshotFlow { state.text.toString() }.collectLatest { text ->
            if (text != value) {
                onValueChange(text)
            }
        }
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        if (label != null) {
            TbText(
                text = label,
                style = TbTheme.typography.label,
                color = colors.secondaryText,
            )
        }
        TbSurface(
            shape = RoundedCornerShape(TbTheme.radii.control),
            color = colors.elevatedSurface,
            border = BorderStroke(Dp.Hairline, colors.separator),
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(
                        min = if (singleLine) 34.dp else 74.dp,
                        max = if (singleLine) 34.dp else 132.dp,
                    )
                    .padding(horizontal = 10.dp, vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = if (singleLine) Alignment.CenterVertically else Alignment.Top,
            ) {
                UnstyledTextField(
                    modifier = Modifier
                        .weight(1f)
                        .then(inputModifier),
                    state = state,
                    cursorBrush = SolidColor(colors.accent),
                    selectionColors = androidx.compose.foundation.text.selection.TextSelectionColors(
                        handleColor = colors.accent,
                        backgroundColor = colors.accent.copy(alpha = 0.24f),
                    ),
                    textStyle = TbTheme.typography.body,
                    textColor = colors.text,
                    keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
                    lineLimits = if (singleLine) {
                        TextFieldLineLimits.SingleLine
                    } else {
                        TextFieldLineLimits.MultiLine(minLines, maxLines)
                    },
                ) {
                    TextInput(modifier = Modifier.fillMaxWidth())
                }
                if (suffix != null) {
                    Box(modifier = Modifier.padding(top = if (singleLine) 0.dp else 1.dp)) {
                        suffix()
                    }
                }
            }
        }
    }
}

@Composable
fun TbMenu(
    expanded: Boolean,
    onExpandedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
    panelContent: @Composable DropdownMenuPanelScope.() -> Unit,
    anchor: @Composable () -> Unit,
) {
    val colors = TbTheme.colors
    UnstyledDropdownMenu(
        expanded = expanded,
        onExpandedChange = onExpandedChange,
        modifier = modifier,
        side = AnchorSide.Bottom,
        alignment = AnchorAlignment.Start,
        sideOffset = 6.dp,
        panel = {
            DropdownMenuPanel(
                modifier = Modifier
                    .shadow(12.dp, RoundedCornerShape(TbTheme.radii.card))
                    .clip(RoundedCornerShape(TbTheme.radii.card))
                    .background(colors.surface)
                    .border(BorderStroke(Dp.Hairline, colors.separator), RoundedCornerShape(TbTheme.radii.card))
                    .padding(5.dp),
            ) {
                panelContent()
            }
        },
        anchor = anchor,
    )
}

@Composable
fun DropdownMenuPanelScope.TbMenuItem(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable RowScope.() -> Unit,
) {
    UnstyledDropdownMenuItem(
        onClick = onClick,
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(7.dp))
            .padding(horizontal = 10.dp, vertical = 8.dp),
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(10.dp),
            verticalAlignment = Alignment.CenterVertically,
            content = content,
        )
    }
}

@Composable
fun TbTabs(
    selectedIndex: Int,
    labels: List<String>,
    onSelectedIndexChange: (Int) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = TbTheme.colors
    UnstyledTabGroup(
        selectedTab = selectedIndex,
        onSelectedTabChange = onSelectedIndexChange,
        tabs = labels.indices.toList(),
        modifier = modifier,
    ) {
        TabList(
            modifier = Modifier
                .fillMaxWidth()
                .background(colors.surface)
                .padding(horizontal = 16.dp, vertical = 8.dp),
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(9.dp))
                    .background(colors.groupedSurface)
                    .padding(2.dp),
            ) {
                labels.forEachIndexed { index, label ->
                    Tab(
                        key = index,
                        modifier = Modifier
                            .weight(1f)
                            .clip(RoundedCornerShape(7.dp))
                            .background(if (index == selectedIndex) colors.surface else Color.Transparent)
                            .padding(vertical = 7.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        TbText(
                            text = label,
                            style = TbTheme.typography.button,
                            color = if (selected) colors.text else colors.secondaryText,
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun TbHorizontalDivider(
    modifier: Modifier = Modifier,
    color: Color = TbTheme.colors.separator,
) {
    UnstyledHorizontalSeparator(
        color = color,
        modifier = modifier,
        thickness = Dp.Hairline,
    )
}

@Composable
fun TbVerticalDivider(
    modifier: Modifier = Modifier,
    color: Color = TbTheme.colors.separator,
) {
    com.composeunstyled.UnstyledVerticalSeparator(
        color = color,
        modifier = modifier,
        thickness = Dp.Hairline,
    )
}

@Composable
fun TbPill(
    label: String,
    modifier: Modifier = Modifier,
    emphasized: Boolean = false,
) {
    val colors = TbTheme.colors
    TbSurface(
        modifier = modifier,
        shape = RoundedCornerShape(TbTheme.radii.pill),
        color = if (emphasized) colors.accentSubtle else colors.controlFill,
        contentColor = if (emphasized) colors.text else colors.secondaryText,
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 5.dp),
            text = label,
            style = TbTheme.typography.label,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
fun TbBadge(
    label: String,
    modifier: Modifier = Modifier,
    color: Color = TbTheme.colors.success,
    background: Color = TbTheme.colors.successSubtle,
) {
    TbSurface(
        modifier = modifier,
        shape = RoundedCornerShape(TbTheme.radii.pill),
        color = background,
        contentColor = color,
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
            text = label,
            style = TbTheme.typography.label,
            maxLines = 1,
        )
    }
}

@Composable
fun TbTooltip(
    text: String,
    enabled: Boolean = true,
    side: AnchorSide = AnchorSide.Top,
    alignment: AnchorAlignment = AnchorAlignment.Center,
    sideOffset: Dp = 8.dp,
    alignmentOffset: Dp = 0.dp,
    anchor: @Composable () -> Unit,
) {
    UnstyledTooltip(
        enabled = enabled,
        side = side,
        alignment = alignment,
        sideOffset = sideOffset,
        alignmentOffset = alignmentOffset,
        hoverDelayMillis = 180L,
        panel = {
            TooltipPanel { placement ->
                TbTooltipPanel(
                    text = text,
                    side = side,
                    crossAxisAdjustment = placement.positionAdjustment,
                )
            }
        },
        anchor = anchor,
    )
}

@Composable
private fun TbTooltipPanel(
    text: String,
    side: AnchorSide,
    crossAxisAdjustment: androidx.compose.ui.unit.IntOffset,
) {
    val colors = TbTheme.colors
    val density = LocalDensity.current
    val pointerXOffset = with(density) { (-crossAxisAdjustment.x).toDp() }
    val pointerYOffset = with(density) { (-crossAxisAdjustment.y).toDp() }

    when (side) {
        AnchorSide.Top -> Column(horizontalAlignment = Alignment.CenterHorizontally) {
            TbTooltipBubble(
                text = text,
                background = colors.tooltipBackground,
                contentColor = colors.tooltipText,
            )
            TbTooltipPointer(
                side = side,
                color = colors.tooltipBackground,
                modifier = Modifier.offset(x = pointerXOffset),
            )
        }

        AnchorSide.Bottom -> Column(horizontalAlignment = Alignment.CenterHorizontally) {
            TbTooltipPointer(
                side = side,
                color = colors.tooltipBackground,
                modifier = Modifier.offset(x = pointerXOffset),
            )
            TbTooltipBubble(
                text = text,
                background = colors.tooltipBackground,
                contentColor = colors.tooltipText,
            )
        }

        AnchorSide.Start -> Row(verticalAlignment = Alignment.CenterVertically) {
            TbTooltipBubble(
                text = text,
                background = colors.tooltipBackground,
                contentColor = colors.tooltipText,
            )
            TbTooltipPointer(
                side = side,
                color = colors.tooltipBackground,
                modifier = Modifier.offset(y = pointerYOffset),
            )
        }

        AnchorSide.End -> Row(verticalAlignment = Alignment.CenterVertically) {
            TbTooltipPointer(
                side = side,
                color = colors.tooltipBackground,
                modifier = Modifier.offset(y = pointerYOffset),
            )
            TbTooltipBubble(
                text = text,
                background = colors.tooltipBackground,
                contentColor = colors.tooltipText,
            )
        }
    }
}

@Composable
private fun TbTooltipBubble(
    text: String,
    background: Color,
    contentColor: Color,
) {
    TbSurface(
        modifier = Modifier.widthIn(max = 320.dp),
        color = background,
        shape = RoundedCornerShape(7.dp),
        shadowElevation = 8.dp,
        contentColor = contentColor,
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 9.dp, vertical = 6.dp),
            text = text,
            style = TbTheme.typography.caption,
            color = contentColor,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun TbTooltipPointer(
    side: AnchorSide,
    color: Color,
    modifier: Modifier = Modifier,
) {
    Canvas(modifier = modifier.size(8.dp)) {
        val path = Path()
        when (side) {
            AnchorSide.Top -> {
                path.moveTo(0f, 0f)
                path.lineTo(size.width / 2f, size.height)
                path.lineTo(size.width, 0f)
            }

            AnchorSide.Bottom -> {
                path.moveTo(0f, size.height)
                path.lineTo(size.width / 2f, 0f)
                path.lineTo(size.width, size.height)
            }

            AnchorSide.Start -> {
                path.moveTo(0f, 0f)
                path.lineTo(size.width, size.height / 2f)
                path.lineTo(0f, size.height)
            }

            AnchorSide.End -> {
                path.moveTo(size.width, 0f)
                path.lineTo(0f, size.height / 2f)
                path.lineTo(size.width, size.height)
            }
        }
        path.close()
        drawPath(path = path, color = color)
    }
}
