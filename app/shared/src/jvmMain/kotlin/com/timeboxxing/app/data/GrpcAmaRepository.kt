package com.timeboxxing.app.data

import com.timeboxxing.app.model.AmaAnswer
import com.timeboxxing.app.model.AmaIndexState
import com.timeboxxing.app.model.AmaIndexStatus
import com.timeboxxing.app.model.AmaSource
import com.timeboxxing.sidecar.ama.v1.AmaServiceGrpcKt
import com.timeboxxing.sidecar.ama.v1.AskRequest
import com.timeboxxing.sidecar.ama.v1.AskResponse
import com.timeboxxing.sidecar.ama.v1.GetSemanticIndexStatusRequest
import com.timeboxxing.sidecar.ama.v1.SemanticIndexStatus
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
        indexStatus = if (hasSemanticIndexStatus()) semanticIndexStatus.toAmaIndexStatus() else null,
    )

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
        Status.Code.UNAVAILABLE -> "AMA sidecar is unavailable. Please try again in a moment."
        Status.Code.INTERNAL -> {
            val description = description.orEmpty().lowercase()
            if ("openrouter" in description || "embedding" in description || "semantic" in description) {
                "Semantic model unavailable. Please try again in a moment."
            } else {
                "AMA could not answer right now. Please try again in a moment."
            }
        }
        else -> "AMA could not answer right now. Please try again in a moment."
    }
