package com.timeboxxing.app.data

import com.timeboxxing.app.model.AiModelOption
import com.timeboxxing.app.model.AiModelOptions
import com.timeboxxing.app.model.AiProvider
import com.timeboxxing.app.model.AiSettings
import com.timeboxxing.sidecar.settings.v1.AiProvider as AiProviderProto
import com.timeboxxing.sidecar.settings.v1.AiSettings as AiSettingsProto
import com.timeboxxing.sidecar.settings.v1.GetAiSettingsRequest
import com.timeboxxing.sidecar.settings.v1.ListModelOptionsRequest
import com.timeboxxing.sidecar.settings.v1.ModelOption
import com.timeboxxing.sidecar.settings.v1.SaveAiSettingsRequest
import com.timeboxxing.sidecar.settings.v1.SettingsServiceGrpcKt
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import io.grpc.Status
import io.grpc.StatusException
import io.grpc.StatusRuntimeException
import java.util.concurrent.TimeUnit

class GrpcSettingsRepository(
    target: String,
    private val channel: ManagedChannel = ManagedChannelBuilder.forTarget(target)
        .usePlaintext()
        .build(),
) : SettingsRepository, AutoCloseable {
    private val stub = SettingsServiceGrpcKt.SettingsServiceCoroutineStub(channel)

    override suspend fun listModelOptions(): AiModelOptions {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listModelOptions(ListModelOptionsRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return AiModelOptions(
            embeddingModels = response.embeddingModelsList.map { it.toAiModelOption() },
            semanticModels = response.semanticModelsList.map { it.toAiModelOption() },
        )
    }

    override suspend fun getAiSettings(): AiSettings {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .getAiSettings(GetAiSettingsRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return response.toAiSettings()
    }

    override suspend fun saveAiSettings(settings: AiSettings): AiSettings {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .saveAiSettings(
                    SaveAiSettingsRequest.newBuilder()
                        .setProvider(settings.provider.toProto())
                        .setOpenrouterBaseUrl(settings.openRouterBaseUrl)
                        .setOllamaBaseUrl(settings.ollamaBaseUrl)
                        .setEmbeddingModelId(settings.embeddingModelId)
                        .setSemanticModelId(settings.semanticModelId)
                        .setOpenrouterSecretExists(settings.openRouterSecretExists || settings.openRouterApiKey.isNotBlank())
                        .setOllamaSecretExists(settings.ollamaSecretExists || settings.ollamaApiKey.isNotBlank())
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return response.toAiSettings().copy(
            openRouterApiKey = settings.openRouterApiKey,
            ollamaApiKey = settings.ollamaApiKey,
        )
    }

    override fun close() {
        channel.shutdownNow()
        channel.awaitTermination(1, TimeUnit.SECONDS)
    }
}

private fun ModelOption.toAiModelOption(): AiModelOption =
    AiModelOption(
        id = id,
        openRouterSlug = openrouterSlug,
        ollamaSlug = ollamaSlug,
        label = label,
    )

private fun AiSettingsProto.toAiSettings(): AiSettings =
    AiSettings(
        provider = provider.toAiProvider(),
        openRouterBaseUrl = openrouterBaseUrl,
        ollamaBaseUrl = ollamaBaseUrl,
        embeddingModelId = embeddingModelId,
        semanticModelId = semanticModelId,
        openRouterSecretExists = openrouterSecretExists,
        ollamaSecretExists = ollamaSecretExists,
    )

private fun AiProviderProto.toAiProvider(): AiProvider =
    when (this) {
        AiProviderProto.OLLAMA -> AiProvider.Ollama
        else -> AiProvider.OpenRouter
    }

private fun AiProvider.toProto(): AiProviderProto =
    when (this) {
        AiProvider.OpenRouter -> AiProviderProto.OPENROUTER
        AiProvider.Ollama -> AiProviderProto.OLLAMA
    }

internal fun StatusRuntimeException.toSettingsErrorMessage(): String = status.toSettingsErrorMessage()

internal fun StatusException.toSettingsErrorMessage(): String = status.toSettingsErrorMessage()

private fun Status.toSettingsErrorMessage(): String =
    when (code) {
        Status.Code.UNAVAILABLE -> "Settings sidecar is unavailable. Please try again in a moment."
        Status.Code.INVALID_ARGUMENT -> description ?: "Settings are invalid."
        else -> "Settings could not be loaded right now. Please try again in a moment."
    }
