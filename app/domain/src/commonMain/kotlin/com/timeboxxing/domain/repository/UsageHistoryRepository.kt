package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.UsageDay
import com.timeboxxing.domain.model.UsageEvent
import kotlinx.coroutines.flow.Flow

interface UsageHistoryRepository {
    suspend fun getUsageEvents(day: UsageDay): List<UsageEvent>

    fun watchUsageEvents(day: UsageDay): Flow<UsageEvent>
}
