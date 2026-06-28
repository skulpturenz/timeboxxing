package com.timeboxxing.app.data

import com.google.protobuf.Timestamp
import com.timeboxxing.app.model.UsageDay
import com.timeboxxing.app.model.UsageEvent
import com.timeboxxing.sidecar.usage.v1.GetUsageEventsRequest
import com.timeboxxing.sidecar.usage.v1.UsageServiceGrpcKt
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.mapNotNull
import java.util.concurrent.TimeUnit

class GrpcUsageHistoryRepository(
    target: String,
    private val channel: ManagedChannel = ManagedChannelBuilder.forTarget(target)
        .usePlaintext()
        .build(),
) : UsageHistoryRepository, AutoCloseable {
    private val stub = UsageServiceGrpcKt.UsageServiceCoroutineStub(channel)

    override suspend fun getUsageEvents(day: UsageDay): List<UsageEvent> {
        val response = stub
            .withDeadlineAfter(5, TimeUnit.SECONDS)
            .getUsageEvents(requestFor(day))

        return response.usageEventsList.mapNotNull { it.toUsageEvent(day) }
    }

    override fun watchUsageEvents(day: UsageDay): Flow<UsageEvent> =
        stub.watchUsageEvents(requestFor(day)).mapNotNull { it.toUsageEvent(day) }

    override fun close() {
        channel.shutdownNow()
        channel.awaitTermination(1, TimeUnit.SECONDS)
    }

    private fun requestFor(day: UsageDay): GetUsageEventsRequest =
        GetUsageEventsRequest.newBuilder()
            .setStartedAt(timestampFromEpochMillis(day.startedAtEpochMillis))
            .setEndedAt(timestampFromEpochMillis(day.endedAtEpochMillis))
            .build()
}

internal fun timestampFromEpochMillis(epochMillis: Long): Timestamp {
    val seconds = Math.floorDiv(epochMillis, 1_000)
    val millis = Math.floorMod(epochMillis, 1_000)
    return Timestamp.newBuilder()
        .setSeconds(seconds)
        .setNanos(millis * 1_000_000)
        .build()
}
