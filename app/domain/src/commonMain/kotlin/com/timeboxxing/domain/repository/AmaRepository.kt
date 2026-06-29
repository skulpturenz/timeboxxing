package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexStatus

interface AmaRepository {
    suspend fun ask(question: String, maxSources: Int = 5): AmaAnswer
    suspend fun getSemanticIndexStatus(): AmaIndexStatus
}
