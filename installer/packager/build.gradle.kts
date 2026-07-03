import org.gradle.api.tasks.Sync
import org.gradle.api.GradleException
import org.jetbrains.compose.desktop.application.dsl.TargetFormat

plugins {
    alias(libs.plugins.kotlinJvm)
    alias(libs.plugins.composeMultiplatform)
    alias(libs.plugins.composeCompiler)
}

dependencies {
    implementation(projects.shared)
    implementation(projects.data)
    implementation(projects.domain)

    implementation(compose.animation)
    implementation(compose.desktop.currentOs)
    implementation(compose.materialIconsExtended)
    implementation(platform(libs.koin.bom))
    implementation(libs.koin.compose)
    implementation(libs.koin.compose.viewmodel)
    implementation(libs.koin.core)
    implementation(libs.kotlinx.coroutinesSwing)
    implementation(libs.sentry)

    implementation(libs.compose.uiToolingPreview)
}

val diagnosticsProductionTaskNames = setOf(
    "packageDmg",
    "packageExe",
    "packageDistributionForCurrentOS",
    "createDistributable",
    "runDistributable",
)
fun isDiagnosticsProductionTask(taskName: String): Boolean =
    taskName.substringAfterLast(":") in diagnosticsProductionTaskNames

val requestedTaskNames = gradle.startParameter.taskNames.toList()
val defaultDiagnosticsEnabled = requestedTaskNames.none(::isDiagnosticsProductionTask)
val diagnosticsEnabledProvider = providers.gradleProperty("timeboxxing.diagnostics")
    .map { it.toBooleanStrict() }
    .orElse(defaultDiagnosticsEnabled)

val javaEnvProvider = providers.gradleProperty("timeboxxing.javaEnv")
    .orElse(providers.environmentVariable("JAVA_ENV"))
    .map {
        val normalized = it.trim().lowercase()
        if (normalized !in setOf("production", "development", "test", "local")) {
            throw GradleException("Invalid JAVA_ENV '$it'. Supported values: production, development, test, local.")
        }
        normalized
    }
    .orElse("local")

val generatedBuildConfigDir = layout.buildDirectory.dir("generated/timeboxxingBuildConfig/kotlin")
val generateDesktopBuildConfig by tasks.registering {
    inputs.property("diagnosticsEnabled", diagnosticsEnabledProvider)
    inputs.property("javaEnv", javaEnvProvider)
    outputs.dir(generatedBuildConfigDir)
    doLast {
        val outputFile = generatedBuildConfigDir.get()
            .file("com/timeboxxing/app/DesktopBuildConfig.kt")
            .asFile
        outputFile.parentFile.mkdirs()
        outputFile.writeText(
            """
            package com.timeboxxing.app

            internal object DesktopBuildConfig {
                const val DiagnosticsEnabled: Boolean = ${diagnosticsEnabledProvider.get()}
                const val JavaEnv: String = "${javaEnvProvider.get()}"
            }
            """.trimIndent() + "\n",
        )
    }
}

kotlin {
    sourceSets {
        main {
            kotlin.srcDir(rootProject.layout.projectDirectory.dir("../app/desktopApp/src/main/kotlin"))
            kotlin.srcDir(generatedBuildConfigDir)
        }
    }
}

tasks.named("compileKotlin") {
    dependsOn(generateDesktopBuildConfig)
}

fun resolveGoBinary(): String {
    val home = System.getenv("HOME").orEmpty()
    val candidates = listOfNotNull(
        System.getenv("GO_BINARY")?.takeIf { it.isNotBlank() },
        "$home/.local/share/mise/installs/go/1.26.4/bin/go".takeIf { home.isNotBlank() },
        "/opt/homebrew/bin/go",
        "/usr/local/bin/go",
    )
    return candidates.firstOrNull { file(it).exists() } ?: "go"
}

val sidecarExecutableName = if (System.getProperty("os.name").lowercase().contains("windows")) {
    "timeboxxing-sidecar.exe"
} else {
    "timeboxxing-sidecar"
}

val sidecarDir = rootProject.layout.projectDirectory.dir("../sidecar")
val sidecarOutput = layout.buildDirectory.file("sidecar/$sidecarExecutableName")
val installerResourcesRoot = layout.buildDirectory.dir("generated/installerResources")

val buildSidecar by tasks.registering(Exec::class) {
    workingDir = sidecarDir.asFile
    inputs.dir(sidecarDir)
    outputs.file(sidecarOutput)
    commandLine(resolveGoBinary(), "build", "-tags", "assert", "-o", sidecarOutput.get().asFile.absolutePath, ".")
}

val syncInstallerResources by tasks.registering(Sync::class) {
    dependsOn(buildSidecar)
    into(installerResourcesRoot)
    from(sidecarOutput) {
        into("common/sidecar")
    }
    from(rootProject.layout.projectDirectory.dir("../sidecar/db/sqlite-vector")) {
        into("common/sidecar/sqlite-vector")
    }
}

compose.desktop {
    application {
        mainClass = "com.timeboxxing.app.MainKt"

        nativeDistributions {
            targetFormats(TargetFormat.Dmg, TargetFormat.Exe)
            packageName = "Timeboxxing"
            packageVersion = "1.0.0"
            description = "Timeboxxing desktop app"
            vendor = "Skulpture"
            appResourcesRootDir.set(installerResourcesRoot)

            macOS {
                bundleID = "com.skulpture.timeboxxing"
                packageName = "Timeboxxing"
                dockName = "Timeboxxing"
            }
        }
    }
}

tasks.matching {
    it.name in setOf(
        "createDistributable",
        "runDistributable",
        "packageDistributionForCurrentOS",
        "packageDmg",
        "packageExe",
        "prepareAppResources",
    )
}.configureEach {
    dependsOn(syncInstallerResources)
}
