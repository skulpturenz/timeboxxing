package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexState
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaSource
import com.timeboxxing.domain.repository.AmaRepository

class StaticAmaRepository : AmaRepository {
    override suspend fun ask(question: String, maxSources: Int): AmaAnswer =
        AmaAnswer(
            answer = "I found matching usage history for \"$question\". Connect the sidecar to answer with your real indexed activity.",
            model = "preview",
            sources = listOf(
                AmaSource(
                    transitionEventId = 1,
                    documentKey = "event:1",
                    documentType = "event",
                    startedAtEpochMillis = null,
                    endedAtEpochMillis = null,
                    content = "Application: Google Chrome\nTab: Morgan Ltd. proposal research\nReason: tab_change",
                    distance = 0.12,
                ),
            ),
            indexStatus = getSemanticIndexStatus(),
        )

    override suspend fun getSemanticIndexStatus(): AmaIndexStatus =
        AmaIndexStatus(
            state = AmaIndexState.Ready,
            completedEventCount = 1,
            indexedEventCount = 1,
            pendingEventCount = 0,
            backfillRunning = false,
            message = "Semantic index is ready.",
        )
}

class UnavailableAmaRepository(
    private val message: String,
) : AmaRepository {
    override suspend fun ask(question: String, maxSources: Int): AmaAnswer {
        throw IllegalStateException(message)
    }

    override suspend fun getSemanticIndexStatus(): AmaIndexStatus =
        AmaIndexStatus(
            state = AmaIndexState.Unavailable,
            completedEventCount = 0,
            indexedEventCount = 0,
            pendingEventCount = 0,
            backfillRunning = false,
            message = message,
        )
}
