package com.timeboxxing.domain.repository

import com.timeboxxing.domain.model.Project

interface ProjectRepository {
    suspend fun listProjects(): List<Project>
    suspend fun createProject(name: String, colorArgb: Long): Project
    suspend fun deleteProject(projectId: String)
}
