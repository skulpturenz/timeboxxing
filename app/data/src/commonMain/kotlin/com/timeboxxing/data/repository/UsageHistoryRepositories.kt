package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import com.timeboxxing.domain.repository.UsageHistoryRepository
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.flow

class StaticUsageHistoryRepository(
    private val events: List<UsageEvent>,
) : UsageHistoryRepository {
    override suspend fun getUsageEvents(day: UsageDay): List<UsageEvent> = events

    override fun watchUsageEvents(day: UsageDay): Flow<UsageEvent> = emptyFlow()
}

class EmptyUsageHistoryRepository : UsageHistoryRepository {
    override suspend fun getUsageEvents(day: UsageDay): List<UsageEvent> = emptyList()

    override fun watchUsageEvents(day: UsageDay): Flow<UsageEvent> = emptyFlow()
}

class UnavailableUsageHistoryRepository(
    private val message: String,
) : UsageHistoryRepository {
    override suspend fun getUsageEvents(day: UsageDay): List<UsageEvent> {
        throw IllegalStateException(message)
    }

    override fun watchUsageEvents(day: UsageDay): Flow<UsageEvent> = flow {
        throw IllegalStateException(message)
    }
}
