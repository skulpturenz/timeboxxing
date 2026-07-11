package com.timeboxxing.data.grpc

import com.google.protobuf.Timestamp
import com.timeboxxing.domain.model.AiModelOption
import com.timeboxxing.domain.model.AiModelOptions
import com.timeboxxing.domain.model.AiProvider
import com.timeboxxing.domain.model.AiSettings
import com.timeboxxing.domain.model.DatabaseMaintenanceStatus
import com.timeboxxing.domain.model.DatabasePruneCounts
import com.timeboxxing.domain.model.DatabasePruneRange
import com.timeboxxing.domain.model.DatabasePruneResult
import com.timeboxxing.domain.model.DatabaseVacuumResult
import com.timeboxxing.domain.repository.SettingsRepository
import com.timeboxxing.sidecar.settings.v1.AiProvider as AiProviderProto
import com.timeboxxing.sidecar.settings.v1.AiSettings as AiSettingsProto
import com.timeboxxing.sidecar.settings.v1.DatabaseMaintenanceStatus as DatabaseMaintenanceStatusProto
import com.timeboxxing.sidecar.settings.v1.GetAiSettingsRequest
import com.timeboxxing.sidecar.settings.v1.GetDatabaseMaintenanceStatusRequest
import com.timeboxxing.sidecar.settings.v1.ListModelOptionsRequest
import com.timeboxxing.sidecar.settings.v1.ModelOption
import com.timeboxxing.sidecar.settings.v1.PruneDatabaseRangeRequest
import com.timeboxxing.sidecar.settings.v1.PruneDatabaseRangeResponse
import com.timeboxxing.sidecar.settings.v1.SaveAiSettingsRequest
import com.timeboxxing.sidecar.settings.v1.SettingsServiceGrpcKt
import com.timeboxxing.sidecar.settings.v1.VacuumDatabaseRequest
import com.timeboxxing.sidecar.settings.v1.VacuumDatabaseResponse
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
            embeddingModels = response.modelsList.filter { it.embedding }.map { it.toAiModelOption() },
            semanticModels = response.modelsList.filter { it.semantic }.map { it.toAiModelOption() },
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
                        .setModelProviderBaseUrl(settings.modelProviderBaseUrl)
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

    override suspend fun getDatabaseMaintenanceStatus(): DatabaseMaintenanceStatus {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .getDatabaseMaintenanceStatus(GetDatabaseMaintenanceStatusRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return response.toDatabaseMaintenanceStatus()
    }

    override suspend fun pruneDatabaseRange(range: DatabasePruneRange): DatabasePruneResult {
        val response = try {
            stub
                .withDeadlineAfter(30, TimeUnit.SECONDS)
                .pruneDatabaseRange(
                    PruneDatabaseRangeRequest.newBuilder()
                        .setStartedAt(range.startedAtEpochMillis.toTimestamp())
                        .setEndedAt(range.endedAtEpochMillis.toTimestamp())
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return response.toDatabasePruneResult()
    }

    override suspend fun vacuumDatabase(): DatabaseVacuumResult {
        val response = try {
            stub
                .withDeadlineAfter(120, TimeUnit.SECONDS)
                .vacuumDatabase(VacuumDatabaseRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toSettingsErrorMessage(), error)
        }

        return response.toDatabaseVacuumResult()
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
        modelProviderBaseUrl = modelProviderBaseUrl,
        embeddingModelId = embeddingModelId,
        semanticModelId = semanticModelId,
        openRouterSecretExists = openrouterSecretExists,
        ollamaSecretExists = ollamaSecretExists,
    )

internal fun DatabaseMaintenanceStatusProto.toDatabaseMaintenanceStatus(): DatabaseMaintenanceStatus =
    DatabaseMaintenanceStatus(sizeBytes = sizeBytes)

internal fun PruneDatabaseRangeResponse.toDatabasePruneResult(): DatabasePruneResult =
    DatabasePruneResult(
        status = DatabaseMaintenanceStatus(sizeBytes = sizeBytes),
        counts = DatabasePruneCounts(
            ledgerItemsDeleted = ledgerItemsDeleted,
            ledgerItemTimelineEntriesDeleted = ledgerItemTimelineEntriesDeleted,
            timelineDeleted = timelineDeleted,
            foregroundProcessesDeleted = foregroundProcessesDeleted,
            foregroundProcessMetadataDeleted = foregroundProcessMetadataDeleted,
            timelineSemanticDocumentsDeleted = timelineSemanticDocumentsDeleted,
            timelineEmbeddingsDeleted = timelineEmbeddingsDeleted,
            applicationsDeleted = applicationsDeleted,
        ),
    )

internal fun VacuumDatabaseResponse.toDatabaseVacuumResult(): DatabaseVacuumResult =
    DatabaseVacuumResult(
        sizeBeforeBytes = sizeBeforeBytes,
        sizeAfterBytes = sizeAfterBytes,
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

private fun Long.toTimestamp(): Timestamp =
    Timestamp.newBuilder()
        .setSeconds(this / 1_000L)
        .setNanos(((this % 1_000L) * 1_000_000L).toInt())
        .build()

internal fun StatusRuntimeException.toSettingsErrorMessage(): String = status.toSettingsErrorMessage()

internal fun StatusException.toSettingsErrorMessage(): String = status.toSettingsErrorMessage()

private fun Status.toSettingsErrorMessage(): String =
    when (code) {
        Status.Code.UNAVAILABLE -> "Settings sidecar is unavailable. Please try again in a moment."
        Status.Code.INVALID_ARGUMENT -> description ?: "Settings are invalid."
        else -> "Settings could not be loaded right now. Please try again in a moment."
    }
