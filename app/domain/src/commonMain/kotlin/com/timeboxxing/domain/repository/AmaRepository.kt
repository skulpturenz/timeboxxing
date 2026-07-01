package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.AmaAnswer
import com.timeboxxing.domain.model.AmaIndexStatus
import com.timeboxxing.domain.model.AmaStructuredQuery

interface AmaRepository {
    suspend fun ask(question: String, maxSources: Int = 5): AmaAnswer
    suspend fun askStructured(query: AmaStructuredQuery): AmaAnswer
    suspend fun getSemanticIndexStatus(): AmaIndexStatus
}
