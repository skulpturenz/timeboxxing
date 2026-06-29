package com.timeboxxing.app.data

import com.google.protobuf.Timestamp
import com.timeboxxing.app.model.AmaAppUsageChart
import com.timeboxxing.sidecar.ama.v1.AppUsageBucket
import com.timeboxxing.sidecar.ama.v1.AppUsageChart
import com.timeboxxing.sidecar.ama.v1.Artifact
import com.timeboxxing.sidecar.ama.v1.AskResponse
import com.timeboxxing.sidecar.ama.v1.SemanticIndexStatus
import com.timeboxxing.sidecar.ama.v1.Source
import io.grpc.Status
import io.grpc.StatusRuntimeException
import kotlin.test.Test
import kotlin.test.assertEquals

class AmaRepositoryMapperTest {
    @Test
    fun mapsAskResponseToAmaAnswer() {
        val response = AskResponse.newBuilder()
            .setAnswer("You worked in Chrome.")
            .setModel("google/gemma-4-31b-it:free")
            .addSources(
                Source.newBuilder()
                    .setTransitionEventId(42)
                    .setDocumentKey("event:42")
                    .setDocumentType("event")
                    .setContent("Application: Google Chrome")
                    .setDistance(0.125)
                    .build(),
            )
            .addArtifacts(
                Artifact.newBuilder()
                    .setAppUsageChart(
                        AppUsageChart.newBuilder()
                            .setStartedAt(Timestamp.newBuilder().setSeconds(1).build())
                            .setEndedAt(Timestamp.newBuilder().setSeconds(3601).build())
                            .setTimezone("UTC")
                            .setPeriodLabel("Yesterday")
                            .setTotalDurationSeconds(3600)
                            .addBuckets(
                                AppUsageBucket.newBuilder()
                                    .setName("Google Chrome")
                                    .setSourceType("browser")
                                    .setDurationSeconds(3600)
                                    .setSessionCount(2)
                                    .setApplicationIdentifier("com.google.Chrome")
                                    .setApplicationPath("/Applications/Google Chrome.app")
                                    .build(),
                            )
                            .build(),
                    )
                    .build(),
            )
            .setSemanticIndexStatus(
                SemanticIndexStatus.newBuilder()
                    .setState(SemanticIndexStatus.State.READY)
                    .setCompletedEventCount(3)
                    .setIndexedEventCount(3)
                    .setPendingEventCount(0)
                    .setMessage("Semantic index is ready.")
                    .build(),
            )
            .build()

        val answer = response.toAmaAnswer()

        assertEquals("You worked in Chrome.", answer.answer)
        assertEquals("google/gemma-4-31b-it:free", answer.model)
        assertEquals(1, answer.sources.size)
        assertEquals(42, answer.sources.first().transitionEventId)
        assertEquals("event:42", answer.sources.first().documentKey)
        assertEquals("event", answer.sources.first().documentType)
        assertEquals("Application: Google Chrome", answer.sources.first().content)
        assertEquals(0.125, answer.sources.first().distance)
        assertEquals(3, answer.indexStatus?.completedEventCount)
        val chart = answer.artifacts.first() as AmaAppUsageChart
        assertEquals("Yesterday", chart.periodLabel)
        assertEquals(1000, chart.startedAtEpochMillis)
        assertEquals(3601000, chart.endedAtEpochMillis)
        assertEquals("UTC", chart.timeZone)
        assertEquals(3600L, chart.totalDurationSeconds)
        assertEquals("Google Chrome", chart.buckets.first().name)
        assertEquals("browser", chart.buckets.first().sourceType)
    }

    @Test
    fun mapsGrpcDeadlineToFriendlyMessage() {
        val error = StatusRuntimeException(
            Status.DEADLINE_EXCEEDED.withDescription("CallOptions deadline exceeded remote_addr=/127.0.0.1:50097"),
        )

        assertEquals(
            "AMA is still catching up. Try again in a moment.",
            error.toAmaErrorMessage(),
        )
    }

    @Test
    fun mapsGrpcUnavailableToFriendlyMessage() {
        val error = StatusRuntimeException(
            Status.UNAVAILABLE.withDescription("connection refused remote_addr=/127.0.0.1:50097"),
        )

        assertEquals(
            "AMA sidecar is unavailable. Please try again in a moment.",
            error.toAmaErrorMessage(),
        )
    }

    @Test
    fun preservesFailedPreconditionProviderMessage() {
        val error = StatusRuntimeException(
            Status.FAILED_PRECONDITION.withDescription("OpenRouter API key is invalid or expired."),
        )

        assertEquals(
            "OpenRouter API key is invalid or expired.",
            error.toAmaErrorMessage(),
        )
    }

    @Test
    fun preservesResourceExhaustedProviderMessage() {
        val error = StatusRuntimeException(
            Status.RESOURCE_EXHAUSTED.withDescription("OpenRouter rate limit exceeded. Try again in 2 minutes."),
        )

        assertEquals(
            "OpenRouter rate limit exceeded. Try again in 2 minutes.",
            error.toAmaErrorMessage(),
        )
    }

    @Test
    fun preservesUnavailableProviderMessage() {
        val error = StatusRuntimeException(
            Status.UNAVAILABLE.withDescription("OpenRouter provider is temporarily unavailable."),
        )

        assertEquals(
            "OpenRouter provider is temporarily unavailable.",
            error.toAmaErrorMessage(),
        )
    }

    @Test
    fun mapsUnsafeInternalSemanticErrorToGenericMessage() {
        val error = StatusRuntimeException(
            Status.INTERNAL.withDescription("answer question: search answer context: embed semantic search query: openrouter request failed"),
        )

        assertEquals(
            "AMA could not answer right now. Please try again in a moment.",
            error.toAmaErrorMessage(),
        )
    }
}
