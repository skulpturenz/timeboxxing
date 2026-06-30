package com.timeboxxing.data.repository

import com.timeboxxing.domain.model.Project
import com.timeboxxing.domain.repository.ProjectRepository

class StaticProjectRepository(
    initialProjects: List<Project> = emptyList(),
) : ProjectRepository {
    private val projects = initialProjects.toMutableList()

    override suspend fun listProjects(): List<Project> = projects.toList()

    override suspend fun createProject(name: String, colorArgb: Long): Project {
        val trimmedName = name.trim()
        require(trimmedName.isNotEmpty()) { "Project name is required." }
        require(projects.none { it.name.equals(trimmedName, ignoreCase = true) }) {
            "A project with this name already exists."
        }
        val project = Project(
            id = uniqueProjectId(trimmedName, projects.map { it.id }.toSet()),
            name = trimmedName,
            client = "",
            colorArgb = colorArgb,
            hourlyRateCents = 0,
        )
        projects += project
        return project
    }

    override suspend fun deleteProject(projectId: String) {
        projects.removeAll { it.id == projectId }
    }
}

class UnavailableProjectRepository(
    private val message: String,
) : ProjectRepository {
    override suspend fun listProjects(): List<Project> {
        throw IllegalStateException(message)
    }

    override suspend fun createProject(name: String, colorArgb: Long): Project {
        throw IllegalStateException(message)
    }

    override suspend fun deleteProject(projectId: String) {
        throw IllegalStateException(message)
    }
}

private fun uniqueProjectId(
    name: String,
    existingIds: Set<String>,
): String {
    val base = projectSlug(name)
    if (base !in existingIds) return base

    var suffix = 2
    while ("$base-$suffix" in existingIds) {
        suffix += 1
    }
    return "$base-$suffix"
}

private fun projectSlug(name: String): String {
    val slug = name
        .trim()
        .lowercase()
        .replace(Regex("[^a-z0-9]+"), "-")
        .trim('-')
    return slug.ifEmpty { "project" }
}
