package com.timeboxxing.app.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.composeunstyled.TooltipHost

@Immutable
data class TbColors(
    val appBackground: Color,
    val surface: Color,
    val groupedSurface: Color,
    val elevatedSurface: Color,
    val text: Color,
    val secondaryText: Color,
    val tertiaryText: Color,
    val separator: Color,
    val accent: Color,
    val accentPressed: Color,
    val accentSubtle: Color,
    val accentText: Color,
    val success: Color,
    val successSubtle: Color,
    val warning: Color,
    val warningSubtle: Color,
    val destructive: Color,
    val destructiveSubtle: Color,
    val controlFill: Color,
    val controlFillHover: Color,
    val tooltipBackground: Color,
    val tooltipText: Color,
)

@Immutable
data class TbTypography(
    val largeTitle: TextStyle,
    val title: TextStyle,
    val title2: TextStyle,
    val headline: TextStyle,
    val body: TextStyle,
    val bodySmall: TextStyle,
    val caption: TextStyle,
    val label: TextStyle,
    val button: TextStyle,
)

@Immutable
data class TbRadii(
    val control: androidx.compose.ui.unit.Dp = 8.dp,
    val pill: androidx.compose.ui.unit.Dp = 999.dp,
    val card: androidx.compose.ui.unit.Dp = 8.dp,
    val panel: androidx.compose.ui.unit.Dp = 8.dp,
)

object TbTheme {
    val colors: TbColors
        @Composable get() = LocalTbColors.current

    val typography: TbTypography
        @Composable get() = LocalTbTypography.current

    val radii: TbRadii
        @Composable get() = LocalTbRadii.current
}

val TbLightColors = TbColors(
    appBackground = Color.White,
    surface = Color.White,
    groupedSurface = Color(0xFFF8F8F4),
    elevatedSurface = Color.White,
    text = Color(0xFF151515),
    secondaryText = Color(0xFF5F6258),
    tertiaryText = Color(0xFF8A8D80),
    separator = Color(0xFFE6E6DC),
    accent = Color(0xFF00FFEE),
    accentPressed = Color(0xFF00E2D4),
    accentSubtle = Color(0xFFE2FFFC),
    accentText = Color(0xFF151515),
    success = Color(0xFF16A34A),
    successSubtle = Color(0xFFEAF8EF),
    warning = Color(0xFFD97706),
    warningSubtle = Color(0xFFFFF5DF),
    destructive = Color(0xFFDC2626),
    destructiveSubtle = Color(0xFFFFECEB),
    controlFill = Color(0xFFF4F4ED),
    controlFillHover = Color(0xFFECECE2),
    tooltipBackground = Color(0xF21D1D1F),
    tooltipText = Color.White,
)

val TbDarkColors = TbColors(
    appBackground = Color.Black,
    surface = Color(0xFF050505),
    groupedSurface = Color(0xFF0A0A0A),
    elevatedSurface = Color(0xFF0F0F0E),
    text = Color(0xFFF5F5EA),
    secondaryText = Color(0xFFC7CAB5),
    tertiaryText = Color(0xFF8B8F7A),
    separator = Color(0xFF24251C),
    accent = Color(0xFF00FFEE),
    accentPressed = Color(0xFF6EFFF5),
    accentSubtle = Color(0xFF032B28),
    accentText = Color(0xFF050505),
    success = Color(0xFF5DD17E),
    successSubtle = Color(0xFF12281B),
    warning = Color(0xFFFFB85C),
    warningSubtle = Color(0xFF2E2111),
    destructive = Color(0xFFFF6B6B),
    destructiveSubtle = Color(0xFF351818),
    controlFill = Color(0xFF11110F),
    controlFillHover = Color(0xFF1A1A16),
    tooltipBackground = Color(0xF2F5F7FA),
    tooltipText = Color(0xFF151821),
)

private val LocalTbColors = staticCompositionLocalOf { TbLightColors }
private val LocalTbTypography = staticCompositionLocalOf { tbTypography(FontFamily.Default, TbLightColors) }
private val LocalTbRadii = staticCompositionLocalOf { TbRadii() }
val LocalTbContentColor = staticCompositionLocalOf { TbLightColors.text }

@Composable
fun TbTheme(
    fontFamily: FontFamily? = null,
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val colors = if (darkTheme) TbDarkColors else TbLightColors
    val typography = tbTypography(fontFamily ?: FontFamily.Default, colors)

    CompositionLocalProvider(
        LocalTbColors provides colors,
        LocalTbTypography provides typography,
        LocalTbRadii provides TbRadii(),
        LocalTbContentColor provides colors.text,
    ) {
        TooltipHost {
            content()
        }
    }
}

private fun tbTypography(fontFamily: FontFamily, colors: TbColors): TbTypography =
    TbTypography(
        largeTitle = TextStyle(
            fontFamily = fontFamily,
            fontSize = 24.sp,
            lineHeight = 30.sp,
            fontWeight = FontWeight.SemiBold,
            color = colors.text,
        ),
        title = TextStyle(
            fontFamily = fontFamily,
            fontSize = 19.sp,
            lineHeight = 25.sp,
            fontWeight = FontWeight.SemiBold,
            color = colors.text,
        ),
        title2 = TextStyle(
            fontFamily = fontFamily,
            fontSize = 15.sp,
            lineHeight = 20.sp,
            fontWeight = FontWeight.SemiBold,
            color = colors.text,
        ),
        headline = TextStyle(
            fontFamily = fontFamily,
            fontSize = 14.sp,
            lineHeight = 19.sp,
            fontWeight = FontWeight.Medium,
            color = colors.text,
        ),
        body = TextStyle(
            fontFamily = fontFamily,
            fontSize = 14.sp,
            lineHeight = 20.sp,
            fontWeight = FontWeight.Normal,
            color = colors.text,
        ),
        bodySmall = TextStyle(
            fontFamily = fontFamily,
            fontSize = 13.sp,
            lineHeight = 18.sp,
            fontWeight = FontWeight.Normal,
            color = colors.secondaryText,
        ),
        caption = TextStyle(
            fontFamily = fontFamily,
            fontSize = 12.sp,
            lineHeight = 16.sp,
            fontWeight = FontWeight.Medium,
            color = colors.secondaryText,
        ),
        label = TextStyle(
            fontFamily = fontFamily,
            fontSize = 12.sp,
            lineHeight = 16.sp,
            fontWeight = FontWeight.SemiBold,
            color = colors.secondaryText,
        ),
        button = TextStyle(
            fontFamily = fontFamily,
            fontSize = 13.sp,
            lineHeight = 17.sp,
            fontWeight = FontWeight.SemiBold,
            color = colors.text,
        ),
    )
