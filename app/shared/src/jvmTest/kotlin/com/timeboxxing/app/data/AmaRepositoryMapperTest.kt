package com.timeboxxing.app.data

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
    fun mapsGrpcInternalSemanticErrorToFriendlyMessage() {
        val error = StatusRuntimeException(
            Status.INTERNAL.withDescription("answer question: search answer context: embed semantic search query: openrouter request failed"),
        )

        assertEquals(
            "Semantic model unavailable. Please try again in a moment.",
            error.toAmaErrorMessage(),
        )
    }
}
