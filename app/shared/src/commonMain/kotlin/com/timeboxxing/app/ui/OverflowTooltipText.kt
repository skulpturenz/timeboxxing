package com.timeboxxing.app.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.isSpecified
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow

@Composable
fun OverflowTooltipText(
    text: String,
    modifier: Modifier = Modifier,
    style: TextStyle,
    color: Color = Color.Unspecified,
    fontWeight: FontWeight? = null,
    maxLines: Int = Int.MAX_VALUE,
) {
    var hasOverflow by remember(text, maxLines) { mutableStateOf(false) }
    val textStyle = if (fontWeight == null) style else style.copy(fontWeight = fontWeight)
    val resolvedColor = if (color.isSpecified) color else LocalTbContentColor.current

    val textContent: @Composable () -> Unit = {
        TbText(
            modifier = modifier,
            text = text,
            style = textStyle,
            color = resolvedColor,
            maxLines = maxLines,
            overflow = TextOverflow.Ellipsis,
            onTextLayout = { result ->
                hasOverflow = result.hasVisualOverflow
            },
        )
    }

    if (hasOverflow) {
        TbTooltip(text = text) {
            textContent()
        }
    } else {
        textContent()
    }
}
