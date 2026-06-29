package com.timeboxxing.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.ExpandMore
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.timeboxxing.app.model.AiModelOption
import com.timeboxxing.app.model.AiProvider
import com.timeboxxing.app.model.AppearanceMode
import com.timeboxxing.app.state.TimeboxxingAction
import com.timeboxxing.app.state.TimeboxxingScreenState

@Composable
fun SettingsPane(
    state: TimeboxxingScreenState,
    onAction: (TimeboxxingAction) -> Unit,
    modifier: Modifier = Modifier,
) {
    val draft = state.settingsDraft
    val embeddingModels = state.settingsOptions.embeddingModels.filter { it.supports(draft.provider) }
    val semanticModels = state.settingsOptions.semanticModels.filter { it.supports(draft.provider) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp, vertical = 24.dp),
        verticalArrangement = Arrangement.spacedBy(18.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                TbText(
                    text = "Settings",
                    style = TbTheme.typography.largeTitle,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                TbText(
                    text = if (state.settingsLoading) "Loading" else activeProviderLabel(draft.provider),
                    style = TbTheme.typography.body,
                    color = TbTheme.colors.secondaryText,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            TbButton(
                onClick = { onAction(TimeboxxingAction.SaveSettings) },
                enabled = !state.settingsLoading && !state.settingsSaving && embeddingModels.isNotEmpty() && semanticModels.isNotEmpty(),
            ) {
                TbText(
                    text = if (state.settingsSaving) "Saving" else "Save",
                    style = TbTheme.typography.button,
                )
            }
        }

        state.settingsError?.let { message ->
            SettingsNotice(message = message, destructive = true)
        }
        state.settingsSavedMessage?.let { message ->
            SettingsNotice(message = message, destructive = false)
        }

        TbCard(
            modifier = Modifier.widthIn(max = 760.dp),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(18.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                TbText("Appearance", style = TbTheme.typography.headline)
                AppearanceModeSelector(
                    selectedMode = state.appearanceMode,
                    onModeChange = { onAction(TimeboxxingAction.UpdateAppearanceMode(it)) },
                )

                Spacer(modifier = Modifier.height(2.dp))

                TbText("Provider", style = TbTheme.typography.headline)
                ProviderSelector(
                    provider = draft.provider,
                    onProviderChange = { onAction(TimeboxxingAction.UpdateSettingsProvider(it)) },
                )

                when (draft.provider) {
                    AiProvider.OpenRouter -> {
                        TbTextField(
                            value = draft.openRouterBaseUrl,
                            onValueChange = { onAction(TimeboxxingAction.UpdateOpenRouterBaseUrl(it)) },
                            label = "OpenRouter base URL",
                            singleLine = true,
                        )
                        TbTextField(
                            value = draft.openRouterApiKey,
                            onValueChange = { onAction(TimeboxxingAction.UpdateOpenRouterApiKey(it)) },
                            label = "OpenRouter API key",
                            singleLine = true,
                        )
                    }

                    AiProvider.Ollama -> {
                        TbTextField(
                            value = draft.ollamaBaseUrl,
                            onValueChange = { onAction(TimeboxxingAction.UpdateOllamaBaseUrl(it)) },
                            label = "Ollama base URL",
                            singleLine = true,
                        )
                        TbTextField(
                            value = draft.ollamaApiKey,
                            onValueChange = { onAction(TimeboxxingAction.UpdateOllamaApiKey(it)) },
                            label = "Ollama bearer token",
                            singleLine = true,
                        )
                    }
                }

                Spacer(modifier = Modifier.height(2.dp))

                ModelMenu(
                    label = "Embedding model",
                    selectedId = draft.embeddingModelId,
                    models = embeddingModels,
                    provider = draft.provider,
                    onModelSelected = { onAction(TimeboxxingAction.UpdateSettingsEmbeddingModel(it)) },
                )

                ModelMenu(
                    label = "Semantic model",
                    selectedId = draft.semanticModelId,
                    models = semanticModels,
                    provider = draft.provider,
                    onModelSelected = { onAction(TimeboxxingAction.UpdateSettingsSemanticModel(it)) },
                )
            }
        }
    }
}

@Composable
private fun AppearanceModeSelector(
    selectedMode: AppearanceMode,
    onModeChange: (AppearanceMode) -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AppearanceMode.entries.forEach { option ->
            TbButton(
                onClick = { onModeChange(option) },
                variant = if (selectedMode == option) TbButtonVariant.Primary else TbButtonVariant.Secondary,
            ) {
                TbText(option.label, style = TbTheme.typography.button)
            }
        }
    }
}

