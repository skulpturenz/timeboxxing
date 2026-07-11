package com.timeboxxing.data.grpc

import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.repository.ProjectRepository
import com.timeboxxing.sidecar.projects.v1.CreateProjectRequest
import com.timeboxxing.sidecar.projects.v1.DeleteProjectRequest
import com.timeboxxing.sidecar.projects.v1.ListProjectsRequest
import com.timeboxxing.sidecar.projects.v1.Project as ProjectProto
import com.timeboxxing.sidecar.projects.v1.ProjectsServiceGrpcKt
import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import io.grpc.Status
import io.grpc.StatusException
import io.grpc.StatusRuntimeException
import java.util.concurrent.TimeUnit

class GrpcProjectRepository(
    target: String,
    private val channel: ManagedChannel = ManagedChannelBuilder.forTarget(target)
        .usePlaintext()
        .build(),
) : ProjectRepository, AutoCloseable {
    private val stub = ProjectsServiceGrpcKt.ProjectsServiceCoroutineStub(channel)

    override suspend fun listProjects(): List<Project> {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listProjects(ListProjectsRequest.getDefaultInstance())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        }

        return response.projectsList.map { it.toProject() }
    }

    override suspend fun createProject(name: String, colorArgb: Long): Project {
        val response = try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .createProject(
                    CreateProjectRequest.newBuilder()
                        .setName(name)
                        .setColorArgb(colorArgb)
                        .build(),
                )
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        }

        return response.toProject()
    }

    override suspend fun deleteProject(projectId: String) {
        try {
            stub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .deleteProject(DeleteProjectRequest.newBuilder().setId(projectId.toLongOrNull() ?: 0L).build())
        } catch (error: StatusRuntimeException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        } catch (error: StatusException) {
            throw IllegalStateException(error.toProjectErrorMessage(), error)
        }
    }

    override fun close() {
        channel.shutdownNow()
        channel.awaitTermination(1, TimeUnit.SECONDS)
    }
}

private fun ProjectProto.toProject(): Project =
    Project(
        id = id.toString(),
        name = name,
        client = client,
        colorArgb = colorArgb,
        hourlyRateCents = hourlyRateCents.toInt(),
    )

internal fun StatusRuntimeException.toProjectErrorMessage(): String = status.toProjectErrorMessage()

internal fun StatusException.toProjectErrorMessage(): String = status.toProjectErrorMessage()

private fun Status.toProjectErrorMessage(): String =
    when (code) {
        Status.Code.UNAVAILABLE -> "Project sidecar is unavailable. Please try again in a moment."
        Status.Code.INVALID_ARGUMENT -> description ?: "Project details are invalid."
        Status.Code.ALREADY_EXISTS -> description ?: "A project with this name already exists."
        else -> "Projects could not be saved right now. Please try again in a moment."
    }
