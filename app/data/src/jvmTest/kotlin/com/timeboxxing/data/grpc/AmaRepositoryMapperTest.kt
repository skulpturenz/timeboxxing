package com.timeboxxing.data.grpc

import com.google.protobuf.Timestamp
import com.timeboxxing.domain.model.AmaUsageTimeline
import com.timeboxxing.sidecar.ama.v1.Artifact
import com.timeboxxing.sidecar.ama.v1.AskResponse
import com.timeboxxing.sidecar.ama.v1.SemanticIndexStatus
import com.timeboxxing.sidecar.ama.v1.Source
import com.timeboxxing.sidecar.ama.v1.UsageTimeline as UsageTimelineProto
import com.timeboxxing.sidecar.ama.v1.UsageTimelineEvent as UsageTimelineEventProto
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
                    .setUsageTimeline(
                        UsageTimelineProto.newBuilder()
                            .setStartedAt(Timestamp.newBuilder().setSeconds(1).build())
                            .setEndedAt(Timestamp.newBuilder().setSeconds(3601).build())
                            .setTimezone("UTC")
                            .setPeriodLabel("Yesterday")
                            .setTotalDurationSeconds(3600)
                            .setTotalEventCount(1)
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
        val timeline = answer.artifacts.first() as AmaUsageTimeline
        assertEquals("Yesterday", timeline.periodLabel)
        assertEquals(1000, timeline.startedAtEpochMillis)
        assertEquals(3601000, timeline.endedAtEpochMillis)
        assertEquals("UTC", timeline.timeZone)
        assertEquals(3600L, timeline.totalDurationSeconds)
        assertEquals(1, timeline.totalEventCount)
    }

    @Test
    fun mapsUsageTimelineArtifactToAmaAnswer() {
        val event = UsageTimelineEventProto.newBuilder()
            .setTransitionEventId(42)
            .setTitle("Docs")
            .setSourceName("Google Chrome")
            .setSourceType("browser")
            .setStartedAt(Timestamp.newBuilder().setSeconds(10).build())
            .setEndedAt(Timestamp.newBuilder().setSeconds(70).build())
            .setDurationSeconds(60)
            .setApplicationIdentifier("com.google.Chrome")
            .setApplicationPath("/Applications/Google Chrome.app")
            .setUrlHost("example.com")
            .build()
        val response = AskResponse.newBuilder()
            .setAnswer("Usage insight.")
            .setModel("test-model")
            .addArtifacts(
                Artifact.newBuilder()
                    .setUsageTimeline(
                        UsageTimelineProto.newBuilder()
                            .setStartedAt(Timestamp.newBuilder().setSeconds(10).build())
                            .setEndedAt(Timestamp.newBuilder().setSeconds(130).build())
                            .setTimezone("UTC")
                            .setPeriodLabel("Today")
                            .setTotalDurationSeconds(120)
                            .setTotalEventCount(3)
                            .setTruncated(true)
                            .addEvents(event)
                            .build(),
                    )
                    .build(),
            )
            .build()

        val answer = response.toAmaAnswer()

        val timeline = answer.artifacts[0] as AmaUsageTimeline
        assertEquals("Today", timeline.periodLabel)
        assertEquals(10000L, timeline.startedAtEpochMillis)
        assertEquals(130000L, timeline.endedAtEpochMillis)
        assertEquals(120L, timeline.totalDurationSeconds)
        assertEquals(3, timeline.totalEventCount)
        assertEquals(true, timeline.truncated)
        assertEquals(42L, timeline.events.first().transitionEventId)
        assertEquals("example.com", timeline.events.first().urlHost)

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