@Composable
private fun ProviderSelector(
    provider: AiProvider,
    onProviderChange: (AiProvider) -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AiProvider.entries.forEach { option ->
            TbButton(
                onClick = { onProviderChange(option) },
                variant = if (provider == option) TbButtonVariant.Primary else TbButtonVariant.Secondary,
            ) {
                TbText(option.label, style = TbTheme.typography.button)
            }
        }
    }
}

private val AppearanceMode.label: String
    get() = when (this) {
        AppearanceMode.System -> "System"
        AppearanceMode.Light -> "Light"
        AppearanceMode.Dark -> "Dark"
    }

@Composable
private fun ModelMenu(
    label: String,
    selectedId: Long,
    models: List<AiModelOption>,
    provider: AiProvider,
    onModelSelected: (Long) -> Unit,
) {
    val selected = models.firstOrNull { it.id == selectedId }
    var expanded by remember(models, selectedId) { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        TbText(
            text = label,
            style = TbTheme.typography.label,
            color = TbTheme.colors.secondaryText,
        )
        TbMenu(
            expanded = expanded,
            onExpandedChange = { expanded = it },
            anchor = {
                TbButton(
                    modifier = Modifier.fillMaxWidth(),
                    onClick = { expanded = true },
                    enabled = models.isNotEmpty(),
                    variant = TbButtonVariant.Secondary,
                    contentPadding = PaddingValues(horizontal = 12.dp, vertical = 9.dp),
                ) {
                    Column(
                        modifier = Modifier.weight(1f),
                        verticalArrangement = Arrangement.spacedBy(2.dp),
                    ) {
                        TbText(
                            text = selected?.label ?: "No available models",
                            style = TbTheme.typography.button,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        selected?.slugFor(provider)?.takeIf { it.isNotBlank() }?.let { slug ->
                            TbText(
                                text = slug,
                                style = TbTheme.typography.caption,
                                color = TbTheme.colors.secondaryText,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                    TbIcon(
                        imageVector = Icons.Rounded.ExpandMore,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                    )
                }
            },
            panelContent = {
                models.forEach { model ->
                    TbMenuItem(
                        onClick = {
                            expanded = false
                            onModelSelected(model.id)
                        },
                    ) {
                        Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            TbText(
                                text = model.label,
                                style = TbTheme.typography.body,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            TbText(
                                text = model.slugFor(provider),
                                style = TbTheme.typography.caption,
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
}

@Composable
private fun SettingsNotice(
    message: String,
    destructive: Boolean,
) {
    TbSurface(
        modifier = Modifier.widthIn(max = 760.dp),
        color = if (destructive) TbTheme.colors.destructiveSubtle else TbTheme.colors.successSubtle,
        contentColor = if (destructive) TbTheme.colors.destructive else TbTheme.colors.success,
        shape = androidx.compose.foundation.shape.RoundedCornerShape(TbTheme.radii.control),
    ) {
        TbText(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 9.dp),
            text = message,
            style = TbTheme.typography.body,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

private val AiProvider.label: String
    get() = when (this) {
        AiProvider.OpenRouter -> "OpenRouter"
        AiProvider.Ollama -> "Ollama"
    }

private fun activeProviderLabel(provider: AiProvider): String =
    when (provider) {
        AiProvider.OpenRouter -> "OpenRouter"
        AiProvider.Ollama -> "Ollama"
    }

private fun AiModelOption.slugFor(provider: AiProvider): String =
    when (provider) {
        AiProvider.OpenRouter -> openRouterSlug
        AiProvider.Ollama -> ollamaSlug
    }
