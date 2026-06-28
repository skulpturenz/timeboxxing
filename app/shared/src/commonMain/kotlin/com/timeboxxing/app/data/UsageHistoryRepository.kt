package com.timeboxxing.app.data

import com.timeboxxing.app.model.UsageDay
import com.timeboxxing.app.model.UsageEvent
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.flow

interface UsageHistoryRepository {
    suspend fun getUsageEvents(day: UsageDay): List<UsageEvent>

    fun watchUsageEvents(day: UsageDay): Flow<UsageEvent>
}

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
