package com.timeboxxing.data.grpc

import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaArtifact
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaQueryKind
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.domain.model.AmaStructuredQuery
import com.timeboxxing.domain.model.AmaTimeWindow
import com.timeboxxing.domain.model.AmaUsageTimeline
import com.timeboxxing.domain.model.AmaUsageTimelineEvent
import com.timeboxxing.domain.repository.AmaRepository
import com.timeboxxing.sidecar.ama.v1.Artifact
import com.timeboxxing.sidecar.ama.v1.AmaServiceGrpcKt
import com.timeboxxing.sidecar.ama.v1.AskRequest
import com.timeboxxing.sidecar.ama.v1.AskResponse
import com.timeboxxing.sidecar.ama.v1.GetSemanticIndexStatusRequest
import com.timeboxxing.sidecar.ama.v1.QueryKind
import com.timeboxxing.sidecar.ama.v1.SemanticIndexStatus
import com.timeboxxing.sidecar.ama.v1.StructuredQuery
import com.timeboxxing.sidecar.ama.v1.TimeWindow
import com.timeboxxing.sidecar.ama.v1.UsageTimelineEvent
import io.grpc.Status
import io.grpc.StatusException
import io.grpc.StatusRuntimeException
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import java.util.concurrent.TimeUnit

class GrpcAmaRepository(
    target: String,
    private val channel: ManagedChannel = ManagedChannelBuilder.forTarget(target)
        .usePlaintext()
        .build(),
) : AmaRepository, AutoCloseable {
    private val stub = AmaServiceGrpcKt.AmaServiceCoroutineStub(channel)

    override suspend fun ask(question: String, maxSources: Int): AmaAnswer {
        val response = try {
            stub
                .withDeadlineAfter(75, TimeUnit.SECONDS)
                .ask(
                    AskRequest.newBuilder()
                        .setQuestion(question)
                        .setMaxSources(maxSources)
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        }

        return response.toAmaAnswer()
    }

    override suspend fun askStructured(query: AmaStructuredQuery): AmaAnswer {
        val response = try {
            stub
                .withDeadlineAfter(75, TimeUnit.SECONDS)
                .ask(
                    AskRequest.newBuilder()
                        .setStructuredQuery(query.toProto())
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        }

        return response.toAmaAnswer()
    }

    override suspend fun getSemanticIndexStatus(): AmaIndexStatus {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .getSemanticIndexStatus(GetSemanticIndexStatusRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toAmaErrorMessage(), error)
        }

        return response.toAmaIndexStatus()
    }

    override fun close() {
        channel.shutdownNow()
        channel.awaitTermination(1, TimeUnit.SECONDS)
    }
}

internal fun AskResponse.toAmaAnswer(): AmaAnswer =
    AmaAnswer(
        answer = answer,
        model = model,
        sources = sourcesList.map { source ->
            AmaSource(
                transitionEventId = source.transitionEventId,
                documentKey = source.documentKey,
                documentType = source.documentType,
                startedAtEpochMillis = if (source.hasStartedAt()) source.startedAt.toEpochMillis() else null,
                endedAtEpochMillis = if (source.hasEndedAt()) source.endedAt.toEpochMillis() else null,
                content = source.content,
                distance = source.distance,
            )
        },
        artifacts = artifactsList.mapNotNull { it.toAmaArtifact() },
        indexStatus = if (hasSemanticIndexStatus()) semanticIndexStatus.toAmaIndexStatus() else null,
    )

internal fun Artifact.toAmaArtifact(): AmaArtifact? =
    when {
        hasUsageTimeline() -> {
            val timeline = usageTimeline
            AmaUsageTimeline(
                periodLabel = timeline.periodLabel,
                startedAtEpochMillis = if (timeline.hasStartedAt()) timeline.startedAt.toEpochMillis() else null,
                endedAtEpochMillis = if (timeline.hasEndedAt()) timeline.endedAt.toEpochMillis() else null,
                timeZone = timeline.timezone,
                totalDurationSeconds = timeline.totalDurationSeconds,
                totalEventCount = timeline.totalEventCount,
                truncated = timeline.truncated,
                events = timeline.eventsList.map { it.toAmaUsageTimelineEvent() },
            )
        }
        else -> null
    }

private fun UsageTimelineEvent.toAmaUsageTimelineEvent(): AmaUsageTimelineEvent =
    AmaUsageTimelineEvent(
        transitionEventId = transitionEventId,
        title = title,
        sourceName = sourceName,
        sourceType = sourceType,
        startedAtEpochMillis = if (hasStartedAt()) startedAt.toEpochMillis() else null,
        endedAtEpochMillis = if (hasEndedAt()) endedAt.toEpochMillis() else null,
        durationSeconds = durationSeconds,
        applicationIdentifier = applicationIdentifier,
        applicationPath = applicationPath,
        urlHost = urlHost,
        idle = idle,
    )

private fun AmaStructuredQuery.toProto(): StructuredQuery {
    return StructuredQuery.newBuilder()
        .setKind(kind.toProto())
        .setWindow(window.toProto())
        .setLimit(limit)
        .setIncludeIdle(includeIdle)
        .setPeriodLabel(periodLabel)
        .build()
}

private fun AmaTimeWindow.toProto(): TimeWindow =
    TimeWindow.newBuilder()
        .setStartedAt(timestampFromEpochMillis(startedAtEpochMillis))
        .setEndedAt(timestampFromEpochMillis(endedAtEpochMillis))
        .build()

private fun AmaQueryKind.toProto(): QueryKind =
    when (this) {
        AmaQueryKind.Timeline -> QueryKind.QUERY_KIND_TIMELINE
    }

internal fun SemanticIndexStatus.toAmaIndexStatus(): AmaIndexStatus =
    AmaIndexStatus(
        state = state.toAmaIndexState(),
        completedEventCount = completedEventCount,
        indexedEventCount = indexedEventCount,
        pendingEventCount = pendingEventCount,
        backfillRunning = backfillRunning,
        message = message,
    )

private fun SemanticIndexStatus.State.toAmaIndexState(): AmaIndexState =
    when (this) {
        SemanticIndexStatus.State.READY -> AmaIndexState.Ready
        SemanticIndexStatus.State.INDEXING -> AmaIndexState.Indexing
        SemanticIndexStatus.State.EMPTY -> AmaIndexState.Empty
        SemanticIndexStatus.State.UNAVAILABLE -> AmaIndexState.Unavailable
        else -> AmaIndexState.Unknown
    }

private fun com.google.protobuf.Timestamp.toEpochMillis(): Long =
    seconds * 1000 + nanos / 1_000_000

internal fun StatusRuntimeException.toAmaErrorMessage(): String = status.toAmaErrorMessage()

internal fun StatusException.toAmaErrorMessage(): String = status.toAmaErrorMessage()

private fun Status.toAmaErrorMessage(): String =
    when (code) {
        Status.Code.DEADLINE_EXCEEDED -> "AMA is still catching up. Try again in a moment."
        Status.Code.FAILED_PRECONDITION,
        Status.Code.RESOURCE_EXHAUSTED -> description.safeProviderMessage() ?: "AMA is not configured. Check Settings and try again."
        Status.Code.UNAVAILABLE -> description.safeProviderMessage()
            ?: "AMA sidecar is unavailable. Please try again in a moment."
        Status.Code.INTERNAL -> {
            description.safeProviderMessage()
                ?: description.safeAmaMessage()
                ?: "AMA could not answer right now. Please try again in a moment."
        }
        else -> "AMA could not answer right now. Please try again in a moment."
    }

private fun String?.safeProviderMessage(): String? {
    val message = this?.trim().orEmpty()
    if (message.isBlank()) return null
    val safePrefixes = listOf(
        "OpenRouter ",
        "Ollama ",
        "Selected OpenRouter ",
        "Selected Ollama ",
    )
    return message.takeIf { safePrefixes.any(message::startsWith) }
}

private fun String?.safeAmaMessage(): String? {
    val message = this?.trim().orEmpty()
    if (message.isBlank()) return null
    return message.takeIf {
        it == "AMA could not answer right now. Please try again in a moment." ||
            it == "Semantic index is unavailable."
    }
}
